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
	"errors"
	"fmt"

	"github.com/beekhof/galera-operator/pkg/util"
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ErrLostQuorum indicates that the etcd cluster lost its quorum.
var ErrLostQuorum = errors.New("lost quorum")

// reconcile reconciles cluster current state to desired state specified by spec.
// - it tries to reconcile the cluster to desired size.
// - if the cluster needs for upgrade, it tries to upgrade old member one by one.
func (c *Cluster) reconcile(pods []*v1.Pod) error {
	c.logger.Infoln("Start reconciling")
	defer c.logger.Infoln("Finish reconciling")

	defer func() {
		c.status.Replicas = c.peers.Size()
	}()

	var err error = nil
	sp := c.cluster.Spec
	running := c.podsToMemberSet(pods, c.isSecureClient())
	if c.peers.AppMembers() != sp.GetNumReplicas() {
		err = c.reconcileMembers(running)
	} else {
		c.status.SetReadyCondition()
	}

	for _, m := range c.peers {

		// TODO: Make the threshold configurable
		// ' > 1' means that we tried at least a start and a stop
		if m.AppFailed && m.Failures > 1 {
			err = c.config.KubeCli.CoreV1().Pods(c.cluster.Namespace).Delete(m.Name, &metav1.DeleteOptions{})
			if err != nil {
				c.logger.Errorf("reconcile: could not delete pod %v  (%v failures): %v", m.Name, m.Failures, err)
				return fmt.Errorf("Pod deletion failed: %v", err)

			} else {
				c.logger.Warnf("reconcile: deleted pod %v (%v failures)", m.Name, m.Failures)
				c.memberOffline(m)
			}

		} else if m.AppFailed {
			err = c.stopAppMember(m)
			if err != nil {
				c.logger.Errorf("reconcile: could not stop pod %v: %v", m.Name, err)
				return fmt.Errorf("Application stop failed on %v: %v", m.Name, err)

			} else {
				c.logger.Infof("reconcile: pod %v cleaned up", m.Name)
			}

		} else if !m.AppRunning || !m.Online {
			continue

		} else if len(c.cluster.Spec.Pod.Commands.Status) > 0 {
			action := "check"
			level := logrus.DebugLevel

			stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.Pod.Commands.Status...)
			if err != nil {
				level = logrus.ErrorLevel
				if m.AppRunning {
					m.AppFailed = true
					c.logger.Errorf("check:  pod %v: exec failed: %v", m.Name, err)
				} else {
					c.logger.Warnf("check:  pod %v: exec failed: %v", m.Name, err)
				}
			}
			util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stdout", action)), level, m.Name, stdout)
			util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stderr", action)), level, m.Name, stderr)
		}
	}
	c.status.SetReadyCondition()
	return err
}

// reconcileMembers reconciles
// - running pods on k8s and cluster membership
// - cluster membership and expected size of etcd cluster
// Steps:
// 1. Remove all pods from running set that does not belong to member set.
// 2. L consist of remaining pods of runnings
// 3. If L = members, the current state matches the membership state. END.
// 4. If len(L) < len(members)/2 + 1, return quorum lost error.
// 5. Add one missing member. END.
func (c *Cluster) reconcileMembers(running etcdutil.MemberSet) error {
	c.logger.Infof(" current membership: %s", running)
	c.logger.Infof("previous membership: %s", c.peers)

	lostMembers := c.peers.Diff(running)
	unknownMembers := running.Diff(c.peers)
	if unknownMembers.Size() > 0 || lostMembers.Size() > 0 {
		c.logger.Infof("Updating membership: new=[%v], lost=[%v]", unknownMembers, lostMembers)
		c.updateMembers(running)
	}

	if c.peers.Size() < c.cluster.Spec.GetNumReplicas()/2+1 {
		c.logger.Infof("Quorum lost")
		return ErrLostQuorum
	}
	return nil
}
