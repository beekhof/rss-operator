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
	api "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCluster(genName string, size int, image string, labels map[string]string, annotations map[string]string) *api.ReplicatedStatefulSet {
	if image == "" {
		image = "quay.io/beekhof/dummy:latest"
	}
	return &api.ReplicatedStatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.ReplicatedStatefulSetResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
			GenerateName: genName,
			Labels:       labels,
			Annotations:  annotations,
		},
		Spec: api.ClusterSpec{
			Replicas: size,
			Pod: api.PodPolicy{
				AntiAffinity: true,
			},
			Containers: []v1.Container{
				{
					Image:           image,
					Name:            "test",
					ImagePullPolicy: v1.PullAlways,
				},
			},
			Commands: api.ClusterCommands{
				Status:   []string{"/check.sh"},
				Sequence: []string{"/sequence.sh"},
				Seed:     []string{"/seed.sh"},
				Primary:  []string{"/start.sh"},
				Stop:     []string{"/stop.sh"},
			},
		},
	}
}

func NewClusterWithSpec(genName string, spec api.ClusterSpec, labels map[string]string, annotations map[string]string) *api.ReplicatedStatefulSet {
	return &api.ReplicatedStatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.ReplicatedStatefulSetResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
			GenerateName: genName,
			Labels:       labels,
			Annotations:  annotations,
		},
		Spec: spec,
	}
}

func ClusterWithVersion(cl *api.ReplicatedStatefulSet, version string) *api.ReplicatedStatefulSet {
	// cl.Spec.Version = version
	return cl
}

// NameLabelSelector returns a label selector of the form name=<name>
func NameLabelSelector(name string) map[string]string {
	return map[string]string{"name": name}
}
