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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"time"

	api "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	"github.com/beekhof/galera-operator/pkg/debug"
	"github.com/beekhof/galera-operator/pkg/generated/clientset/versioned"
	"github.com/beekhof/galera-operator/pkg/util"
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"
	"github.com/beekhof/galera-operator/pkg/util/retryutil"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	// "github.com/pborman/uuid"

	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
)

type clusterEventType string

const (
	eventModifyCluster clusterEventType = "Modify"
)

type clusterEvent struct {
	typ     clusterEventType
	cluster *api.ReplicatedStatefulSet
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

	cluster *api.ReplicatedStatefulSet
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

	// Watch for new config maps to be created so we can mount them into running containers
	cmapInf cache.SharedIndexInformer
	// secrInf cache.SharedIndexInformer
}

func New(config Config, cl *api.ReplicatedStatefulSet) *Cluster {
	lg := util.GetLogger("cluster").WithField("cluster-name", cl.Name)
	lg.Infof("Creating %v/%v", cl.Name, cl.GenerateName)

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

	resyncPeriod := 5 * time.Minute

	c.cmapInf = cache.NewSharedIndexInformer(
		cache.NewListWatchFromClient(config.KubeCli.Core().RESTClient(), "configmaps", cl.Namespace, fields.Everything()),
		&v1.ConfigMap{}, resyncPeriod, cache.Indexers{},
	)
	c.cmapInf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleConfigMapAdd,
		DeleteFunc: c.handleConfigMapDelete,
		UpdateFunc: c.handleConfigMapUpdate,
	})

	// cl.Spec defaults set in makeStatefulSet(), should happen in ClusterSpec.Cleanup()
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

func (c *Cluster) handleConfigMapAdd(obj interface{}) {
	// Recreate and update the ReplicatedStatefulSet
}

func (c *Cluster) handleConfigMapDelete(obj interface{}) {
	// Recreate and update the ReplicatedStatefulSet
}

func (c *Cluster) handleConfigMapUpdate(old, cur interface{}) {
	// Ignore
}

func (c *Cluster) keyFunc(obj interface{}) (string, bool) {
	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		c.logger.Error("msg", "creating key failed", "err", err)
		return k, false
	}
	return k, true
}

func (c *Cluster) ruleFileConfigMaps(cl *api.ReplicatedStatefulSet) ([]*v1.ConfigMap, error) {
	res := []*v1.ConfigMap{}

	ruleSelector, err := metav1.LabelSelectorAsSelector(cl.Spec.RuleSelector)
	if err != nil {
		return nil, err
	}

	cache.ListAllByNamespace(c.cmapInf.GetIndexer(), cl.Namespace, ruleSelector, func(obj interface{}) {
		_, ok := c.keyFunc(obj)
		if ok {
			res = append(res, obj.(*v1.ConfigMap))
		}
	})

	return res, nil
}

func (c *Cluster) create() error {
	c.status.SetPhase(api.ClusterPhaseCreating)

	if err := c.updateCRStatus(); err != nil {
		return fmt.Errorf("cluster create: failed to update cluster phase (%v): %v", api.ClusterPhaseCreating, err)
	}

	// Create governing service if it doesn't exist.
	svcClient := c.config.KubeCli.Core().Services(c.cluster.Namespace)
	if err := k8sutil.CreateOrUpdateService(svcClient, makeStatefulSetService(c.cluster, c.config, true)); err != nil {
		return errors.Wrap(err, "synchronizing internal service failed")
	}
	if len(c.cluster.Spec.Service.ExternalIPs) > 0 {
		if err := k8sutil.CreateOrUpdateService(svcClient, makeStatefulSetService(c.cluster, c.config, false)); err != nil {
			return errors.Wrap(err, "synchronizing external service failed")
		}
	}
	ruleFileConfigMaps, err := c.ruleFileConfigMaps(c.cluster)

	// Create Secret if it doesn't exist.
	s, err := makeEmptyConfig(*c.cluster, ruleFileConfigMaps, c.config)
	if err != nil {
		return errors.Wrap(err, "generating empty config secret failed")
	}
	if _, err := c.config.KubeCli.Core().Secrets(c.cluster.Namespace).Create(s); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "creating empty config file failed")
	}

	c.logger.Infof("Creating cluster STS in %v", c.cluster.Namespace)
	sts, err := makeStatefulSet(*c.cluster, nil, &c.config, ruleFileConfigMaps)
	if err != nil {
		return errors.Wrap(err, "creating statefulset definition failed")
	}

	ssetClient := c.config.KubeCli.AppsV1beta1().StatefulSets(c.cluster.Namespace)
	if _, err := ssetClient.Create(sts); err != nil {
		return errors.Wrap(err, "creating statefulset failed")
	}

	c.LogObject("creating cluster with Spec:", c.cluster.Spec)
	//util.JsonLogObject(c.logger, sts, "StatefulSet")
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
	//	c.status.ServiceName = k8sutil.ClientServiceName(c.cluster.Name)

	c.status.SetPhase(api.ClusterPhaseRunning)
	if err := c.updateCRStatus(); err != nil {
		c.logger.Warningf("update initial CR status failed: %v", err)
	}
	c.logger.Infof("start running...")

	ctx := context.TODO()
	go c.cmapInf.Run(ctx.Done()) //c.stopCh)

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

		case <-time.After(*c.cluster.Spec.ReconcileInterval):
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
				c.logger.Warningf("all %v pods are dead.", c.cluster.Name)
				c.updateMembers(etcdutil.MemberSet{})
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
			c.logger.Errorf("Reconciliation failed: %v", rerr)
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

	// Most of the time, this will be c.status being updated

	// We need to identify whether the changes relate to the containers and or
	// services and update those

	if isSpecEqual(event.cluster.Spec, *oldSpec) {
		// We have some fields that once created can not be mutated.
		if !reflect.DeepEqual(event.cluster.Spec, *oldSpec) {
			c.logger.Warnf("Ignoring update event: %#v", event.cluster.Spec)
		}
		return nil
	}

	c.logSpecUpdate(*oldSpec, event.cluster.Spec)

	// TODO: Handle "Paused"
	// Changes to Primaries will be handled at the next reconciliation interval

	// Patch the Stateful Set
	stsname := prefixedName(c.cluster.Name)
	sts, err := c.config.KubeCli.AppsV1beta1().StatefulSets(c.cluster.Namespace).Get(stsname, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("fail to get sts (%s): %v", stsname, err)
	}
	oldsts := sts.DeepCopy()

	c.logger.Infof("Changing the sts %v size from %v to %v", stsname, oldSpec.GetNumReplicas(), c.cluster.Spec.GetNumReplicas())
	intVal := int32(c.cluster.Spec.GetNumReplicas())
	sts.Spec.Replicas = &intVal
	patchdata, err := k8sutil.CreatePatch(oldsts, sts, v1beta1.StatefulSet{})
	if err != nil {
		return fmt.Errorf("error creating patch: %v", err)
	}

	_, err = c.config.KubeCli.AppsV1beta1().StatefulSets(c.cluster.Namespace).Patch(sts.GetName(), types.StrategicMergePatchType, patchdata)
	if err != nil {
		return fmt.Errorf("fail to update the sts (%s): %v", stsname, err)
	}
	c.logger.Infof("finished upgrading the sts %v", stsname)

	return nil
}

func isSpecEqual(s1, s2 api.ClusterSpec) bool {

	if s1.GetNumReplicas() != s2.GetNumReplicas() {
		return false
	}

	if s1.Paused != s2.Paused {
		return false
	}

	if s1.GetNumPrimaries() != s2.GetNumPrimaries() {
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

func (c *Cluster) Update(cl *api.ReplicatedStatefulSet) {
	c.send(&clusterEvent{
		typ:     eventModifyCluster,
		cluster: cl,
	})
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
	var failed []string
	var unready []string
	var primary []string
	var secondary []string

	for _, m := range members {
		if !util.PresentIn(m.Name, running) {
			c.logger.Infof("updateMemberStatus:  pod %v: not ready", m.Name)
			unready = append(unready, m.Name)
		} else if m.AppFailed {
			failed = append(failed, m.Name)
		} else if m.AppRunning && m.AppPrimary {
			primary = append(primary, m.Name)
		} else if m.AppRunning {
			secondary = append(secondary, m.Name)
		}
	}

	c.status.Members.Ready = running
	c.status.Members.Failed = failed
	c.status.Members.Unready = unready
	c.status.Members.Primary = primary
	c.status.Members.Secondary = secondary
}

func (c *Cluster) updateCRStatus() error {
	if reflect.DeepEqual(c.cluster.Status, c.status) {
		return nil
	}

	newCluster := c.cluster
	newCluster.Status = c.status
	newCluster, err := c.config.EtcdCRCli.ClusterlabsV1alpha1().ReplicatedStatefulSets(c.cluster.Namespace).Update(c.cluster)
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

		cl, err := c.config.EtcdCRCli.ClusterlabsV1alpha1().ReplicatedStatefulSets(c.cluster.Namespace).
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

func (c *Cluster) LogObject(text string, spec interface{}) {
	util.JsonLogObject(c.logger, spec, text)
}

func (c *Cluster) logSpecUpdate(oldSpec, newSpec api.ClusterSpec) {
	c.LogObject("spec update: Old Spec:", oldSpec)
	c.LogObject("spec update: New Spec:", newSpec)

	// TODO: Maybe this with the MarshalIndent() output
	// "github.com/sergi/go-diff/diffmatchpatch"
	//
	// dmp := diffmatchpatch.New()
	// diffs := dmp.DiffMain(specText1, specText2, false)
	// fmt.Println(dmp.DiffPrettyText(diffs))

	if c.isDebugLoggerEnabled() {
		newSpecBytes, _ := json.MarshalIndent(newSpec, "", "    ")
		oldSpecBytes, _ := json.MarshalIndent(oldSpec, "", "    ")
		c.debugLogger.LogClusterSpecUpdate(string(oldSpecBytes), string(newSpecBytes))
	}
}

func (c *Cluster) isDebugLoggerEnabled() bool {
	return c.debugLogger != nil
}
