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

package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/beekhof/rss-operator/pkg/util"
	"github.com/beekhof/rss-operator/test/e2e/e2eutil"
	"github.com/beekhof/rss-operator/test/e2e/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReadyMembersStatus(t *testing.T) {
	if os.Getenv(envParallelTest) == envParallelTestTrue {
		t.Parallel()
	}
	f := framework.Global
	size := int32(1)
	testEtcd, err := e2eutil.CreateCluster(t, f.CRClient, f.Namespace, e2eutil.NewCluster("test-etcd-", size, "", nil, nil))
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := e2eutil.DeleteCluster(t, f.CRClient, f.KubeClient, testEtcd); err != nil {
			t.Fatal(err)
		}
	}()

	if _, err := e2eutil.WaitUntilSizeReached(t, f.CRClient, int(size), 3, testEtcd); err != nil {
		t.Fatalf("failed to create %d members etcd cluster: %v", size, err)
	}

	err = util.Retry(5*time.Second, 3, func() (done bool, err error) {
		currEtcd, err := f.CRClient.ClusterlabsV1alpha1().ReplicatedStatefulSets(f.Namespace).Get(testEtcd.Name, metav1.GetOptions{})
		if err != nil {
			e2eutil.LogfWithTimestamp(t, "failed to get updated cluster object: %v", err)
			return false, nil
		}
		if len(currEtcd.Status.Members.Ready) != int(size) {
			e2eutil.LogfWithTimestamp(t, "size of ready members want = %d, get = %d ReadyMembers(%v) UnreadyMembers(%v). Will retry checking ReadyMembers", size, len(currEtcd.Status.Members.Ready), currEtcd.Status.Members.Ready, currEtcd.Status.Members.Unready)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to get size of ReadyMembers to reach %d : %v", size, err)
	}
}
