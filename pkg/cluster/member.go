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
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"

	"k8s.io/api/core/v1"
)

func (c *Cluster) updateMembers(known etcdutil.MemberSet) error {
	if c.peers == nil {
		c.peers = etcdutil.MemberSet{}
	}
	for _, m := range known {

		if _, ok := c.peers[m.Name]; !ok {
			c.peers[m.Name] = c.newMember(m.Name, m.Namespace)
		}

		c.peers[m.Name].Online = true

		if len(c.cluster.Spec.Commands.Status) > 0 {
			stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.Commands.Status...)
			if err != nil {
				c.logger.Errorf("updateMembers:  pod %v: exec failed: %v", m.Name, err)
			}
			if stdout != "" {
				c.logger.Infof("updateMembers:  pod %v stdout: %v", m.Name, stdout)
			}
			if stderr != "" {
				c.logger.Errorf("updateMembers:  pod %v stderr: %v", m.Name, stderr)
			}
		}
	}

	missing := c.peers.Diff(known)
	for _, m := range missing {
		c.peers[m.Name].Online = false
		c.logger.Warnf("updateMembers:  pod %v offline", m.Name)
	}

	return nil
}

func (c *Cluster) newMember(name string, namespace string) *etcdutil.Member {
	if namespace == "" {
		namespace = c.cluster.Namespace
	}
	return &etcdutil.Member{
		Name:         name,
		Namespace:    namespace,
		SecurePeer:   c.isSecurePeer(),
		SecureClient: c.isSecureClient(),
	}
}

func (c *Cluster) podsToMemberSet(pods []*v1.Pod, sc bool) etcdutil.MemberSet {
	members := etcdutil.MemberSet{}
	for _, pod := range pods {
		m := c.newMember(pod.Name, pod.Namespace)
		members.Add(m)
	}
	return members
}
