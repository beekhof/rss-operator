// Copyright 2017 The etcd-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"time"

	api "github.com/coreos/etcd-operator/pkg/apis/galera/v1alpha1"
	"github.com/coreos/etcd-operator/pkg/util/probe"

	"k8s.io/apimachinery/pkg/fields"
	kwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// TODO: get rid of this once we use workqueue
var pt *panicTimer

func init() {
	pt = newPanicTimer(time.Minute, "unexpected long blocking (> 1 Minute) when handling cluster event")
}

func (c *Controller) Start() error {
	// TODO: get rid of this init code. CRD and storage class will be managed outside of operator.
	for {
		err := c.initResource()
		if err == nil {
			break
		}
		c.logger.Errorf("initialization failed: %v", err)
		c.logger.Infof("retry in %v...", initRetryWaitTime)
		time.Sleep(initRetryWaitTime)
	}

	probe.SetReady()
	c.run()
	panic("unreachable")
}

func (c *Controller) run() {
	source := cache.NewListWatchFromClient(
		c.Config.EtcdCRCli.GaleraV1alpha1().RESTClient(),
		api.GaleraClusterResourcePlural,
		c.Config.Namespace,
		fields.Everything())

	_, informer := cache.NewIndexerInformer(source, &api.GaleraCluster{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAddEtcdClus,
		UpdateFunc: c.onUpdateEtcdClus,
		DeleteFunc: c.onDeleteEtcdClus,
	}, cache.Indexers{})

	ctx := context.TODO()
	// TODO: use workqueue to avoid blocking
	informer.Run(ctx.Done())
}

func (c *Controller) initResource() error {
	if c.Config.CreateCRD {
		err := c.initCRD()
		if err != nil {
			return fmt.Errorf("fail to init CRD: %v", err)
		}
	}
	return nil
}

func (c *Controller) onAddEtcdClus(obj interface{}) {
	c.logger.Errorf("beekhof: Creating a new cluster", err)
	c.syncEtcdClus(obj.(*api.GaleraCluster))
}

func (c *Controller) onUpdateEtcdClus(oldObj, newObj interface{}) {
	c.syncEtcdClus(newObj.(*api.GaleraCluster))
}

func (c *Controller) onDeleteEtcdClus(obj interface{}) {
	clus, ok := obj.(*api.GaleraCluster)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			panic(fmt.Sprintf("unknown object from GaleraCluster delete event: %#v", obj))
		}
		clus, ok = tombstone.Obj.(*api.GaleraCluster)
		if !ok {
			panic(fmt.Sprintf("Tombstone contained object that is not an GaleraCluster: %#v", obj))
		}
	}
	ev := &Event{
		Type:   kwatch.Deleted,
		Object: clus,
	}

	pt.start()
	err := c.handleClusterEvent(ev)
	if err != nil {
		c.logger.Warningf("fail to handle event: %v", err)
	}
	pt.stop()
}

func (c *Controller) syncEtcdClus(clus *api.GaleraCluster) {
	ev := &Event{
		Type:   kwatch.Added,
		Object: clus,
	}
	// re-watch or restart could give ADD event.
	// If for an ADD event the cluster spec is invalid then it is not added to the local cache
	// so modifying that cluster will result in another ADD event
	if _, ok := c.clusters[clus.Name]; ok {
		ev.Type = kwatch.Modified
	}

	pt.start()
	err := c.handleClusterEvent(ev)
	if err != nil {
		c.logger.Warningf("fail to handle event: %v", err)
	}
	pt.stop()
}
