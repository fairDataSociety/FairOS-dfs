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
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/pkg/file"

	"github.com/asabya/swarm-blockstore/bee"
	"github.com/asabya/swarm-blockstore/bee/mock"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	mockpost "github.com/ethersphere/bee/v2/pkg/postage/mock"
	mockstorer "github.com/ethersphere/bee/v2/pkg/storer/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
	"github.com/sirupsen/logrus"

	"github.com/plexsysio/taskmanager"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dir"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
)

func TestRmdir(t *testing.T) {
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
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	defer func() {
		_ = tm.Stop(context.Background())
	}()

	fd := feed.New(pod1AccountInfo, mockClient, -1, 0, logger)
	user := acc.GetAddress(1)
	mockFile := file.NewFile("pod1", mockClient, fd, user, tm, logger)

	t.Run("simple-rmdir", func(t *testing.T) {
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, tm, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", podPassword, user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}

		err = dirObject.RmDir("", podPassword)
		if !errors.Is(err, dir.ErrInvalidDirectoryName) {
			t.Fatal("invalid dir name")
		}

		err = dirObject.RmDir("asdasd", podPassword)
		if !errors.Is(err, dir.ErrInvalidDirectoryName) {
			t.Fatal("invalid dir name")
		}
		err = dirObject.RmDir("/asdasd", podPassword)
		if !errors.Is(err, dir.ErrDirectoryNotPresent) {
			t.Fatal("dir not present")
		}

		// now delete the directory
		err = dirObject.RmDir("/dirToRemove", podPassword)
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err := dirObject.ListDir("/", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if len(dirEntry) != 0 {
			t.Fatalf("could not delete directory")
		}

		err = dirObject.RmDir("/", podPassword)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("nested-rmdir", func(t *testing.T) {
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, tm, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", podPassword, user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove1", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2/dirToRemove", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}

		// make sure directories were created
		dirEntry, _, err := dirObject.ListDir("/dirToRemove1", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2\" was not created")
		}
		dirEntry, _, err = dirObject.ListDir("/dirToRemove1/dirToRemove2", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2/dirToRemove\" was not created")
		}

		// now delete the directory
		err = dirObject.RmDir("/dirToRemove1", podPassword)
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err = dirObject.ListDir("/", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if len(dirEntry) != 0 {
			t.Fatalf("could not delete directory")
		}
	})
}

func TestRmRootDirByPath(t *testing.T) {
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
	mockFile := file.NewFile("pod1", mockClient, fd, user, tm, logger)

	t.Run("rmrootdir", func(t *testing.T) {
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, tm, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", podPassword, user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove1", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2/dirToRemove", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}

		// make sure directories were created
		dirEntry, _, err := dirObject.ListDir("/dirToRemove1", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2\" was not created")
		}
		dirEntry, _, err = dirObject.ListDir("/dirToRemove1/dirToRemove2", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2/dirToRemove\" was not created")
		}

		_, fileEntry, err := dirObject.ListDir("/dirToRemove1", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if len(fileEntry) != 0 {
			t.Fatal("there should a file entry")
		}
		// now delete the root directory
		err = dirObject.RmDir("/", podPassword)
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err = dirObject.ListDir("/", podPassword)
		if err == nil {
			t.Fatal("root directory was not deleted")
		}
		if dirEntry != nil {
			t.Fatalf("could not delete directory")
		}
	})
}

func TestRmRootDir(t *testing.T) {
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
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	defer func() {
		_ = tm.Stop(context.Background())
	}()

	fd := feed.New(pod1AccountInfo, mockClient, -1, 0, logger)
	user := acc.GetAddress(1)
	mockFile := file.NewFile("pod1", mockClient, fd, user, tm, logger)

	t.Run("rmrootdir", func(t *testing.T) {
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		dirObject := dir.NewDirectory("pod1", mockClient, fd, user, mockFile, tm, logger)

		// make root dir so that other directories can be added
		err = dirObject.MkRootDir("pod1", podPassword, user, fd)
		if err != nil {
			t.Fatal(err)
		}

		// create a new dir
		err := dirObject.MkDir("/dirToRemove1", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		// create a new dir
		err = dirObject.MkDir("/dirToRemove1/dirToRemove2/dirToRemove", podPassword, 0)
		if err != nil {
			t.Fatal(err)
		}
		node := dirObject.GetDirFromDirectoryMap("/dirToRemove1/dirToRemove2/dirToRemove")
		if node.GetDirInodePathAndName() != "/dirToRemove1/dirToRemove2/dirToRemove" {
			t.Fatal("node returned wrong path and name")
		}

		// make sure directories were created
		dirEntry, _, err := dirObject.ListDir("/dirToRemove1", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2\" was not created")
		}
		dirEntry, _, err = dirObject.ListDir("/dirToRemove1/dirToRemove2", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if dirEntry == nil {
			t.Fatal("nested directory \"/dirToRemove1/dirToRemove2/dirToRemove\" was not created")
		}

		_, fileEntry, err := dirObject.ListDir("/", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		if len(fileEntry) != 0 {
			t.Fatal("there should no file entry")
		}

		// now delete the root directory
		err = dirObject.RmRootDir(podPassword)
		if err != nil {
			t.Fatal(err)
		}

		// verify if the directory is actually removed
		dirEntry, _, err = dirObject.ListDir("/", podPassword)
		if err == nil {
			t.Fatal("root directory was not deleted")
		}
		if dirEntry != nil {
			t.Fatalf("could not delete directory")
		}
	})
}
