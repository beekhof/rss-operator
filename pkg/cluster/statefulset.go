// Copyright 2016 The prometheus-operator Authors
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

package cluster

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/beekhof/rss-operator/pkg/apis/galera/v1alpha1"
	"github.com/beekhof/rss-operator/pkg/util/constants"
	"github.com/beekhof/rss-operator/pkg/util/k8sutil"

	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	configMapsFilename   = "configmaps.json"
	configFilename       = "rss-config.yaml"
	prometheusSecretsDir = "/etc/rss/secrets/"
)

func mergeLabels(labels map[string]string, otherLabels map[string]string) map[string]string {
	mergedLabels := map[string]string{}

	for key, value := range otherLabels {
		mergedLabels[key] = value
	}

	for key, value := range labels {
		mergedLabels[key] = value
	}
	return mergedLabels
}

func makeStatefulSet(cluster api.ReplicatedStatefulSet, old *v1beta1.StatefulSet, config *Config, ruleConfigMaps []*v1.ConfigMap) (*v1beta1.StatefulSet, error) {
	spec, err := makeStatefulSetSpec(cluster, config, ruleConfigMaps)
	if err != nil {
		return nil, errors.Wrap(err, "make StatefulSet spec")
	}

	statefulset := &v1beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          mergeLabels(k8sutil.LabelsForCluster(cluster.Name), cluster.ObjectMeta.Labels),
			Name:            prefixedName(cluster.Name),
			Annotations:     cluster.ObjectMeta.Annotations,
			OwnerReferences: []metav1.OwnerReference{cluster.AsOwner()},
		},
		Spec: *spec,
	}

	// if cluster.Spec.ImagePullSecrets != nil && len(cluster.Spec.ImagePullSecrets) > 0 {
	// 	statefulset.Spec.Template.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets
	// }

	if old != nil {
		statefulset.Annotations = old.Annotations

		// Updates to statefulset spec for fields other than 'replicas', 'template', and 'updateStrategy' are forbidden.
		statefulset.Spec.PodManagementPolicy = old.Spec.PodManagementPolicy
	}

	return statefulset, nil
}

type ConfigMapReference struct {
	Key      string `json:"key"`
	Checksum string `json:"checksum"`
}

type ConfigMapReferenceList struct {
	Items []*ConfigMapReference `json:"items"`
}

func (l *ConfigMapReferenceList) Len() int {
	return len(l.Items)
}

func (l *ConfigMapReferenceList) Less(i, j int) bool {
	return l.Items[i].Key < l.Items[j].Key
}

func (l *ConfigMapReferenceList) Swap(i, j int) {
	l.Items[i], l.Items[j] = l.Items[j], l.Items[i]
}

func makeStatefulSetService(cluster *api.ReplicatedStatefulSet, config Config, internal bool) *v1.Service {
	var svc *v1.Service
	if internal {
		svc = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   cluster.ServiceName(internal),
				Labels: mergeLabels(cluster.Labels, k8sutil.LabelsForCluster(cluster.Name)),
			},
			Spec: v1.ServiceSpec{
				ClusterIP: "None",
				Ports:     cluster.Spec.GetServicePorts(),
				Selector:  k8sutil.LabelsForCluster(cluster.Name),
				//SessionAffinity: cluster.Spec.Service.SessionAfinity,
			},
		}

	} else {
		ips := []string{}
		if cluster.Spec.Service != nil {
			ips = cluster.Spec.Service.ExternalIPs
		}
		svc = &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   cluster.ServiceName(internal),
				Labels: mergeLabels(cluster.Labels, k8sutil.LabelsForCluster(cluster.Name)),
			},
			Spec: v1.ServiceSpec{
				Type:        "ClusterIP",
				Ports:       cluster.Spec.GetServicePorts(),
				Selector:    k8sutil.LabelsForCluster(cluster.Name),
				ExternalIPs: ips,
				//SessionAffinity: cluster.Spec.Service.SessionAfinity,
			},
		}
	}
	return svc
}

func applyPodSpecPolicy(clusterName string, podSpec *v1.PodSpec, policy *api.PodPolicy) {
	if policy == nil {
		return
	}

	if policy.AntiAffinity {
		ls := &metav1.LabelSelector{MatchLabels: map[string]string{
			"rss_cluster": clusterName,
		}}

		affinity := &v1.Affinity{
			PodAntiAffinity: &v1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					{
						LabelSelector: ls,
						TopologyKey:   "kubernetes.io/hostname",
					},
				},
			},
		}
		podSpec.Affinity = affinity
	}

	if policy.AutomountServiceAccountToken != nil {
		podSpec.AutomountServiceAccountToken = policy.AutomountServiceAccountToken
	}
}

func makeStatefulSetSpec(cluster api.ReplicatedStatefulSet, c *Config, ruleConfigMaps []*v1.ConfigMap) (*v1beta1.StatefulSetSpec, error) {
	// Prometheus may take quite long to shut down to checkpoint existing data.
	// Allow up to 10 minutes for clean termination.
	terminationGracePeriod := int64(600)
	securityContext := v1.PodSecurityContext{}

	// ReadinessProbe: &v1.Probe{
	// 	Handler:          v1.Handler{
	// 		HTTPGet: &v1.HTTPGetAction{
	// 		    Path: path.Clean(webRoutePrefix + "/-/ready"),
	// 		    Port: intstr.FromString("web"),
	// 	    },
	//  },
	// 	TimeoutSeconds:   probeTimeoutSeconds,
	// 	PeriodSeconds:    5,
	// 	FailureThreshold: 6,
	// },

	// We need to be very particular how the spec is built, items need to be
	// fully formed before being added to their parent.
	//
	// For example, if .Env was empty in the cluster.Spec.Containers
	// definiton, we cannot:
	//
	// - assign cluster.Spec.Containers to PodSpec.Containers
	// - cycle through the PodSpec.Containers adding .Env entries
	//
	// or even:
	//
	// - copy cluster.Spec.Containers to 'containers'
	// - cycle through the 'containers' adding .Env entries
	// - assign 'containers' to PodSpec.Containers
	//
	// the only way that works is:
	//
	// - cycle through the cluster.Spec.Containers
	// - adding .Env entries and appending to 'containers'
	// - assign 'containers' to PodSpec.Containers

	intSize := int32(cluster.Spec.GetNumReplicas())
	volumes := cluster.Spec.Pod.Volumes
	var containers []v1.Container

	for _, container := range cluster.Spec.Pod.Containers {
		// Append generated details
		if len(container.Env) == 0 {
			container.Env = []v1.EnvVar{
				{
					Name:  constants.EnvOperatorServiceName,
					Value: cluster.ServiceName(true),
				},
			}

		} else {
			container.Env = append(container.Env, v1.EnvVar{
				Name:  constants.EnvOperatorServiceName,
				Value: cluster.ServiceName(true),
			})
		}
		// The spec author could add themselves though...
		container.Env = append(container.Env, v1.EnvVar{
			Name: constants.EnvOperatorPodName,
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		})

		container.Env = append(container.Env, v1.EnvVar{
			Name: constants.EnvOperatorPodNamespace,
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				},
			},
		})

		if cluster.Spec.ChaosLevel != nil {
			container.Env = append(container.Env, v1.EnvVar{
				Name:  "CHAOS_LEVEL",
				Value: fmt.Sprintf("%v", 20-*cluster.Spec.ChaosLevel),
			})
		}

		// Now add and mount the downward API
		volumes = append(volumes, v1.Volume{
			Name: "podinfo",
			VolumeSource: v1.VolumeSource{
				DownwardAPI: &v1.DownwardAPIVolumeSource{
					Items: []v1.DownwardAPIVolumeFile{
						{
							Path: "labels",
							FieldRef: &v1.ObjectFieldSelector{
								FieldPath: "metadata.labels",
							},
						},
						{
							Path: "annotations",
							FieldRef: &v1.ObjectFieldSelector{
								FieldPath: "metadata.annotations",
							},
						},
					},
				},
			},
		})
		container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
			Name:      "podinfo",
			ReadOnly:  false,
			MountPath: "/etc/podinfo",
			SubPath:   "",
		})

		// Now add and mount any secrets volumes
		for _, s := range cluster.Spec.Secrets {
			volumes = append(volumes, v1.Volume{
				Name: "secret-" + s,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: s,
					},
				},
			})
			container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
				Name:      "secret-" + s,
				ReadOnly:  true,
				MountPath: prometheusSecretsDir + s,
			})
		}

		containers = append(containers, container)
	}

	podSpec := v1.PodSpec{
		Volumes:                       volumes,
		Containers:                    containers,
		ServiceAccountName:            cluster.Spec.ServiceAccountName,
		NodeSelector:                  cluster.Spec.NodeSelector,
		Tolerations:                   cluster.Spec.Tolerations,
		Affinity:                      cluster.Spec.Affinity,
		SecurityContext:               &securityContext,
		TerminationGracePeriodSeconds: &terminationGracePeriod,
	}

	applyPodSpecPolicy(cluster.Name, &podSpec, &cluster.Spec.Pod)

	podAnnotations := map[string]string{}
	if cluster.ObjectMeta.Annotations != nil {
		podAnnotations = cluster.ObjectMeta.Annotations
	}

	return &v1beta1.StatefulSetSpec{
		ServiceName:          cluster.ServiceName(true),
		Replicas:             &intSize,
		PodManagementPolicy:  v1beta1.ParallelPodManagement,
		VolumeClaimTemplates: cluster.Spec.Pod.VolumeClaimTemplates,
		UpdateStrategy: v1beta1.StatefulSetUpdateStrategy{
			Type: v1beta1.OnDeleteStatefulSetStrategyType,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:          mergeLabels(k8sutil.LabelsForCluster(cluster.Name), cluster.ObjectMeta.Labels),
				Annotations:     podAnnotations,
				OwnerReferences: []metav1.OwnerReference{cluster.AsOwner()},
			},
			Spec: podSpec,
		},
	}, nil
}

func configSecretName(name string) string {
	return prefixedName(name)
}

func prefixedName(name string) string {
	return fmt.Sprintf("rss-%s", name)
}

func makeRuleConfigMap(cm *v1.ConfigMap) (*ConfigMapReference, error) {
	keys := []string{}
	for k := range cm.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	m := yaml.MapSlice{}
	for _, k := range keys {
		m = append(m, yaml.MapItem{Key: k, Value: cm.Data[k]})
	}

	b, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}

	return &ConfigMapReference{
		Key:      cm.Namespace + "/" + cm.Name,
		Checksum: fmt.Sprintf("%x", sha256.Sum256(b)),
	}, nil
}

func makeRuleConfigMapListFile(configMaps []*v1.ConfigMap) ([]byte, error) {
	cml := &ConfigMapReferenceList{}

	for _, cm := range configMaps {
		configmap, err := makeRuleConfigMap(cm)
		if err != nil {
			return nil, err
		}
		cml.Items = append(cml.Items, configmap)
	}

	sort.Sort(cml)
	return json.Marshal(cml)
}

func makeConfigSecret(cluster api.ReplicatedStatefulSet, configMaps []*v1.ConfigMap, config Config) (*v1.Secret, error) {
	b, err := makeRuleConfigMapListFile(configMaps)
	if err != nil {
		return nil, err
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            configSecretName(cluster.Name),
			Labels:          mergeLabels(k8sutil.LabelsForCluster(cluster.Name), cluster.ObjectMeta.Labels),
			OwnerReferences: []metav1.OwnerReference{cluster.AsOwner()},
		},
		Data: map[string][]byte{
			configFilename:     {},
			configMapsFilename: b,
		},
	}, nil
}

func makeEmptyConfig(cluster api.ReplicatedStatefulSet, configMaps []*v1.ConfigMap, config Config) (*v1.Secret, error) {
	s, err := makeConfigSecret(cluster, configMaps, config)
	if err != nil {
		return nil, err
	}

	s.ObjectMeta.Annotations = map[string]string{
		"empty": "true",
	}

	return s, nil
}
