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

package pod_test

import (
	"io/ioutil"
	"testing"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
)

func TestSync(t *testing.T) {
	mockClient := mock.NewMockBeeClient()
	logger := logging.New(ioutil.Discard, 0)
	acc := account.New(logger)
	_, _, err := acc.CreateUserAccount("password", "")
	if err != nil {
		t.Fatal(err)
	}

	fd := feed.New(acc.GetUserAccountInfo(), mockClient, logger)
	pod1 := pod.NewPod(mockClient, fd, acc, logger)
	podName1 := "test1"

	t.Run("sync-pod", func(t *testing.T) {
		// create a pod
		info, err := pod1.CreatePod(podName1, "password", "")
		if err != nil {
			t.Fatalf("error creating pod %s", podName1)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod1, podName1)

		// open the pod ( ths triggers sync too
		gotInfo, err := pod1.OpenPod(podName1, "password")
		if err != nil {
			t.Fatal(err)
		}

		// validate if the directory and files are synced
		dirObject := gotInfo.GetDirectory()
		dirInode1 := dirObject.GetDirFromDirectoryMap("/parentDir/subDir1")
		if dirInode1 == nil {
			t.Fatalf("invalid dir entry")
		}
		if dirInode1.Meta.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if dirInode1.Meta.Name != "subDir1" {
			t.Fatalf("invalid dir entry")
		}
		dirInode2 := dirObject.GetDirFromDirectoryMap("/parentDir/subDir2")
		if dirInode2 == nil {
			t.Fatalf("invalid dir entry")
		}
		if dirInode2.Meta.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if dirInode2.Meta.Name != "subDir2" {
			t.Fatalf("invalid dir entry")
		}

		fileObject := gotInfo.GetFile()
		fileMeta1 := fileObject.GetFromFileMap("/parentDir/file1")
		if fileMeta1 == nil {
			t.Fatalf("invalid file meta")
		}
		if fileMeta1.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if fileMeta1.Name != "file1" {
			t.Fatalf("invalid file entry")
		}
		if fileMeta1.Size != uint64(100) {
			t.Fatalf("invalid file size")
		}
		if fileMeta1.BlockSize != uint32(10) {
			t.Fatalf("invalid block size")
		}
		fileMeta2 := fileObject.GetFromFileMap("/parentDir/file2")
		if fileMeta2 == nil {
			t.Fatalf("invalid file meta")
		}
		if fileMeta2.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if fileMeta2.Name != "file2" {
			t.Fatalf("invalid file entry")
		}
		if fileMeta2.Size != uint64(200) {
			t.Fatalf("invalid file size")
		}
		if fileMeta2.BlockSize != uint32(20) {
			t.Fatalf("invalid block size")
		}
	})
}
