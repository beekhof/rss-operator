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

	sp := c.cluster.Spec
	running := c.podsToMemberSet(pods, c.isSecureClient())
	if c.peers.AppMembers() != sp.Replicas {
		return c.reconcileMembers(running)
	}

	for _, m := range c.peers {
		if !m.AppPrimary || m.AppFailed {
			continue
		} else if !m.Online {
			continue

		} else if len(c.cluster.Spec.Commands.Status) > 0 {
			action := "check"
			level := logrus.DebugLevel

			stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.Commands.Status...)
			if err != nil {
				level = logrus.ErrorLevel
				c.logger.Errorf("check:  pod %v: exec failed: %v", m.Name, err)
			}
			util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stdout", action)), level, m.Name, stdout)
			util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stderr", action)), level, m.Name, stderr)
		}
	}
	c.status.SetReadyCondition()

	return nil
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
	c.logger.Infof("running members: %s", running)
	c.logger.Infof("cluster membership: %s", c.peers)

	unknownMembers := running.Diff(c.peers)
	if unknownMembers.Size() > 0 {
		c.logger.Infof("updating pods: %v", unknownMembers)
		c.updateMembers(running)
	}

	if c.peers.Size() < c.cluster.Spec.Replicas/2+1 {
		c.logger.Infof("lost quorum")
		return ErrLostQuorum
	}
	return nil
}
