// Copyright 2016 The etcd-operator Authors
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
	"fmt"
	"time"

	api "github.com/beekhof/rss-operator/pkg/apis/clusterlabs/v1alpha1"
	"github.com/beekhof/rss-operator/pkg/cluster"
	"github.com/beekhof/rss-operator/pkg/generated/clientset/versioned"
	"github.com/beekhof/rss-operator/pkg/util"
	"github.com/beekhof/rss-operator/pkg/util/k8sutil"

	"github.com/sirupsen/logrus"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kwatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

var initRetryWaitTime = 30 * time.Second

type Event struct {
	Type   kwatch.EventType
	Object *api.ReplicatedStatefulSet
}

type Controller struct {
	logger *util.RssLogger
	Config

	clusters map[string]*cluster.Cluster
}

type Config struct {
	Namespace      string
	ServiceAccount string
	KubeCli        kubernetes.Interface
	KubeExtCli     apiextensionsclient.Interface
	EtcdCRCli      versioned.Interface
	CreateCRD      bool
}

func New(cfg Config) *Controller {
	return &Controller{
		logger: util.GetLogger("controller"),

		Config:   cfg,
		clusters: make(map[string]*cluster.Cluster),
	}
}

func (c *Controller) handleClusterEvent(event *Event) error {
	clus := event.Object

	if clus.Status.IsFailed() {
		clustersFailed.Inc()
		if event.Type == kwatch.Deleted {
			delete(c.clusters, clus.Name)
			return nil
		}
		return fmt.Errorf("Ignoring failed cluster (%s). Please delete its CR", clus.Name)
	}

	// Sets detaults
	clus.Spec.Cleanup()

	if err := clus.Validate(); err != nil {
		logrus.Error("Bad RSS object", clus)
		return fmt.Errorf("Invalid cluster spec: %v", err)
	}

	switch event.Type {
	case kwatch.Added:
		if _, ok := c.clusters[clus.Name]; ok {
			return fmt.Errorf("Unsafe state. Cluster (%s) was created before but we received event (%s)", clus.Name, event.Type)
		}

		nc := cluster.New(c.makeClusterConfig(), clus)

		c.clusters[clus.Name] = nc

		clustersCreated.Inc()
		clustersTotal.Inc()

	case kwatch.Modified:
		if _, ok := c.clusters[clus.Name]; !ok {
			return fmt.Errorf("Unsafe state. Cluster (%s) was never created but we received event (%s)", clus.Name, event.Type)
		}
		c.clusters[clus.Name].Update(clus)
		clustersModified.Inc()

	case kwatch.Deleted:
		if _, ok := c.clusters[clus.Name]; !ok {
			return fmt.Errorf("Unsafe state. Cluster (%s) was never created but we received event (%s)", clus.Name, event.Type)
		}
		c.clusters[clus.Name].Delete()
		delete(c.clusters, clus.Name)
		clustersDeleted.Inc()
		clustersTotal.Dec()
	}
	return nil
}

func (c *Controller) makeClusterConfig() cluster.Config {
	return cluster.Config{
		ServiceAccount: c.Config.ServiceAccount,
		KubeCli:        c.Config.KubeCli,
		EtcdCRCli:      c.Config.EtcdCRCli,
	}
}

func (c *Controller) initCRD() error {
	err := k8sutil.CreateCRD(c.KubeExtCli, api.ReplicatedStatefulSetCRDName, api.ReplicatedStatefulSetResourceKind, api.ReplicatedStatefulSetResourcePlural, api.ReplicatedStatefulSetResourceShort)
	if err != nil {
		return fmt.Errorf("Failed to create CRD: %v", err)
	}
	return k8sutil.WaitCRDReady(c.KubeExtCli, api.ReplicatedStatefulSetCRDName)
}
