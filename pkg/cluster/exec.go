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
	"regexp"
	"strconv"
	"strings"
	"time"

	api "github.com/beekhof/rss-operator/pkg/apis/galera/v1alpha1"
	"github.com/beekhof/rss-operator/pkg/util"
	"github.com/beekhof/rss-operator/pkg/util/k8sutil"

	"github.com/sirupsen/logrus"
)

func parseExitCode(err error) int {
	search := "command terminated with exit code "
	if err == nil {
		return 0
	}
	str := err.Error()
	i := strings.Index(str, search)
	if i < 0 {
		return 1
	}
	i = i + len(search)
	sub := str[i:]
	val, err := strconv.ParseInt(sub, 10, 32)
	if err != nil {
		logrus.Errorf("No integer in '%v'", sub)
		return 1
	}
	return int(val)
}

func parseDuration(str *string) time.Duration {

	if str == nil {
		return time.Duration(0)
	}

	durationRegex := regexp.MustCompile(`(?P<years>\d+Y)?(?P<months>\d+M)?(?P<days>\d+D)?T?(?P<hours>\d+h)?(?P<minutes>\d+m)?(?P<seconds>\d+s)?`)
	matches := durationRegex.FindStringSubmatch(*str)

	years := parseInt64(matches[1])
	months := parseInt64(matches[2])
	days := parseInt64(matches[3])
	hours := parseInt64(matches[4])
	minutes := parseInt64(matches[5])
	seconds := parseInt64(matches[6])

	if matches[0] == "" {
		// Simple numbers are treated as seconds
		intSeconds, _ := strconv.Atoi(*str)
		seconds = int64(intSeconds)
	}

	hour := int64(time.Hour)
	minute := int64(time.Minute)
	second := int64(time.Second)
	return time.Duration(years*24*365*hour + months*30*24*hour + days*24*hour + hours*hour + minutes*minute + seconds*second)
}

func parseInt64(value string) int64 {
	if len(value) == 0 {
		return 0
	}
	parsed, err := strconv.Atoi(value[:len(value)-1])
	if err != nil {
		return 0
	}
	return int64(parsed)
}

func (c *Cluster) execute(action string, podName string, silent bool) (string, string, error, int) {
	rc := 0
	level := logrus.DebugLevel
	cmd := c.rss.Spec.Pod.Commands[action]
	timeout := parseDuration(cmd.Timeout)

	// Sanitise timeouts
	minTimeout := 10 * time.Second
	if timeout == time.Duration(0) {
		timeout = 10 * time.Minute
	} else if timeout < minTimeout {
		timeout = minTimeout
	}

	str := fmt.Sprintf("Calling '%v' command on %v with timeout %v: %v", action, podName, timeout, cmd.Command)

	switch action {
	case api.StatusCommandKey, api.SequenceCommandKey:
		c.logger.Debug(str)
	default:
		c.logger.Info(str)
	}

	stdout, stderr, err := k8sutil.ExecWithOptions(&c.execContext, k8sutil.ExecOptions{
		Command:       c.appendPrimaries(cmd.Command),
		Namespace:     c.rss.Namespace,
		PodName:       podName,
		ContainerName: "", // Auto-detect

		Timeout: timeout,

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
		rc = parseExitCode(err)
		return stdout, stderr, fmt.Errorf("Application %v on pod %v failed: %v", action, podName, err), rc
	}
	return stdout, stderr, nil, rc
}

func (c *Cluster) appendPrimaries(cmd []string) []string {
	for _, m := range c.peers {
		if m.Online && m.AppPrimary {
			cmd = append(cmd, fmt.Sprintf("%v.%v", m.Name, c.rss.ServiceName(true)))
		}
	}
	return cmd

}
