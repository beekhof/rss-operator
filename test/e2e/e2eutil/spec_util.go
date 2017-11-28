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

// NewS3Backup creates a GaleraBackup object using clusterName.
func NewS3Backup(clusterName, bucket, secret string) *api.GaleraBackup {
	return &api.GaleraBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.GaleraBackupResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: clusterName,
		},
		Spec: api.BackupSpec{
			ClusterName: clusterName,
			StorageType: api.BackupStorageTypeS3,
			BackupStorageSource: api.BackupStorageSource{
				S3: &api.S3Source{
					S3Bucket:  bucket,
					AWSSecret: secret,
				},
			},
		},
	}
}

// NewS3RestoreSource returns an S3RestoreSource with the specified path and secret
func NewS3RestoreSource(path, awsSecret string) *api.S3RestoreSource {
	return &api.S3RestoreSource{
		Path:      path,
		AWSSecret: awsSecret,
	}
}

// NewGaleraRestore returns an GaleraRestore CR with the specified RestoreSource
func NewGaleraRestore(restoreName, version string, size int, restoreSource api.RestoreSource) *api.GaleraRestore {
	return &api.GaleraRestore{
		TypeMeta: metav1.TypeMeta{
			Kind:       api.GaleraRestoreResourceKind,
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: restoreName,
		},
		Spec: api.RestoreSpec{
			ClusterSpec: api.ClusterSpec{
				BaseImage: "gcr.io/etcd-development/etcd",
				Size:      size,
				Version:   version,
			},
			RestoreSource: restoreSource,
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
