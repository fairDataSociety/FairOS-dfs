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

package dir_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	bm "github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dir"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	fm "github.com/fairdatasociety/fairOS-dfs/pkg/file/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
)

func TestRmdir(t *testing.T) {
	mockClient := bm.NewMockBeeClient()
	logger := logging.New(io.Discard, 0)
	acc := account.New(logger)
	_, _, err := acc.CreateUserAccount("password", "")
	if err != nil {
		t.Fatal(err)
	}
	pod1AccountInfo, err := acc.CreatePodAccount(1, "password", false)
	if err != nil {
		t.Fatal(err)
	}
	fd := feed.New(pod1AccountInfo, mockClient, logger)
	user := acc.GetAddress(1)
	mockFile := fm.NewMockFile()

	t.Run("simple-rmdir", func(t *testing.T) {
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove")
		if err != nil {
			t.Fatal(err)
		}

		err = dirObject.RmDir("")
		if !errors.Is(err, dir.ErrInvalidDirectoryName) {
			t.Fatal("invalid dir name")
		}

		err = dirObject.RmDir("asdasd")
		if !errors.Is(err, dir.ErrInvalidDirectoryName) {
			t.Fatal("invalid dir name")
		}
		err = dirObject.RmDir("/asdasd")
		if !errors.Is(err, dir.ErrDirectoryNotPresent) {
			t.Fatal("dir not present")
		}

		// now delete the directory
		err = dirObject.RmDir("/dirToRemove")
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err := dirObject.ListDir("/")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry != nil {
			t.Fatalf("could not delete directory")
		}

		err = dirObject.RmDir("/")
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("nested-rmdir", func(t *testing.T) {
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2")
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2/dirToRemove")
		if err != nil {
			t.Fatal(err)
		}

		// make sure directories were created
		dirEntry, _, err := dirObject.ListDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2\" was not created")
		}
		dirEntry, _, err = dirObject.ListDir("/dirToRemove1/dirToRemove2")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2/dirToRemove\" was not created")
		}

		// now delete the directory
		err = dirObject.RmDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err = dirObject.ListDir("/")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry != nil {
			t.Fatalf("could not delete directory")
		}
	})
}

func TestRmRootDirByPath(t *testing.T) {
	mockClient := bm.NewMockBeeClient()
	logger := logging.New(io.Discard, 0)
	acc := account.New(logger)
	_, _, err := acc.CreateUserAccount("password", "")
	if err != nil {
		t.Fatal(err)
	}
	pod1AccountInfo, err := acc.CreatePodAccount(1, "password", false)
	if err != nil {
		t.Fatal(err)
	}
	fd := feed.New(pod1AccountInfo, mockClient, logger)
	user := acc.GetAddress(1)
	mockFile := fm.NewMockFile()

	t.Run("rmrootdir", func(t *testing.T) {
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2")
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2/dirToRemove")
		if err != nil {
			t.Fatal(err)
		}

		// make sure directories were created
		dirEntry, _, err := dirObject.ListDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2\" was not created")
		}
		dirEntry, _, err = dirObject.ListDir("/dirToRemove1/dirToRemove2")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2/dirToRemove\" was not created")
		}

		fileName := "file1"
		err = dirObject.AddEntryToDir("/dirToRemove1", fileName, true)
		if err != nil {
			t.Fatal(err)
		}
		_, fileEntry, err := dirObject.ListDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		if len(fileEntry) != 1 {
			t.Fatal("there should a file entry")
		}
		// now delete the root directory
		err = dirObject.RmDir("/")
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err = dirObject.ListDir("/")
		if err != nil && !strings.HasSuffix(err.Error(), dir.ErrResourceDeleted.Error()) {
			t.Fatal("root directory was not deleted")
		}
		if dirEntry != nil {
			t.Fatalf("could not delete directory")
		}
	})
}

func TestRmRootDir(t *testing.T) {
	mockClient := bm.NewMockBeeClient()
	logger := logging.New(io.Discard, 0)
	acc := account.New(logger)
	_, _, err := acc.CreateUserAccount("password", "")
	if err != nil {
		t.Fatal(err)
	}
	pod1AccountInfo, err := acc.CreatePodAccount(1, "password", false)
	if err != nil {
		t.Fatal(err)
	}
	fd := feed.New(pod1AccountInfo, mockClient, logger)
	user := acc.GetAddress(1)
	mockFile := fm.NewMockFile()

	t.Run("rmrootdir", func(t *testing.T) {
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2")
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2/dirToRemove")
		if err != nil {
			t.Fatal(err)
		}
		node := dirObject.GetDirFromDirectoryMap("/dirToRemove1/dirToRemove2/dirToRemove")
		if node.GetDirInodePathAndName() != "/dirToRemove1/dirToRemove2/dirToRemove" {
			t.Fatal("node returned wrong path and name")
		}

		// make sure directories were created
		dirEntry, _, err := dirObject.ListDir("/dirToRemove1")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2\" was not created")
		}
		dirEntry, _, err = dirObject.ListDir("/dirToRemove1/dirToRemove2")
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2/dirToRemove\" was not created")
		}

		fileName := "file1"
		err = dirObject.AddEntryToDir("/", fileName, true)
		if err != nil {
			t.Fatal(err)
		}
		_, fileEntry, err := dirObject.ListDir("/")
		if err != nil {
			t.Fatal(err)
		}
		if len(fileEntry) != 1 {
			t.Fatal("there should a file entry")
		}

		// now delete the root directory
		err = dirObject.RmRootDir()
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err = dirObject.ListDir("/")
		if err != nil && !strings.HasSuffix(err.Error(), dir.ErrResourceDeleted.Error()) {
			t.Fatal("root directory was not deleted")
		}
		if dirEntry != nil {
			t.Fatalf("could not delete directory")
		}
	})
}
