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
	api "github.com/coreos/etcd-operator/pkg/apis/galera/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCluster(genName string, size int) *api.GaleraCluster {
	return &api.GaleraCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.GaleraClusterResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: genName,
		},
		Spec: api.ClusterSpec{
			Size: size,
		},
	}
}


func ClusterWithVersion(cl *api.GaleraCluster, version string) *api.GaleraCluster {
	cl.Spec.Version = version
	return cl
}

func ClusterWithSelfHosted(cl *api.GaleraCluster, sh *api.SelfHostedPolicy) *api.GaleraCluster {
	cl.Spec.SelfHosted = sh
	return cl
}

// NameLabelSelector returns a label selector of the form name=<name>
func NameLabelSelector(name string) map[string]string {
	return map[string]string{"name": name}
}
