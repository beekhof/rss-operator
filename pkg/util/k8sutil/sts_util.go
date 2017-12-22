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

package k8sutil

import (
	apps "k8s.io/api/apps/v1beta2"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetStatefulSet(cli kubernetes.Interface, namespace string, setName string) (*apps.StatefulSet, error) {
	return cli.AppsV1beta2().StatefulSets(namespace).Get(setName, metav1.GetOptions{})
}

func GetPodsForStatefulSet(cli kubernetes.Interface, sts *apps.StatefulSet) (*api.PodList, error) {
	// TODO: Do we want the statefulset to fight with RCs? check parent statefulset annotation, or name prefix?
	sel, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	if err != nil {
		return &api.PodList{}, err
	}
	return cli.CoreV1().Pods(sts.GetNamespace()).List(metav1.ListOptions{LabelSelector: sel.String()})
}
