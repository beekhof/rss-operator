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
	if c.cluster.Spec.MaxSeeds < 1 {
		c.cluster.Spec.MaxSeeds = c.cluster.Spec.Size
	}
	if c.cluster.Spec.MaxSeeds > c.cluster.Spec.Size {
		c.cluster.Spec.MaxSeeds = c.cluster.Spec.Size
	}

	for c.peers.Seeds() < c.cluster.Spec.MaxSeeds {
		err := c.startSeed()
		if err != nil {
			return fmt.Errorf("%v of %v seeds available: %v", c.peers.Seeds(), c.cluster.Spec.MaxSeeds, err)
		}
	}

	for c.peers.Seeds() > c.cluster.Spec.MaxSeeds {
		err := c.demoteSeed()
		if err != nil {
			return fmt.Errorf("%v of %v seeds still running: %v", c.peers.Seeds(), c.cluster.Spec.MaxSeeds, err)
		}
	}

	// Needed? Or just let STS controller stop them
	for c.peers.AppMembers() > c.cluster.Spec.Size {
		m, err := chooseSeed(c)
		if err != nil {
			return fmt.Errorf("%v of %v members available: %v", c.peers.AppMembers(), c.cluster.Spec.Size, err)
		}
		c.startMember(m)
	}
	return nil
}

func chooseSeed(c *Cluster) (*etcdutil.Member, error) {
	var bestPeer *etcdutil.Member
	if c.peers == nil {
		return nil, fmt.Errorf("No known peers")
	}
	for _, m := range c.peers {
		if !m.Online {
			continue
		} else if m.AppSeed {
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

func chooseCurrentSeed(c *Cluster) (*etcdutil.Member, error) {
	var bestPeer *etcdutil.Member
	if c.peers == nil {
		return nil, fmt.Errorf("No know peers")
	}
	for _, m := range c.peers {
		if !m.AppSeed {
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

func (c *Cluster) demoteSeed() error {
	seed, err := chooseCurrentSeed(c)
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

func (c *Cluster) startSeed() error {
	seed, err := chooseSeed(c)
	if err != nil {
		return err
	}
	return c.startAppMember(seed, true)
}

func (c *Cluster) startMember(m *etcdutil.Member) error {
	return c.startAppMember(m, false)
}

func (c *Cluster) startAppMember(m *etcdutil.Member, asSeed bool) error {
	startCmd := c.cluster.Spec.StartMemberCommand

	if asSeed && len(c.cluster.Spec.StartSeedCommand) > 0 {
		startCmd = c.cluster.Spec.StartSeedCommand
	}

	if asSeed {
		c.logger.Infof("Seeding from pod %v: %v", m.Name, m.SEQ)
	}
	stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, startCmd...)
	if err != nil {
		c.logger.Errorf("startAppMember: pod %v: exec failed: %v", m.Name, err)
		if asSeed {
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
		m.AppSeed = asSeed
		m.AppRunning = true
	}
	return nil
}

func (c *Cluster) stopAppMember(m *etcdutil.Member) error {
	c.logger.Infof("Stopping pod %v: %v", m.Name, m.SEQ)
	stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.StopCommand...)
	if err != nil {
		c.logger.Errorf("stopAppMember: pod %v: exec failed: %v", m.Name, err)
		return fmt.Errorf("Could not stop %v: %v", m.Name, err)

	} else {
		if stdout != "" {
			c.logger.Infof("stopAppMember: pod %v stdout: %v", m.Name, stdout)
		}
		if stderr != "" {
			c.logger.Errorf("stopAppMember: pod %v stderr: %v", m.Name, stderr)
		}
		m.AppSeed = false
		m.AppRunning = false
	}
	return nil
}
