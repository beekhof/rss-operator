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

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GaleraBackupList is a list of GaleraBackup.
type GaleraBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []GaleraBackup `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GaleraBackup represents a Kubernetes GaleraBackup Custom Resource.
type GaleraBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              BackupSpec     `json:"spec"`
	Status            BackupCRStatus `json:"status,omitempty"`
}

// BackupSpec contains a backup specification for an etcd cluster.
type BackupSpec struct {
	// ClusterName is the etcd cluster name.
	ClusterName string `json:"clusterName,omitempty"`
	// StorageType is the etcd backup storage type.
	StorageType string `json:"storageType"`
	// BackupStorageSource is the backup storage source.
	BackupStorageSource `json:",inline"`
}

// BackupStorageSource contains the supported backup sources.
type BackupStorageSource struct {
	S3 *S3Source `json:"s3,omitempty"`
}

// BackupCRStatus represents the status of the GaleraBackup Custom Resource.
type BackupCRStatus struct {
	// Succeeded indicates if the backup has Succeeded.
	Succeeded bool `json:"succeeded"`
	// Reason indicates the reason for any backup related failures.
	Reason string `json:"Reason,omitempty"`
	// If S3Source is used to store the backup, this field reports the
	// S3 path where the backup is saved.
	S3Path string `json:"s3Path,omitempty"`
}
