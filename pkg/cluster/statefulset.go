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
	"fmt"
	"path"
	"strings"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	api "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	//"github.com/beekhof/galera-operator/pkg/util/k8sutil"
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

const (
	//defaultRetention     = "24h"
	//configMapsFilename   = "configmaps.json"
	governingServiceName = "prometheus-operated"
	DefaultVersion       = "v2.0.0"
	configFilename       = "prometheus.yaml"
	prometheusConfDir    = "/etc/prometheus/config"
	prometheusConfFile   = prometheusConfDir + "/" + configFilename
	prometheusStorageDir = "/var/prometheus/data"
	prometheusRulesDir   = "/etc/prometheus/rules"
	prometheusSecretsDir = "/etc/prometheus/secrets/"
)

var (
	minSize = 1
	//managedByOperatorLabel      = "managed-by"
	//managedByOperatorLabelValue = "prometheus-operator"
	//managedByOperatorLabels     = map[string]string{
	//	managedByOperatorLabel: managedByOperatorLabelValue,
	//}
	probeTimeoutSeconds int32 = 3

	CompatibilityMatrix = []string{
		"v1.4.0",
		"v1.4.1",
		"v1.5.0",
		"v1.5.1",
		"v1.5.2",
		"v1.5.3",
		"v1.6.0",
		"v1.6.1",
		"v1.6.2",
		"v1.6.3",
		"v1.7.0",
		"v1.7.1",
		"v1.7.2",
		"v1.8.0",
		"v2.0.0",
	}
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

func makeStatefulSet(p api.GaleraCluster, old *v1beta1.StatefulSet, config *Config, ruleConfigMaps []*v1.ConfigMap) (*v1beta1.StatefulSet, error) {
	// TODO(fabxc): is this the right point to inject defaults?
	// Ideally we would do it before storing but that's currently not possible.
	// Potentially an update handler on first insertion.

	if p.Spec.BaseImage == "" {
		p.Spec.BaseImage = api.DefaultBaseImage
	}
	if p.Spec.Version == "" {
		p.Spec.Version = api.DefaultVersion
	}
	if p.Spec.Size == 0 {
		p.Spec.Size = minSize
	}
	if p.Spec.Size != 0 && p.Spec.Size < 0 {
		p.Spec.Size = 0
	}

	if p.Spec.Resources.Requests == nil {
		p.Spec.Resources.Requests = v1.ResourceList{}
	}
	if _, ok := p.Spec.Resources.Requests[v1.ResourceMemory]; !ok {
		p.Spec.Resources.Requests[v1.ResourceMemory] = resource.MustParse("2Gi")
	}

	spec, err := makeStatefulSetSpec(p, config, ruleConfigMaps)
	if err != nil {
		return nil, errors.Wrap(err, "make StatefulSet spec")
	}

	boolTrue := true
	statefulset := &v1beta1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      mergeLabels(p.Spec.Pod.Labels, p.ObjectMeta.Labels),
			Name:        prefixedName(p.Name),
			Annotations: p.ObjectMeta.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         p.APIVersion,
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               p.Kind,
					Name:               p.Name,
					UID:                p.UID,
				},
			},
		},
		Spec: *spec,
	}

	if p.Spec.ImagePullSecrets != nil && len(p.Spec.ImagePullSecrets) > 0 {
		statefulset.Spec.Template.Spec.ImagePullSecrets = p.Spec.ImagePullSecrets
	}

	if p.Spec.VolumeClaimTemplate == nil {
		statefulset.Spec.Template.Spec.Volumes = append(statefulset.Spec.Template.Spec.Volumes, v1.Volume{
			Name: volumeName(p.Name),
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	} else {
		pvcTemplate := p.Spec.VolumeClaimTemplate
		pvcTemplate.Name = volumeName(p.Name)
		pvcTemplate.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
		pvcTemplate.Spec.Resources = p.Spec.VolumeClaimTemplate.Spec.Resources
		pvcTemplate.Spec.Selector = p.Spec.VolumeClaimTemplate.Spec.Selector
		statefulset.Spec.VolumeClaimTemplates = append(statefulset.Spec.VolumeClaimTemplates, *pvcTemplate)
	}

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

func makeStatefulSetService(p *api.GaleraCluster, config Config) *v1.Service {
	labels := map[string]string{
		"operated-prometheus": "true",
	}

	if p.Spec.Pod != nil && p.Spec.Pod.Labels != nil {
		labels = mergeLabels(p.Spec.Pod.Labels, labels)
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   governingServiceName,
			Labels: labels,
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "None",
			Ports: []v1.ServicePort{
				{
					Name:       "web",
					Port:       9090,
					TargetPort: intstr.FromString("web"),
				},
			},
			Selector: map[string]string{
				"app": "prometheus",
			},
		},
	}
	return svc
}

func makeStatefulSetSpec(p api.GaleraCluster, c *Config, ruleConfigMaps []*v1.ConfigMap) (*v1beta1.StatefulSetSpec, error) {
	// Prometheus may take quite long to shut down to checkpoint existing data.
	// Allow up to 10 minutes for clean termination.
	terminationGracePeriod := int64(600)

	versionStr := strings.TrimLeft(p.Spec.Version, "v")

	version, err := semver.Parse(versionStr)
	if err != nil {
		return nil, errors.Wrap(err, "parse version")
	}

	var promArgs []string
	var securityContext v1.PodSecurityContext

	switch version.Major {
	case 1:
		promArgs = append(promArgs,
			// "-storage.local.retention="+p.Spec.Retention,
			"-storage.local.num-fingerprint-mutexes=4096",
			fmt.Sprintf("-storage.local.path=%s", prometheusStorageDir),
			"-storage.local.chunk-encoding-version=2",
			fmt.Sprintf("-config.file=%s", prometheusConfFile))
		// We attempt to specify decent storage tuning flags based on how much the
		// requested memory can fit. The user has to specify an appropriate buffering
		// in memory limits to catch increased memory usage during query bursts.
		// More info: https://prometheus.io/docs/operating/storage/.
		reqMem := p.Spec.Resources.Requests[v1.ResourceMemory]

		if version.Minor < 6 {
			// 1024 byte is the fixed chunk size. With increasing number of chunks actually
			// in memory, overhead owed to their management, higher ingestion buffers, etc.
			// increases.
			// We are conservative for now an assume this to be 80% as the Kubernetes environment
			// generally has a very high time series churn.
			memChunks := reqMem.Value() / 1024 / 5

			promArgs = append(promArgs,
				fmt.Sprintf("-storage.local.memory-chunks=%d", memChunks),
				fmt.Sprintf("-storage.local.max-chunks-to-persist=%d", memChunks/2),
			)
		}

		securityContext = v1.PodSecurityContext{}
	default:
		return nil, errors.Errorf("unsupported Prometheus major version %s", version)
	}

	// promArgs = append(promArgs, "-web.external-url="+p.Spec.ExternalURL)

	volumes := []v1.Volume{
		{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: configSecretName(p.Name),
				},
			},
		},
		{
			Name: "rules",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}

	promVolumeMounts := []v1.VolumeMount{
		{
			Name:      "config",
			ReadOnly:  true,
			MountPath: prometheusConfDir,
		},
		{
			Name:      "rules",
			ReadOnly:  true,
			MountPath: prometheusRulesDir,
		},
		{
			Name:      volumeName(p.Name),
			MountPath: prometheusStorageDir,
			SubPath:   subPathForStorage(p.Spec.VolumeClaimTemplate),
		},
	}

	for _, s := range p.Spec.Secrets {
		volumes = append(volumes, v1.Volume{
			Name: "secret-" + s,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: s,
				},
			},
		})
		promVolumeMounts = append(promVolumeMounts, v1.VolumeMount{
			Name:      "secret-" + s,
			ReadOnly:  true,
			MountPath: prometheusSecretsDir + s,
		})
	}

	webRoutePrefix := "/"
	var livenessProbeHandler v1.Handler
	var readinessProbeHandler v1.Handler
	var livenessProbeInitialDelaySeconds int32
	livenessProbeHandler = v1.Handler{
		HTTPGet: &v1.HTTPGetAction{
			Path: path.Clean(webRoutePrefix + "/-/healthy"),
			Port: intstr.FromString("web"),
		},
	}
	readinessProbeHandler = v1.Handler{
		HTTPGet: &v1.HTTPGetAction{
			Path: path.Clean(webRoutePrefix + "/-/ready"),
			Port: intstr.FromString("web"),
		},
	}
	livenessProbeInitialDelaySeconds = 30

	podAnnotations := map[string]string{}
	podLabels := map[string]string{}
	if p.ObjectMeta.Labels != nil {
		for k, v := range p.ObjectMeta.Labels {
			podLabels[k] = v
		}
	}
	if p.ObjectMeta.Annotations != nil {
		for k, v := range p.ObjectMeta.Annotations {
			podAnnotations[k] = v
		}
	}
	podLabels["app"] = "prometheus"
	podLabels["prometheus"] = p.Name
	intSize := int32(p.Spec.Size)
	return &v1beta1.StatefulSetSpec{
		ServiceName:         governingServiceName,
		Replicas:            &intSize,
		PodManagementPolicy: v1beta1.ParallelPodManagement,
		UpdateStrategy: v1beta1.StatefulSetUpdateStrategy{
			Type: v1beta1.RollingUpdateStatefulSetStrategyType,
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      mergeLabels(p.Spec.Pod.Labels, podLabels),
				Annotations: podAnnotations,
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "prometheus",
						Image: fmt.Sprintf("%s:%s", p.Spec.BaseImage, p.Spec.Version),
						Ports: []v1.ContainerPort{
							{
								Name:          "web",
								ContainerPort: 9090,
								Protocol:      v1.ProtocolTCP,
							},
						},
						Args:         promArgs,
						VolumeMounts: promVolumeMounts,
						LivenessProbe: &v1.Probe{
							Handler:             livenessProbeHandler,
							InitialDelaySeconds: livenessProbeInitialDelaySeconds,
							PeriodSeconds:       5,
							TimeoutSeconds:      probeTimeoutSeconds,
							FailureThreshold:    10,
						},
						ReadinessProbe: &v1.Probe{
							Handler:          readinessProbeHandler,
							TimeoutSeconds:   probeTimeoutSeconds,
							PeriodSeconds:    5,
							FailureThreshold: 6,
						},
						Resources: p.Spec.Resources,
					},
				},
				SecurityContext:               &securityContext,
				ServiceAccountName:            p.Spec.ServiceAccountName,
				NodeSelector:                  p.Spec.NodeSelector,
				TerminationGracePeriodSeconds: &terminationGracePeriod,
				Volumes:     volumes,
				Tolerations: p.Spec.Tolerations,
				Affinity:    p.Spec.Affinity,
			},
		},
	}, nil
}

func configSecretName(name string) string {
	return prefixedName(name)
}

func volumeName(name string) string {
	return fmt.Sprintf("%s-db", prefixedName(name))
}

func prefixedName(name string) string {
	return fmt.Sprintf("prometheus-%s", name)
}

func subPathForStorage(s *v1.PersistentVolumeClaim) string {
	if s == nil {
		return ""
	}

	return "prometheus-db"
}
