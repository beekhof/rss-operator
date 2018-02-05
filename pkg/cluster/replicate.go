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

func appendNonNil(errors []error, err error) []error {
	if err == nil || err.Error() == "" {
		return errors
	}
	return append(errors, err)
}

func combineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	var strArray []string
	for _, err := range errors {
		strArray = append(strArray, err.Error())
	}

	return fmt.Errorf("[%v]", strings.Join(strArray, ", "))
}

func (c *Cluster) replicate() error {
	errors := []error{}

	replicas := c.cluster.Spec.GetNumReplicas()
	primaries := c.cluster.Spec.GetNumPrimaries()

	if c.peers.AppMembers() > replicas {
		return fmt.Errorf("Waiting for %v peers to be stopped", c.peers.AppMembers()-primaries)
	}

	if c.peers.AppPrimaries() == 0 && c.peers.ActiveMembers() != replicas {
		return fmt.Errorf("Waiting for %v additional peer containers to be available before seeding", c.peers.ActiveMembers()-replicas)
	}

	// Always detect members (so that the most up-to-date ones get started)
	c.detectMembers()

	if c.peers.AppPrimaries() == 0 {
		seeds, err := chooseSeeds(c)
		if err != nil {
			return fmt.Errorf("Bootstrapping failed: %v", err)
		}

		for n, seed := range seeds {
			errors = appendNonNil(errors, c.startAppMember(seed, true))
			if c.peers.AppPrimaries() != 0 {
				break
			}
			c.logger.Warnf("Seed %v failed, %v remaining", len(seeds)-n)
		}

		if c.peers.AppPrimaries() == 0 {
			return fmt.Errorf("Bootstrapping from %v possible seeds with version %v failed: %v", len(seeds), seeds[0].SEQ, combineErrors(errors))
		}

		// Continue as long as one seed succeeds
		errors = []error{}
	}

	// Whichever stage we're up to... do as much as we can before exiting rather than giving up at the first error

	if len(errors) == 0 {
		for c.peers.AppPrimaries() < primaries {
			c.logger.Infof("Starting %v primaries", primaries-c.peers.AppPrimaries())

			seed, err := chooseMember(c)
			errors = appendNonNil(errors, err)
			if err != nil {
				break
			}

			errors = appendNonNil(errors, c.startAppMember(seed, true))
			if c.peers.AppPrimaries() == 0 {
				break
			}
		}
	}

	if len(errors) == 0 {
		for c.peers.AppPrimaries() > primaries {
			c.logger.Infof("Demoting %v primaries", c.peers.AppPrimaries()-primaries)

			seed, err := chooseCurrentPrimary(c)
			errors = appendNonNil(errors, err)
			if err != nil {
				break
			}

			errors = appendNonNil(errors, c.stopAppMember(seed))
		}
	}

	if len(errors) == 0 {
		for c.peers.AppMembers() < replicas {
			c.logger.Infof("Starting %v secondaries", replicas-c.peers.AppMembers())

			seed, err := chooseMember(c)
			errors = appendNonNil(errors, err)
			if err != nil {
				break
			}

			errors = appendNonNil(errors, c.startAppMember(seed, false))
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("%v of %v primaries, and %v of %v members available: %v",
			c.peers.AppPrimaries(), primaries, c.peers.AppMembers(), replicas, combineErrors(errors))
	} else if replicas > 0 {
		c.status.RestoreReplicas = replicas
	}

	c.updateCRStatus("replicate")
	c.logger.Infof("Replication complete: %v of %v primaries, and %v of %v members available",
		c.peers.AppPrimaries(), primaries, c.peers.AppMembers(), replicas)
	return nil
}

func (c *Cluster) detectMembers() {
	for _, m := range c.peers {
		stdout, stderr, err := k8sutil.ExecCommandInPodWithFullOutput(c.logger, c.config.KubeCli, c.cluster.Namespace, m.Name, c.cluster.Spec.Pod.Commands.Sequence...)

		if err != nil {
			c.logger.Errorf("discover:  pod %v: exec failed: %v", m.Name, err)

		} else {
			if stdout != "" {
				c.peers[m.Name].SEQ, err = strconv.ParseUint(stdout, 10, 64)
				if err != nil {
					c.logger.WithField("pod", m.Name).Errorf("discover:  pod %v: could not parse '%v' into uint64: %v", m.Name, stdout, err)
				}

			} else {
				c.logger.WithField("pod", m.Name).Infof("discover:  pod %v sequence now: %v", m.Name, c.peers[m.Name].SEQ)
			}
		}

		util.LogOutput(c.logger.WithField("sequence", "stdout"), logrus.InfoLevel, m.Name, stdout)
		util.LogOutput(c.logger.WithField("sequence", "stderr"), logrus.InfoLevel, m.Name, stderr)
	}
}

func chooseSeeds(c *Cluster) ([]*etcdutil.Member, error) {
	var bestPeer []*etcdutil.Member
	if c.peers == nil {
		return bestPeer, fmt.Errorf("No known peers")
	}
	for _, m := range c.peers {
		if !m.Online || m.AppFailed {
			continue
		} else if m.AppPrimary {
			continue
		} else if len(bestPeer) == 0 {
			bestPeer = append(bestPeer, m)
		} else if m.SEQ > bestPeer[0].SEQ {
			bestPeer = []*etcdutil.Member{m}
		} else {
			bestPeer = append(bestPeer, m)
		}
	}
	if len(bestPeer) == 0 {
		return bestPeer, fmt.Errorf("No peers available")
	}
	return bestPeer, nil
}

func chooseMember(c *Cluster) (*etcdutil.Member, error) {
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
		} else if strings.Compare(m.Name, bestPeer.Name) < 0 {
			// Prefer sts members towards the start of the range to be the seed
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
		} else if bestPeer == nil {
			bestPeer = m
		} else if strings.Compare(m.Name, bestPeer.Name) > 0 {
			// Prefer sts members towards the end of the range to stop/demote
			bestPeer = m
		}
	}
	if bestPeer == nil {
		return nil, fmt.Errorf("No peers remaining")
	}
	return bestPeer, nil
}

func (c *Cluster) execCommand(podName string, stdin string, cmd ...string) (string, string, error) {
	return k8sutil.ExecWithOptions(c.logger, c.config.KubeCli, k8sutil.ExecOptions{
		Command:       cmd,
		Namespace:     c.cluster.Namespace,
		PodName:       podName,
		ContainerName: "",

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
	startCmd := c.cluster.Spec.Pod.Commands.Primary

	if asPrimary && c.peers.AppPrimaries() == 0 && len(c.cluster.Spec.Pod.Commands.Seed) > 0 {
		action = "seed"
		startCmd = c.cluster.Spec.Pod.Commands.Seed
	} else if !asPrimary && len(c.cluster.Spec.Pod.Commands.Secondary) > 0 {
		action = "secondary"
		startCmd = c.cluster.Spec.Pod.Commands.Secondary
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
	util.LogOutput(c.logger.WithField(action, "stdout"), level, m.Name, stdout)
	util.LogOutput(c.logger.WithField(action, "stderr"), level, m.Name, stderr)

	if err != nil {
		m.AppFailed = true
		m.Failures += 1
		return fmt.Errorf("Could not create app %v on %v: %v", action, m.Name, err)
	}

	c.logger.Infof("Created app %v on %v", action, m.Name)
	m.AppPrimary = asPrimary
	m.AppRunning = true
	m.AppFailed = false
	return nil
}

func (c *Cluster) stopAppMember(m *etcdutil.Member) error {
	stdout, stderr, err := c.execCommand(m.Name, "", c.cluster.Spec.Pod.Commands.Stop...)
	level := logrus.DebugLevel
	if err != nil {
		level = logrus.ErrorLevel
		c.logger.Errorf("stop: pod %v: exec failed: %v", m.Name, err)
	}
	action := "stop"
	util.LogOutput(c.logger.WithField(action, "stdout"), level, m.Name, stdout)
	util.LogOutput(c.logger.WithField(action, "stderr"), level, m.Name, stderr)
	if err != nil {
		m.AppFailed = true
		m.Failures += 1
		return fmt.Errorf("Could not stop %v: %v", m.Name, err)

	} else {
		m.AppPrimary = false
		m.AppRunning = false
		m.AppFailed = false
	}
	return nil
}
