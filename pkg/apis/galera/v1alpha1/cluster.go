// Copyright 2016 The galera-operator Authors
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

import (
	"errors"
	"fmt"
	"strings"
	"time"

	// "github.com/golang/glog"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	// TODO: move validation code into separate package.
	ErrBackupUnsetRestoreSet = errors.New("spec: backup policy must be set if restore policy is set")
	minClusterSize           = 3
)

const (
	StatusCommandKey    = "status"
	SequenceCommandKey  = "sequence"
	StopCommandKey      = "stop"
	SeedCommandKey      = "seed"
	PrimaryCommandKey   = "primary"
	SecondaryCommandKey = "secondary"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReplicatedStatefulSetList is a list of galera clusters.
type ReplicatedStatefulSetList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReplicatedStatefulSet `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ReplicatedStatefulSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

type ReplicationCommand struct {
	Timeout *string  `json:"timeout,omitempty"`
	Command []string `json:"command"`
}

type ServicePolicy struct {
	ServiceName     string             `json:"serviceName,omitempty"`
	ServicePorts    []v1.ServicePort   `json:"servicePorts,omitempty"`
	ExternalIPs     []string           `json:"externalIPs,omitempty"`
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity,omitempty"`
}

type PodPolicy struct {
	// NodeSelector specifies a map of key-value pairs. For the pod to be eligible
	// to run on a node, the node must have each of the indicated key-value pairs as
	// labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// AntiAffinity determines if the galera-operator tries to avoid putting
	// the galera members in the same cluster onto the same node.
	AntiAffinity bool `json:"antiAffinity"`

	// By default, kubernetes will mount a service account token into the galera pods.
	// AutomountServiceAccountToken indicates whether pods running with the service account should have an API token automatically mounted.
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty"`

	Containers           []v1.Container             `json:"containers"`
	Volumes              []v1.Volume                `json:"volumes,omitempty"`
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	Commands map[string]ReplicationCommand `json:"commands"`
	//./vendor/google.golang.org/genproto/googleapis/devtools/cloudbuild/v1/cloudbuild.pb.go:939:	FileHashes map[string]*FileHashes `protobuf:"bytes,4,rep,name=file_hashes,json=fileHashes" json:"file_hashes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

type ClusterSpec struct {
	// Size is the expected size of the galera cluster.
	// The galera-operator will eventually make the size of the running
	// cluster equal to the expected size.
	Replicas          *int           `json:"replicas"`
	Primaries         *int           `json:"primaries,omitempty"`
	ReconcileInterval *time.Duration `json:"reconcileInterval,omitempty"`

	// An optional list of references to secrets in the same namespace
	// to use for pulling prometheus and alertmanager images from registries
	// see http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Paused is to pause the control of the operator for the galera cluster.
	Paused     bool `json:"paused,omitempty"`
	ChaosLevel *int `json:"chaosLevel"`

	// Pod defines the policy to create pod for the galera pod.
	//
	// Updating Pod does not take effect on any existing galera pods.
	Pod     PodPolicy      `json:"pod"`
	Service *ServicePolicy `json:"service,omitempty"`

	// galera cluster TLS configuration
	TLS *TLSPolicy `json:"TLS,omitempty"`

	// Storage spec to specify how storage shall be used.
	//Storage *StorageSpec `json:"storage,omitempty"`

	// A selector to select which ConfigMaps to mount for loading rule files from.
	RuleSelector *metav1.LabelSelector `json:"ruleSelector,omitempty"`

	// Define resources requests and limits for single Pods.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// Define which Nodes the Pods are scheduled on.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run the
	// Prometheus Pods.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Secrets is a list of Secrets in the same namespace as the Prometheus
	// object, which shall be mounted into the Prometheus Pods.
	// The Secrets are mounted into /etc/prometheus/secrets/<secret-name>.
	// Secrets changes after initial creation of a Prometheus object are not
	// reflected in the running Pods. To change the secrets mounted into the
	// Prometheus Pods, the object must be deleted and recreated with the new list
	// of secrets.
	Secrets []string `json:"secrets,omitempty"`

	// If specified, the pod's scheduling constraints.
	Affinity *v1.Affinity `json:"affinity,omitempty"`

	// If specified, the pod's tolerations.
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

func (rss *ReplicatedStatefulSet) AsOwner() metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       ReplicatedStatefulSetResourceKind,
		Name:       rss.Name,
		UID:        rss.UID,
		Controller: &trueVar,
	}
}

func (rss *ReplicatedStatefulSet) ServiceName(internal bool) string {
	var name string
	if rss.Spec.Service.ServiceName != "" {
		name = rss.Spec.Service.ServiceName
	} else {
		name = fmt.Sprintf("%s-svc", rss.Name)
	}
	if internal {
		name = fmt.Sprintf("%s-int", name)
	}
	return name
}

func (rss *ReplicatedStatefulSet) Validate() error {
	if rss.Spec.TLS != nil {
		if err := rss.Spec.TLS.Validate(); err != nil {
			return err
		}
	}

	if rss.Spec.Pod.Commands == nil {
		return fmt.Errorf("Validate: no cluster commands specified")
	}

	for _, key := range []string{SequenceCommandKey, StopCommandKey, PrimaryCommandKey} {
		if _, ok := rss.Spec.Pod.Commands[key]; !ok {
			return fmt.Errorf("Validate: no %v command specified", key)
		}
	}
	//	glog.Error("Validating build with updated PodSpec")

	for k := range rss.Labels {
		if k == "app" {
			return errors.New(fmt.Sprintf("Validate: definition contains reserved label: %v=%v", k, rss.Labels[k]))
		} else if strings.HasPrefix(k, "rss") {
			return errors.New(fmt.Sprintf("Validate: definition contains reserved value: %v=%v", k, rss.Labels[k]))
		}
	}

	if len(rss.Spec.Pod.Containers) < 1 {
		return errors.New(fmt.Sprintf("Validate: No containers configured for: %v", rss.Name))

	}
	for n, c := range rss.Spec.Pod.Containers {
		if c.Image == "" {
			return errors.New(fmt.Sprintf("Validate: No image configured for container[%v]: %v", n, c.Name))
		}
	}
	return nil
}

func (c *ClusterSpec) GetServicePorts() []v1.ServicePort {
	if c.Service.ServicePorts != nil {
		return c.Service.ServicePorts
	}

	return []v1.ServicePort{
		{
			Name:       "web",
			Port:       9090,
			TargetPort: intstr.FromString("web"),
		},
	}
}

func (c *ClusterSpec) GetNumReplicas() int {
	if c.Replicas == nil {
		return minClusterSize

	} else if *c.Replicas < 0 {
		// Treat as stopped
		return 0

	} else if *c.Replicas > 0 && *c.Replicas < minClusterSize {
		return minClusterSize
	}
	return *c.Replicas
}

func (c *ClusterSpec) GetNumPrimaries() int {
	replicas := c.GetNumReplicas()
	if c.Primaries == nil {
		return replicas

	} else if *c.Primaries < 0 {
		// Treat as stopped
		return 1

	} else if *c.Primaries > replicas {
		return replicas
	}
	return *c.Primaries
}

// Cleanup cleans up user passed spec, e.g. defaulting, transforming fields.
// TODO: move this to admission controller
func (c *ClusterSpec) Cleanup() {

	if c.ReconcileInterval == nil {
		intVal := 60 * time.Second
		c.ReconcileInterval = &intVal
	}
	if c.Resources.Requests == nil {
		c.Resources.Requests = v1.ResourceList{}
	}
	if _, ok := c.Resources.Requests[v1.ResourceMemory]; !ok {
		c.Resources.Requests[v1.ResourceMemory] = resource.MustParse("2M")
	}
}
