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

package e2eutil

import (
	"testing"
	"time"

	api "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	"github.com/beekhof/galera-operator/pkg/generated/clientset/versioned"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"
	"github.com/beekhof/galera-operator/pkg/util/retryutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateCluster(t *testing.T, crClient versioned.Interface, namespace string, cl *api.ReplicatedStatefulSet) (*api.ReplicatedStatefulSet, error) {
	cl.Namespace = namespace
	t.Logf("creating rss cluster")
	res, err := crClient.ClusterlabsV1alpha1().ReplicatedStatefulSets(namespace).Create(cl)
	if err != nil {
		t.Logf("error creating rss cluster: %v", err)
		return nil, err
	}
	t.Logf("created rss cluster: %s", res.Name)

	return res, nil
}

func UpdateCluster(crClient versioned.Interface, cl *api.ReplicatedStatefulSet, maxRetries int, updateFunc k8sutil.ReplicatedStatefulSetCRUpdateFunc) (*api.ReplicatedStatefulSet, error) {
	return AtomicUpdateClusterCR(crClient, cl.Name, cl.Namespace, maxRetries, updateFunc)
}

func AtomicUpdateClusterCR(crClient versioned.Interface, name, namespace string, maxRetries int, updateFunc k8sutil.ReplicatedStatefulSetCRUpdateFunc) (*api.ReplicatedStatefulSet, error) {
	result := &api.ReplicatedStatefulSet{}
	err := retryutil.Retry(1*time.Second, maxRetries, func() (done bool, err error) {
		etcdCluster, err := crClient.ClusterlabsV1alpha1().ReplicatedStatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		updateFunc(etcdCluster)

		result, err = crClient.ClusterlabsV1alpha1().ReplicatedStatefulSets(namespace).Update(etcdCluster)
		if err != nil {
			if apierrors.IsConflict(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	return result, err
}

func DeleteCluster(t *testing.T, crClient versioned.Interface, kubeClient kubernetes.Interface, cl *api.ReplicatedStatefulSet) error {
	t.Logf("deleting rss cluster: %v", cl.Name)
	err := crClient.ClusterlabsV1alpha1().ReplicatedStatefulSets(cl.Namespace).Delete(cl.Name, nil)
	if err != nil {
		return err
	}
	return waitResourcesDeleted(t, kubeClient, cl)
}
