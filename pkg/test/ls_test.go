/*
Copyright © 2020 FairOS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test_test

import (
	"os"
	"testing"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"

	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"

	"github.com/plexsysio/taskmanager"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
)

func TestPod_ListPods(t *testing.T) {
	mockClient := mock.NewMockBeeClient()
	logger := logging.New(os.Stdout, 0)
	acc := account.New(logger)
	accountInfo := acc.GetUserAccountInfo()
	fd := feed.New(accountInfo, mockClient, logger)
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	pod1 := pod.NewPod(mockClient, fd, acc, tm, logger)
	_, _, err := acc.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}

	podName1 := "test1"
	podName2 := "test2"

	t.Run("list-without-pods", func(t *testing.T) {
		_, _, err = pod1.ListPods()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create-two-pods", func(t *testing.T) {
		podPassword, _ := utils.GetRandString(pod.PodPasswordLength)
		_, err := pod1.CreatePod(podName1, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod: %v", err)
		}
		_, err = pod1.CreatePod(podName2, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName1)
		}

		pods, _, err := pod1.ListPods()
		if err != nil {
			t.Fatal(err)
		}

		if pods[0] != podName1 && pods[1] != podName1 {
			t.Fatalf("pod not found")
		}
		if pods[0] != podName2 && pods[1] != podName2 {
			t.Fatalf("pod not found")
		}
	})
}
