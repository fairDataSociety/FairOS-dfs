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

package file_test

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"

	"github.com/asabya/swarm-blockstore/bee"
	"github.com/asabya/swarm-blockstore/bee/mock"
	mockpost "github.com/ethersphere/bee/v2/pkg/postage/mock"
	mockstorer "github.com/ethersphere/bee/v2/pkg/storer/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/file"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
	"github.com/plexsysio/taskmanager"
	"github.com/sirupsen/logrus"
)

func TestStat(t *testing.T) {
	storer := mockstorer.New()
	beeUrl := mock.NewTestBeeServer(t, mock.TestServerOptions{
		Storer:          storer,
		PreventRedirect: true,
		Post:            mockpost.New(mockpost.WithAcceptAll()),
	})

	logger := logging.New(io.Discard, logrus.DebugLevel)
	mockClient := bee.NewBeeClient(beeUrl, bee.WithStamp(mock.BatchOkStr), bee.WithRedundancy(fmt.Sprintf("%d", redundancy.NONE)), bee.WithPinning(true))

	acc := account.New(logger)
	_, _, err := acc.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}
	pod1AccountInfo, err := acc.CreatePodAccount(1, false)
	if err != nil {
		t.Fatal(err)
	}
	fd := feed.New(pod1AccountInfo, mockClient, -1, 0, logger)
	user := acc.GetAddress(1)
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	defer func() {
		_ = tm.Stop(context.Background())
	}()

	podPassword, _ := utils.GetRandString(pod.PasswordLength)
	t.Run("stat-file", func(t *testing.T) {
		fileObject := file.NewFile("pod1", mockClient, fd, user, tm, logger)

		// upload a file
		_, err = uploadFile(t, fileObject, "/dir1", "file1", "", podPassword, 100, file.MinBlockSize)
		if err != nil {
			t.Fatal(err)
		}

		// stat the file
		stats, err := fileObject.GetStats("pod1", "/dir1/file1", podPassword)
		if err != nil {
			t.Fatal(err)
		}

		// validate state
		if stats.PodName != "pod1" {
			t.Fatalf("invalid pod name in stats")
		}
		if stats.FilePath != "/dir1" {
			t.Fatalf("invalid file path in stats")
		}
		if stats.FileName != "file1" {
			t.Fatalf("invalid file name in stats")
		}
		if stats.FileSize != strconv.FormatUint(100, 10) {
			t.Fatalf("invalid file size in stats")
		}
		if stats.BlockSize != fmt.Sprintf("%d", file.MinBlockSize) {
			t.Fatalf("invalid block size in stats")
		}
	})
}
