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
	"fmt"

	"github.com/beekhof/galera-operator/pkg/util"
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

func (c *Cluster) execute(action string, podName string, silent bool) (string, string, error) {
	level := logrus.DebugLevel
	cmd := c.rss.Spec.Pod.Commands[action]

	stdout, stderr, err := k8sutil.ExecWithOptions(c.logger, c.config.KubeCli, k8sutil.ExecOptions{
		Command:       c.appendPrimaries(cmd.Command),
		Namespace:     c.rss.Namespace,
		PodName:       podName,
		ContainerName: "",

		Timeout: cmd.Timeout,

		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: true,
	})

	if !silent {
		if err != nil {
			level = logrus.ErrorLevel
			c.logger.Errorf("Application %v on pod %v failed: %v", action, podName, err)
		}
		util.LogOutput(c.logger.WithField(action, "stdout"), level, podName, stdout)
		util.LogOutput(c.logger.WithField(action, "stderr"), level, podName, stderr)
	}

	if err != nil {
		return stdout, stderr, fmt.Errorf("Application %v on pod %v failed: %v", action, podName, err)
	}
	return stdout, stderr, nil
}

func (c *Cluster) appendPrimaries(cmd []string) []string {
	for _, m := range c.peers {
		if m.Online && m.AppPrimary {
			cmd = append(cmd, fmt.Sprintf("%v.%v", m.Name, c.rss.ServiceName(true)))
		}
	}
	return cmd

}

func (c *Cluster) memberOffline(m *etcdutil.Member) {
	if m.Online {
		c.logger.Warnf("Pod %v offline", m.Name)
	}
	m.Online = false
	m.AppPrimary = false
	m.AppRunning = false
	m.AppFailed = false
}

func (c *Cluster) updateMembers(known etcdutil.MemberSet) error {
	if c.peers == nil {
		c.peers = etcdutil.MemberSet{}
	}
	for _, m := range known {

		if _, ok := c.peers[m.Name]; !ok {
			c.logger.Infof("Pod %v added", m.Name)
			c.peers[m.Name] = c.newMember(m.Name, m.Namespace)
		}

		if !c.peers[m.Name].Online {
			c.logger.Infof("Pod %v online", m.Name)
		}

		c.peers[m.Name].Online = true
	}

	missing := c.peers.Diff(known)
	for _, m := range missing {
		c.memberOffline(c.peers[m.Name])
	}

	return nil
}

func (c *Cluster) newMember(name string, namespace string) *etcdutil.Member {
	if namespace == "" {
		namespace = c.rss.Namespace
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
