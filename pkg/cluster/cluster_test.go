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

package cluster

import (
	"fmt"
	"testing"
	"time"

	api "github.com/beekhof/rss-operator/pkg/apis/galera/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// When ReplicatedStatefulSet update event happens, local object ref should be updated.
func TestUpdateEventUpdateLocalClusterObj(t *testing.T) {
	oldVersion := "123"
	newVersion := "321"

	oldObj := &api.ReplicatedStatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: api.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: oldVersion,
			Name:            "test",
			Namespace:       metav1.NamespaceDefault,
		},
	}
	newObj := oldObj.DeepCopy()
	newObj.ResourceVersion = newVersion

	c := &Cluster{
		rss: oldObj,
	}
	e := &clusterEvent{
		typ: eventModifyCluster,
		rss: newObj,
	}

	err := c.handleUpdateEvent(e)
	if err != nil {
		t.Fatal(err)
	}
	if c.rss.ResourceVersion != newVersion {
		t.Errorf("expect version=%s, get=%s", newVersion, c.rss.ResourceVersion)
	}
}

func TestErrorCodes(t *testing.T) {

	if 7 != parseExitCode(fmt.Errorf("command terminated with exit code 7")) {
		t.Fatal("Expected 7")
	}
	if 1 != parseExitCode(fmt.Errorf("command terminated with exit code ")) {
		t.Fatal("Expected 1 for empty")
	}
	if 1 != parseExitCode(fmt.Errorf("command terminated with exit code A")) {
		t.Fatal("Expected 1 for A")
	}
	if 1 != parseExitCode(fmt.Errorf("command")) {
		t.Fatal("Expected 1 for not found")
	}
	if 0 != parseExitCode(nil) {
		t.Fatal("Expected 0 for nil")
	}
}

func TestParseDuration(t *testing.T) {
	str := "15m"
	expected := time.Duration(15 * time.Minute)
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	str = "15m10s"
	expected = time.Duration(15*time.Minute + 10*time.Second)
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	str = "15m10s"
	expected = time.Duration(15*time.Minute + 10*time.Second)
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	str = "10s"
	expected = time.Duration(10 * time.Second)
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	str = "8"
	expected = time.Duration(8 * time.Second)
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	str = "abc"
	expected = time.Duration(0)
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	str = ""
	if result := parseDuration(&str); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}

	if result := parseDuration(nil); expected != result {
		t.Fatalf("Expected '%v' from '%v', got '%v'", expected, str, result)
	}
}
