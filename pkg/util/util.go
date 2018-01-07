// Copyright 2017 The etcd-operator Authors
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

package util

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

// PresentIn returns true if the given string is part of an array of strings
func PresentIn(a string, list []string) bool {
	for _, l := range list {
		if a == l {
			return true
		}
	}
	return false
}

func LogOutput(logger *logrus.Entry, id string, result string) {
	if result != "" {
		lines := strings.Split(result, "\n")
		for n, l := range lines {
			logger.WithField("pod", id).Infof("[%v][%v]", n, l)
		}
	}
}

func GetLogger(component string) *logrus.Entry {
	l := logrus.New()
	f := new(prefixed.TextFormatter)
	f.ForceFormatting = true
	l.Formatter = f
	return l.WithField("pkg", component)
}

func JsonLogObject(logger *logrus.Entry, spec interface{}, text string) {
	specBytes, err := json.MarshalIndent(spec, "", "    ")
	if err != nil {
		logger.Errorf("failed to marshal spec for '%v': %v", text, err)
	}

	logger.Info(text)
	for _, m := range strings.Split(string(specBytes), "\n") {
		logger.Info(m)
	}
}
