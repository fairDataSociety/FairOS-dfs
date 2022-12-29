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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"

	"github.com/plexsysio/taskmanager"

	"github.com/fairdatasociety/fairOS-dfs/pkg/file"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
)

func TestRemoveFile(t *testing.T) {
	mockClient := mock.NewMockBeeClient()
	logger := logging.New(io.Discard, 0)
	acc := account.New(logger)
	_, _, err := acc.CreateUserAccount("")
	require.NoError(t, err)

	pod1AccountInfo, err := acc.CreatePodAccount(1, false)
	require.NoError(t, err)

	fd := feed.New(pod1AccountInfo, mockClient, logger)
	user := acc.GetAddress(1)
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	defer func() {
		_ = tm.Stop(context.Background())
	}()
	t.Run("delete-file", func(t *testing.T) {
		podPassword, _ := utils.GetRandString(pod.PodPasswordLength)
		fileObject := file.NewFile("pod1", mockClient, fd, user, tm, logger)
		// remove file2
		err = fileObject.RmFile("/dir1/file2", podPassword)
		require.Equal(t, err.Error(), "feed does not exist or was not updated yet")

		file1, _ := utils.GetRandString(12)
		file2, _ := utils.GetRandString(12)
		// upload few files
		_, err = uploadFile(t, fileObject, "/dir1", file1, "", podPassword, 100, 10)
		require.NoError(t, err)

		_, err = uploadFile(t, fileObject, "/dir1", file2, "", podPassword, 200, 20)
		require.NoError(t, err)

		// remove file2
		err = fileObject.RmFile("/dir1/"+file2, podPassword)
		require.NoError(t, err)

		// validate file deletion
		meta := fileObject.GetFromFileMap(utils.CombinePathAndFile("/dir1", file2))
		if meta != nil {
			t.Fatalf("file is not removed")
		}

		// check if other file is present
		meta = fileObject.GetFromFileMap(utils.CombinePathAndFile("/dir1", file1))
		if meta == nil {
			t.Fatalf("file is not present")
		}
		if meta.Name != file1 {
			t.Fatalf("retrieved invalid file name")
		}
		err := fileObject.LoadFileMeta(utils.CombinePathAndFile("/dir1", file1), podPassword)
		require.NoError(t, err)
	})

	t.Run("delete-file-in-loop", func(t *testing.T) {
		fileObject := file.NewFile("pod1", mockClient, fd, user, tm, logger)
		podPassword, _ := utils.GetRandString(pod.PodPasswordLength)

		for i := 0; i < 80; i++ {
			filename, _ := utils.GetRandString(12)
			// upload file1
			_, err = uploadFile(t, fileObject, "/dir1", filename, "", podPassword, 100, 10)
			require.NoError(t, err)

			// remove file1
			err = fileObject.RmFile("/dir1/"+filename, podPassword)
			require.NoError(t, err)

			// validate file deletion
			meta := fileObject.GetFromFileMap("/dir1/" + filename)
			if meta != nil {
				t.Fatalf("file is not removed")
			}
		}
	})
}
