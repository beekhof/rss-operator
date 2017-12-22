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

package cluster

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	api "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	"github.com/beekhof/galera-operator/pkg/debug"
	"github.com/beekhof/galera-operator/pkg/generated/clientset/versioned"
	"github.com/beekhof/galera-operator/pkg/util"
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"
	"github.com/beekhof/galera-operator/pkg/util/retryutil"

	// "github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	reconcileInterval = 8 * time.Second
)

type clusterEventType string

const (
	eventModifyCluster clusterEventType = "Modify"
)

type clusterEvent struct {
	typ     clusterEventType
	cluster *api.GaleraCluster
}

type Config struct {
	ServiceAccount string

	KubeCli   kubernetes.Interface
	EtcdCRCli versioned.Interface
}

type Cluster struct {
	logger      *logrus.Entry
	debugLogger *debug.DebugLogger

	config Config

	cluster *api.GaleraCluster
	// in memory state of the cluster
	// status is the source of truth after Cluster struct is materialized.
	status api.ClusterStatus

	eventCh chan *clusterEvent
	stopCh  chan struct{}

	// members repsersents the members in the etcd cluster.
	// the name of the member is the the name of the pod the member
	// process runs in.
	peers etcdutil.MemberSet

	tlsConfig *tls.Config

	eventsCli corev1.EventInterface
}

func New(config Config, cl *api.GaleraCluster) *Cluster {
	lg := logrus.WithField("pkg", "cluster").WithField("cluster-name", cl.Name)
	c := &Cluster{
		logger:      lg,
		debugLogger: debug.New(cl.Name),
		config:      config,
		cluster:     cl,
		eventCh:     make(chan *clusterEvent, 100),
		stopCh:      make(chan struct{}),
		status:      *(cl.Status.DeepCopy()),
		eventsCli:   config.KubeCli.Core().Events(cl.Namespace),
	}

	go func() {
		c.logger.Infof("setting up cluster")
		if err := c.setup(); err != nil {
			c.logger.Errorf("cluster failed to setup: %v", err)
			if c.status.Phase != api.ClusterPhaseFailed {
				c.status.SetReason(err.Error())
				c.status.SetPhase(api.ClusterPhaseFailed)
				if err := c.updateCRStatus(); err != nil {
					c.logger.Errorf("failed to update cluster phase (%v): %v", api.ClusterPhaseFailed, err)
				}
			}
			c.logger.Infof("exiting early %v", c.status.Phase)
			return
		}
		c.logger.Infof("running")
		c.run()
	}()

	return c
}

func (c *Cluster) setup() error {
	var shouldCreateCluster bool
	switch c.status.Phase {
	case api.ClusterPhaseNone:
		shouldCreateCluster = true
	case api.ClusterPhaseCreating:
		return errCreatedCluster
	case api.ClusterPhaseRunning:
		shouldCreateCluster = false

	default:
		return fmt.Errorf("unexpected cluster phase: %s", c.status.Phase)
	}

	if c.isSecureClient() {
		d, err := k8sutil.GetTLSDataFromSecret(c.config.KubeCli, c.cluster.Namespace, c.cluster.Spec.TLS.Static.OperatorSecret)
		if err != nil {
			return err
		}
		c.tlsConfig, err = etcdutil.NewTLSConfig(d.CertData, d.KeyData, d.CAData)
		if err != nil {
			return err
		}
	}

	c.logger.Infof("creating cluster: %v, %v", shouldCreateCluster, c.status.Phase)
	if shouldCreateCluster {
		return c.create()
	}
	return nil
}

func (c *Cluster) createGalera() error {

	// Create governing service if it doesn't exist.
	svcClient := c.config.KubeCli.Core().Services(c.cluster.Namespace)
	if err := k8sutil.CreateOrUpdateService(svcClient, makeStatefulSetService(c.cluster, c.config)); err != nil {
		return errors.Wrap(err, "synchronizing governing service failed")
	}

	ruleFileConfigMaps := []*v1.ConfigMap{}
	//ruleFileConfigMaps, err := c.ruleFileConfigMaps(p)

	// Create Secret if it doesn't exist.
	s, err := makeEmptyConfig(*c.cluster, ruleFileConfigMaps, c.config)
	if err != nil {
		return errors.Wrap(err, "generating empty config secret failed")
	}
	if _, err := c.config.KubeCli.Core().Secrets(c.cluster.Namespace).Create(s); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "creating empty config file failed")
	}

	c.logger.Errorf("beekhof: creating Galera STS")
	sts, err := makeStatefulSet(*c.cluster, nil, &c.config, ruleFileConfigMaps)
	if err != nil {
		return errors.Wrap(err, "creating statefulset definition failed")
	}

	ssetClient := c.config.KubeCli.AppsV1beta1().StatefulSets(c.cluster.Namespace)
	if _, err := ssetClient.Create(sts); err != nil {
		return errors.Wrap(err, "creating statefulset failed")
	}
	c.logger.Errorf("beekhof: created Galera STS in %v", c.cluster.Namespace)
	return nil
}

func (c *Cluster) create() error {
	c.status.SetPhase(api.ClusterPhaseCreating)

	if err := c.updateCRStatus(); err != nil {
		return fmt.Errorf("cluster create: failed to update cluster phase (%v): %v", api.ClusterPhaseCreating, err)
	}
	if err := c.createGalera(); err != nil {
		c.logger.Errorf("beekhof: starting sts failed %v", err)
		return err
	}
	c.logClusterCreation()
	return nil
}

func (c *Cluster) Delete() {
	c.logger.Info("cluster is deleted by user")
	close(c.stopCh)
}

func (c *Cluster) send(ev *clusterEvent) {
	select {
	case c.eventCh <- ev:
		l, ecap := len(c.eventCh), cap(c.eventCh)
		if l > int(float64(ecap)*0.8) {
			c.logger.Warningf("eventCh buffer is almost full [%d/%d]", l, ecap)
		}
	case <-c.stopCh:
	}
}

func (c *Cluster) run() {
	if err := c.setupServices(); err != nil {
		c.logger.Errorf("fail to setup etcd services: %v", err)
	}
	c.status.ServiceName = k8sutil.ClientServiceName(c.cluster.Name)
	c.status.ClientPort = k8sutil.EtcdClientPort

	c.status.SetPhase(api.ClusterPhaseRunning)
	if err := c.updateCRStatus(); err != nil {
		c.logger.Warningf("update initial CR status failed: %v", err)
	}
	c.logger.Infof("start running...")

	var rerr error
	for {
		select {
		case <-c.stopCh:
			return
		case event := <-c.eventCh:
			switch event.typ {
			case eventModifyCluster:
				err := c.handleUpdateEvent(event)
				if err != nil {
					c.logger.Errorf("handle update event failed: %v", err)
					c.status.SetReason(err.Error())
					c.reportFailedStatus()
					return
				}
			default:
				panic("unknown event type" + event.typ)
			}

		case <-time.After(reconcileInterval):
			start := time.Now()

			if c.cluster.Spec.Paused {
				c.status.PauseControl()
				c.logger.Infof("control is paused, skipping reconciliation")
				continue
			} else {
				c.status.Control()
			}

			running, pending, err := c.pollPods()
			if err != nil {
				c.logger.Errorf("fail to poll pods: %v", err)
				reconcileFailed.WithLabelValues("failed to poll pods").Inc()
				continue
			}

			if len(pending) > 0 {
				// Pod startup might take long, e.g. pulling image. It would deterministically become running or succeeded/failed later.
				c.logger.Infof("skip reconciliation: running (%v), pending (%v)", k8sutil.GetPodNames(running), k8sutil.GetPodNames(pending))
				reconcileFailed.WithLabelValues("not all pods are running").Inc()
				continue
			}
			if len(running) == 0 {
				// TODO: how to handle this case?
				c.logger.Warningf("all etcd pods are dead.")
				break
			}

			// On controller restore, we could have "members == nil"
			if rerr != nil || c.peers == nil {
				rerr = c.updateMembers(c.podsToMemberSet(running, c.isSecureClient()))
				if rerr != nil {
					c.logger.Errorf("failed to update members: %v", rerr)
					break
				}
			}
			rerr = c.reconcile(running)
			if rerr != nil {
				c.logger.Errorf("failed to reconcile: %v", rerr)
				break
			}
			c.updateMemberStatus(c.peers, k8sutil.GetPodNames(running))
			if err := c.updateCRStatus(); err != nil {
				c.logger.Warningf("periodic update CR status failed: %v", err)
			}

			rerr = c.replicate()

			reconcileHistogram.WithLabelValues(c.name()).Observe(time.Since(start).Seconds())
		}

		if rerr != nil {
			reconcileFailed.WithLabelValues(rerr.Error()).Inc()
		}

		if isFatalError(rerr) {
			c.status.SetReason(rerr.Error())
			c.logger.Errorf("cluster failed: %v", rerr)
			c.reportFailedStatus()
			return
		}
	}
}

func (c *Cluster) handleUpdateEvent(event *clusterEvent) error {
	oldSpec := c.cluster.Spec.DeepCopy()
	c.cluster = event.cluster

	if isSpecEqual(event.cluster.Spec, *oldSpec) {
		// We have some fields that once created could not be mutated.
		if !reflect.DeepEqual(event.cluster.Spec, *oldSpec) {
			c.logger.Infof("ignoring update event: %#v", event.cluster.Spec)
		}
		return nil
	}
	// TODO: we can't handle another upgrade while an upgrade is in progress

	c.logSpecUpdate(*oldSpec, event.cluster.Spec)
	return nil
}

func isSpecEqual(s1, s2 api.ClusterSpec) bool {
	if s1.Size != s2.Size || s1.Paused != s2.Paused || s1.Version != s2.Version {
		return false
	}
	return true
}

func (c *Cluster) isSecurePeer() bool {
	return c.cluster.Spec.TLS.IsSecurePeer()
}

func (c *Cluster) isSecureClient() bool {
	return c.cluster.Spec.TLS.IsSecureClient()
}

func (c *Cluster) Update(cl *api.GaleraCluster) {
	c.send(&clusterEvent{
		typ:     eventModifyCluster,
		cluster: cl,
	})
}

func (c *Cluster) setupServices() error {
	err := k8sutil.CreateClientService(c.config.KubeCli, c.cluster.Name, c.cluster.Namespace, c.cluster.AsOwner())
	if err != nil {
		return err
	}

	return k8sutil.CreatePeerService(c.config.KubeCli, c.cluster.Name, c.cluster.Namespace, c.cluster.AsOwner())
}

func (c *Cluster) podOwner(pod *v1.Pod) bool {
	sts, err := k8sutil.GetStatefulSet(c.config.KubeCli, c.cluster.Namespace, prefixedName(c.cluster.Name))
	if err != nil {
		c.logger.Errorf("failed to find sts: %v", err)
	}

	for n := range pod.OwnerReferences {
		if pod.OwnerReferences[n].UID == c.cluster.UID {
			return true
		} else if pod.OwnerReferences[n].UID == sts.UID {
			return true
		}
		c.logger.Infof("mismatch[%v/%v]: %v vs. c.%v and s.%v", n, len(pod.OwnerReferences), pod.OwnerReferences[n].UID, c.cluster.UID, sts.UID)
	}
	return false
}

// func (c *Cluster) getStatefulSet() (*apps.StatefulSet, err error) {
// //	Labels:          mergeLabels(p.Spec.PodLabels(), p.ObjectMeta.Labels),
// 	return cli.AppsV1beta2().StatefulSets(c.cluster.Namespace).Get(c.cluster.Name, metav1.GetOptions{})
// }
func (c *Cluster) pollPods() (running, pending []*v1.Pod, err error) {
	// sel, err := metav1.LabelSelectorAsSelector(sts.Spec.Selector)
	// c.logger.Infof("filter: %v -> %v.", sts.Spec.Selector, sel)
	// podList, err := k8sutil.GetPodsForStatefulSet(c.config.KubeCli, sts)

	podList, err := c.config.KubeCli.Core().Pods(c.cluster.Namespace).List(k8sutil.ClusterListOpt(c.cluster.Name))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list running pods: %v", err)
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		if len(pod.OwnerReferences) < 1 {
			c.logger.Warningf("pollPods: ignore pod %v: no owner", pod.Name)
			continue
		}
		if !c.podOwner(pod) {
			c.logger.Warningf("pollPods: ignore pod %v: owner (%v) is not %v",
				pod.Name, pod.OwnerReferences[0].UID, c.cluster.UID)
			continue
		}
		switch pod.Status.Phase {
		case v1.PodRunning:
			running = append(running, pod)
		case v1.PodPending:
			pending = append(pending, pod)
		}
	}

	return running, pending, nil
}

func (c *Cluster) updateMemberStatus(members etcdutil.MemberSet, running []string) {
	var unready []string
	for _, m := range members {
		if !util.PresentIn(m.Name, running) {
			c.logger.Infof("updateMemberStatus:  pod %v: not ready", m.Name)
			unready = append(unready, m.Name)
		}
	}

	c.status.Members.Ready = running
	c.status.Members.Unready = unready
}

func (c *Cluster) updateCRStatus() error {
	if reflect.DeepEqual(c.cluster.Status, c.status) {
		return nil
	}

	newCluster := c.cluster
	newCluster.Status = c.status
	newCluster, err := c.config.EtcdCRCli.GaleraV1alpha1().GaleraClusters(c.cluster.Namespace).Update(c.cluster)
	if err != nil {
		return fmt.Errorf("failed to update CR status: %v", err)
	}

	c.cluster = newCluster

	return nil
}

func (c *Cluster) reportFailedStatus() {
	c.logger.Info("cluster failed. Reporting failed reason...")

	retryInterval := 5 * time.Second
	f := func() (bool, error) {
		c.status.SetPhase(api.ClusterPhaseFailed)
		err := c.updateCRStatus()
		if err == nil || k8sutil.IsKubernetesResourceNotFoundError(err) {
			return true, nil
		}

		if !apierrors.IsConflict(err) {
			c.logger.Warningf("retry report status in %v: fail to update: %v", retryInterval, err)
			return false, nil
		}

		cl, err := c.config.EtcdCRCli.GaleraV1alpha1().GaleraClusters(c.cluster.Namespace).
			Get(c.cluster.Name, metav1.GetOptions{})
		if err != nil {
			// Update (PUT) will return conflict even if object is deleted since we have UID set in object.
			// Because it will check UID first and return something like:
			// "Precondition failed: UID in precondition: 0xc42712c0f0, UID in object meta: ".
			if k8sutil.IsKubernetesResourceNotFoundError(err) {
				return true, nil
			}
			c.logger.Warningf("retry report status in %v: fail to get latest version: %v", retryInterval, err)
			return false, nil
		}
		c.cluster = cl
		return false, nil
	}

	retryutil.Retry(retryInterval, math.MaxInt64, f)
}

func (c *Cluster) name() string {
	return c.cluster.GetName()
}

func (c *Cluster) logClusterCreation() {
	specBytes, err := json.MarshalIndent(c.cluster.Spec, "", "    ")
	if err != nil {
		c.logger.Errorf("failed to marshal cluster spec: %v", err)
	}

	c.logger.Info("creating cluster with Spec:")
	for _, m := range strings.Split(string(specBytes), "\n") {
		c.logger.Info(m)
	}
}

func (c *Cluster) logSpecUpdate(oldSpec, newSpec api.ClusterSpec) {
	oldSpecBytes, err := json.MarshalIndent(oldSpec, "", "    ")
	if err != nil {
		c.logger.Errorf("failed to marshal cluster spec: %v", err)
	}
	newSpecBytes, err := json.MarshalIndent(newSpec, "", "    ")
	if err != nil {
		c.logger.Errorf("failed to marshal cluster spec: %v", err)
	}

	c.logger.Infof("spec update: Old Spec:")
	for _, m := range strings.Split(string(oldSpecBytes), "\n") {
		c.logger.Info(m)
	}

	c.logger.Infof("New Spec:")
	for _, m := range strings.Split(string(newSpecBytes), "\n") {
		c.logger.Info(m)
	}

	if c.isDebugLoggerEnabled() {
		c.debugLogger.LogClusterSpecUpdate(string(oldSpecBytes), string(newSpecBytes))
	}
}

func (c *Cluster) isDebugLoggerEnabled() bool {
	return c.debugLogger != nil
}
