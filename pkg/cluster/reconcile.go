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

	api "github.com/beekhof/galera-operator/pkg/apis/galera/v1alpha1"
	"github.com/beekhof/galera-operator/pkg/util"
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
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
	var errors []error
	c.logger.Infoln("Start reconciling")
	defer c.logger.Infoln("Finish reconciling")

	defer func() {
		c.status.Replicas = c.peers.Size()
		c.updateCRStatus("reconcile")
	}()

	sp := c.rss.Spec
	running := c.podsToMemberSet(pods, c.isSecureClient())
	errors = appendNonNil(errors, c.reconcileMembers(running))

	for _, m := range c.peers {

		// TODO: Make the threshold configurable
		// ' > 1' means that we tried at least a start and a stop
		if m.AppFailed && m.Failures > 1 {
			errors = append(errors, fmt.Errorf("%v deletion after %v failures", m.Name, m.Failures))
			err := c.config.KubeCli.CoreV1().Pods(c.rss.Namespace).Delete(m.Name, &metav1.DeleteOptions{})
			if err != nil {
				errors = appendNonNil(errors, fmt.Errorf("reconcile: could not delete pod %v", m.Name, err))

			} else {
				c.logger.Warnf("reconcile: deleted pod %v", m.Name)
				c.memberOffline(m)
			}

		} else if !m.Online {
			c.logger.Infof("reconcile: Skipping offline pod %v", m.Name)
			continue

		} else if m.AppFailed {
			c.logger.Warnf("reconcile: Cleaning up pod %v", m.Name)
			errors = appendNonNil(errors, c.stopAppMember(m))

		} else if !m.AppRunning {
			c.logger.Infof("reconcile: Skipping stopped pod %v", m.Name)
			continue

		} else if _, ok := c.rss.Spec.Pod.Commands[api.StatusCommandKey]; ok {
			_, _, err, _ := c.execute(api.StatusCommandKey, m.Name, false)
			if err != nil {
				m.AppFailed = true
			}
		}
	}

	if c.peers.ActiveMembers() > sp.GetNumReplicas() {
		c.status.SetScalingDownCondition(c.peers.ActiveMembers(), sp.GetNumReplicas())

	} else if c.peers.ActiveMembers() < sp.GetNumReplicas() {
		c.status.SetScalingUpCondition(c.peers.ActiveMembers(), sp.GetNumReplicas())

	} else if len(errors) > 0 {
		c.status.SetRecoveringCondition()

	} else {
		c.status.SetReadyCondition()
	}

	return combineErrors(errors)
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

	if c.peers.Size() < c.rss.Spec.GetNumReplicas()/2+1 {
		c.logger.Infof("Quorum lost")
		return ErrLostQuorum
	}
	return nil
}
