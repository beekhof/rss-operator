// Copyright 2017 Andrew Beekhof
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
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/beekhof/galera-operator/pkg/util"
	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"
)

func (c *Cluster) replicate() error {
	var err error = nil
	primaries := 0

	if c.cluster.Spec.Primaries != nil {
		primaries = *c.cluster.Spec.Primaries
	}

	if primaries < 1 || primaries > *c.cluster.Spec.Replicas {
		primaries = *c.cluster.Spec.Replicas
	}

	if c.peers.AppPrimaries() == 0 {
		c.detectMembers()
	}

	for err == nil && c.peers.AppPrimaries() < primaries {
		err = c.startPrimary()
	}

	for err == nil && c.peers.AppPrimaries() > primaries {
		err = c.demotePrimary()
	}

	for err == nil && c.peers.AppMembers() < *c.cluster.Spec.Replicas {
		err = c.startMember()
	}

	if err != nil {
		return fmt.Errorf("Replication failed: %v of %v primaries, and %v of %v members available: %v",
			c.peers.AppPrimaries(), primaries, c.peers.AppMembers(), c.cluster.Spec.Replicas, err)
	}
	return nil
}

func (c *Cluster) detectMembers() {
	for _, m := range c.peers {
		stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.Commands.Sequence...)
		util.LogOutput(c.logger.WithField("source", "detectMembers:stdout"), logrus.InfoLevel, m.Name, stdout)
		util.LogOutput(c.logger.WithField("source", "detectMembers:stderr"), logrus.InfoLevel, m.Name, stderr)

		if err != nil {
			c.logger.Errorf("detectMembers:  pod %v: exec failed: %v", m.Name, err)

		} else {
			if stdout != "" {
				c.peers[m.Name].SEQ, err = strconv.ParseUint(stdout, 10, 64)
				if err != nil {
					c.logger.WithField("pod", m.Name).Errorf("detectMembers:  pod %v: could not parse '%v' into uint64: %v", m.Name, stdout, err)
				}

			} else {
				c.logger.WithField("pod", m.Name).Infof("detectMembers:  pod %v sequence now: %v", m.Name, c.peers[m.Name].SEQ)
			}
		}
	}
}

func chooseSeed(c *Cluster) (*etcdutil.Member, error) {
	var bestPeer *etcdutil.Member
	if c.peers == nil {
		return nil, fmt.Errorf("No known peers")
	}
	for _, m := range c.peers {
		if !m.Online || m.AppFailed {
			continue
		} else if m.AppPrimary {
			continue
		} else if bestPeer == nil {
			bestPeer = m
		} else if m.SEQ > bestPeer.SEQ {
			bestPeer = m
		} else if strings.Compare(m.Name, bestPeer.Name) > 0 {
			// Prefer sts members towards the start of the range
			bestPeer = m
		}
	}
	if bestPeer == nil {
		return nil, fmt.Errorf("No peers available")
	}
	return bestPeer, nil
}

func chooseCurrentPrimary(c *Cluster) (*etcdutil.Member, error) {
	var bestPeer *etcdutil.Member
	if c.peers == nil {
		return nil, fmt.Errorf("No known peers")
	}
	for _, m := range c.peers {
		if !m.AppPrimary || m.AppFailed {
			continue
		} else if !m.Online {
			return m, nil
		} else if strings.Compare(m.Name, bestPeer.Name) > 0 {
			// Prefer sts members towards the start of the range
			bestPeer = m
		}
	}
	if bestPeer == nil {
		return nil, fmt.Errorf("No peers available")
	}
	return bestPeer, nil
}

func (c *Cluster) demotePrimary() error {
	seed, err := chooseCurrentPrimary(c)
	if err != nil {
		return fmt.Errorf("Could not demote seed: %v", err)
	}
	err = c.stopAppMember(seed)
	if err != nil {
		return fmt.Errorf("Could not stop app on %v: %v", seed.Name, err)
	}
	err = c.startAppMember(seed, false)
	if err != nil {
		return fmt.Errorf("Could not start app on %v: %v", seed.Name, err)
	}
	return nil
}

func (c *Cluster) startPrimary() error {
	seed, err := chooseSeed(c)
	if err != nil {
		return err
	}
	return c.startAppMember(seed, true)
}

func (c *Cluster) startMember() error {
	m, err := chooseSeed(c)
	if err != nil {
		return fmt.Errorf("%v of %v members available: %v", c.peers.AppMembers(), c.cluster.Spec.Replicas, err)
	}
	return c.startAppMember(m, false)
}

func (c *Cluster) execCommand(podName string, stdin string, cmd ...string) (string, string, error) {
	return k8sutil.ExecWithOptions(c.logger, c.config.KubeCli, k8sutil.ExecOptions{
		Command:       cmd,
		Namespace:     c.cluster.Namespace,
		PodName:       podName,
		ContainerName: "rss",

		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	})
}

func (c *Cluster) appendPrimaries(cmd []string) []string {
	for _, m := range c.peers {
		if m.Online && m.AppPrimary {
			cmd = append(cmd, fmt.Sprintf("%v.%v", m.Name, c.cluster.ServiceName(true)))
		}
	}
	return cmd

}
func (c *Cluster) startAppMember(m *etcdutil.Member, asPrimary bool) error {
	action := "primary"
	startCmd := c.cluster.Spec.Commands.Primary

	if asPrimary && c.peers.AppPrimaries() == 0 && len(c.cluster.Spec.Commands.Seed) > 0 {
		action = "seed"
		startCmd = c.cluster.Spec.Commands.Seed
	} else if !asPrimary && len(c.cluster.Spec.Commands.Secondary) > 0 {
		action = "secondary"
		startCmd = c.cluster.Spec.Commands.Secondary
	}

	startCmd = c.appendPrimaries(startCmd)

	if asPrimary && c.peers.AppPrimaries() == 0 {
		c.logger.Infof("Seeding from pod %v: %v", m.Name, m.SEQ)
	}
	stdout, stderr, err := c.execCommand(m.Name, "beekhof", startCmd...)
	level := logrus.InfoLevel
	if err != nil {
		level = logrus.ErrorLevel
		c.logger.Errorf("%v: pod %v: exec failed: %v", action, m.Name, err)
	}
	util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stdout", action)), level, m.Name, stdout)
	util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stderr", action)), level, m.Name, stderr)
	if err != nil {
		m.AppFailed = true
		if asPrimary {
			return fmt.Errorf("Could not seed app on %v: %v", m.Name, err)
		} else {
			return fmt.Errorf("Could not start app on %v: %v", m.Name, err)
		}

	} else {
		c.logger.WithField("pod", m.Name).Infof("startAppMember: pod %v running: %v", m.Name, asPrimary)
		m.AppPrimary = asPrimary
		m.AppRunning = true
		m.AppFailed = false
	}
	return nil
}

func (c *Cluster) stopAppMember(m *etcdutil.Member) error {
	stdout, stderr, err := c.execCommand(m.Name, "", c.cluster.Spec.Commands.Stop...)
	level := logrus.DebugLevel
	if err != nil {
		level = logrus.ErrorLevel
		c.logger.Errorf("stop: pod %v: exec failed: %v", m.Name, err)
	}
	action := "stop"
	util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stdout", action)), level, m.Name, stdout)
	util.LogOutput(c.logger.WithField("action", fmt.Sprintf("%v:stderr", action)), level, m.Name, stderr)
	if err != nil {
		m.AppFailed = true
		return fmt.Errorf("Could not stop %v: %v", m.Name, err)

	} else {
		m.AppPrimary = false
		m.AppRunning = false
		m.AppFailed = false
	}
	return nil
}
