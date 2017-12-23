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
	"strings"

	"github.com/beekhof/galera-operator/pkg/util/etcdutil"
	"github.com/beekhof/galera-operator/pkg/util/k8sutil"
)

func (c *Cluster) replicate() error {
	primaries := c.cluster.Spec.MaxPrimaries
	var err error = nil

	if primaries < 1 || primaries > c.cluster.Spec.Size {
		primaries = c.cluster.Spec.Size
	}

	for err == nil && c.peers.AppPrimaries() < primaries {
		err := c.startPrimary()
	}

	for err == nil && c.peers.AppPrimaries() > primaries {
		err := c.demotePrimary()
	}

	for err == nil && c.peers.AppMembers() < c.cluster.Spec.Size {
		err = c.startMember(m)
	}

	if err != nil {
		return fmt.Errorf("%v of %v primaries, and %v of %v members available: %v",
			c.peers.AppPrimaries(), primaries, c.peers.AppMembers(), c.cluster.Spec.Size, err)
	}
	return nil
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
		return nil, fmt.Errorf("No know peers")
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
		return fmt.Errorf("%v of %v members available: %v", c.peers.AppMembers(), c.cluster.Spec.Size, err)
	}
	return c.startAppMember(m, false)
}

func (c *Cluster) startAppMember(m *etcdutil.Member, asPrimary bool) error {
	startCmd := c.cluster.Spec.StartPrimaryCommand

	if asPrimary && c.peers.AppPrimaries() == 0 && len(c.cluster.Spec.StartSeedCommand) > 0 {
		startCmd = c.cluster.Spec.StartSeedCommand
	} else if !asPrimary && len(c.cluster.Spec.StartSecondaryCommand) > 0 {
		startCmd = c.cluster.Spec.StartSecondaryCommand
	}

	if asPrimary && c.peers.AppPrimaries() == 0 {
		c.logger.Infof("Seeding from pod %v: %v", m.Name, m.SEQ)
	}
	stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, startCmd...)
	if err != nil {
		c.logger.Errorf("startAppMember: pod %v: exec failed: %v", m.Name, err)
		m.AppFailed = true
		if asPrimary {
			return fmt.Errorf("Could not seed app on %v: %v", m.Name, err)
		} else {
			return fmt.Errorf("Could not start app on %v: %v", m.Name, err)
		}

	} else {
		if stdout != "" {
			c.logger.Infof("startAppMember: pod %v stdout: %v", m.Name, stdout)
		}
		if stderr != "" {
			c.logger.Errorf("startAppMember: pod %v stderr: %v", m.Name, stderr)
		}
		m.AppPrimary = asPrimary
		m.AppRunning = true
		m.AppFailed = false
	}
	return nil
}

func (c *Cluster) stopAppMember(m *etcdutil.Member) error {
	c.logger.Infof("Stopping pod %v: %v", m.Name, m.SEQ)
	stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.StopCommand...)
	if err != nil {
		c.logger.Errorf("stopAppMember: pod %v: exec failed: %v", m.Name, err)
		m.AppFailed = true

		return fmt.Errorf("Could not stop %v: %v", m.Name, err)

	} else {
		if stdout != "" {
			c.logger.Infof("stopAppMember: pod %v stdout: %v", m.Name, stdout)
		}
		if stderr != "" {
			c.logger.Errorf("stopAppMember: pod %v stderr: %v", m.Name, stderr)
		}
		m.AppPrimary = false
		m.AppRunning = false
		m.AppFailed = false
	}
	return nil
}
