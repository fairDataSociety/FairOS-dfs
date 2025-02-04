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
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"

	"github.com/asabya/swarm-blockstore/bee"
	"github.com/asabya/swarm-blockstore/bee/mock"
	"github.com/ethereum/go-ethereum/common"
	mockpost "github.com/ethersphere/bee/v2/pkg/postage/mock"
	mockstorer "github.com/ethersphere/bee/v2/pkg/storer/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/acl/acl"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/file"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
	"github.com/sirupsen/logrus"
)

func TestGroupNew(t *testing.T) {
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

	t.Run("create-first-group", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		groups, err := group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		if len(groups.Groups) != 1 {
			t.Fatalf("length of groups is not 1")
		}

		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}
	})

	t.Run("create-second-group", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)

		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		groupName2, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}
		_, err = group.CreateGroup(groupName2)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName2, err.Error())
		}
		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		_, err = group.OpenGroup(groupName2)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName2, err.Error())
		}
	})

	t.Run("group-file-upload", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		g, err := group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}

		maxfiles := 100
		filePath := "/"
		for i := 1; i <= maxfiles; i++ {
			fileName, _ := utils.GetRandString(100)
			compression := ""
			fileSize := int64(1000)
			blockSize := file.MinBlockSize
			_, err = uploadFile(t, g.GetFile(), filePath, fileName, compression, g.GetPodPassword(), fileSize, blockSize)
			if err != nil {
				t.Fatal(err)
			}
			err = g.GetDirectory().AddEntryToDir("/", g.GetPodPassword(), fileName, true)
			if err != nil {
				t.Fatal(i, err)
			}
		}

		dirInode, err := g.GetDirectory().GetInode(g.GetPodPassword(), filePath)
		if err != nil {
			t.Fatal(err)
		}
		if len(dirInode.FileOrDirNames) != maxfiles {
			t.Fatal("files not present")
		}
	})

	t.Run("group-member-add", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		g, err := group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}
		maxfiles := 10
		filePath := "/"
		for i := 1; i <= maxfiles; i++ {
			fileName, _ := utils.GetRandString(100)
			compression := ""
			fileSize := int64(1000)
			blockSize := file.MinBlockSize
			_, err = uploadFile(t, g.GetFile(), filePath, fileName, compression, g.GetPodPassword(), fileSize, blockSize)
			if err != nil {
				t.Fatal(err)
			}
			err = g.GetDirectory().AddEntryToDir("/", g.GetPodPassword(), fileName, true)
			if err != nil {
				t.Fatal(i, err)
			}
		}

		acc2 := account.New(logger)
		_, _, err = acc2.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		addr := acc2.GetUserAccountInfo().GetAddress()
		addrStr := addr.Hex()
		ref, err := group.AddMember(groupName1, common.HexToAddress(addrStr), acc2.GetUserAccountInfo().GetPublicKey(), acl.PermissionWrite)
		if err != nil {
			t.Fatal(err)
		}
		fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)

		group2 := pod.NewGroup(mockClient, fd2, acc2, mockAcl, logger)
		err = group2.AcceptGroupInvite(ref)
		if err != nil {
			t.Fatal(err)
		}

		gi, err := group2.OpenGroup(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		dirInode, err := gi.GetDirectory().GetInode(gi.GetPodPassword(), filePath)
		if err != nil {
			t.Fatal(err)
		}
		if len(dirInode.FileOrDirNames) != maxfiles {
			t.Fatal("files not present")
		}
	})

	t.Run("group-member-check-no-permission", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}

		acc2 := account.New(logger)
		_, _, err = acc2.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}

		fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl2 := acl.NewACL(mockClient, fd2, logger)

		group2 := pod.NewGroup(mockClient, fd2, acc2, mockAcl2, logger)
		groupName2, _ := utils.GetRandString(10)
		_, err = group2.CreateGroup(groupName2)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group2.OpenGroup(groupName2)
		if err != nil {
			t.Fatal(err)
		}
		perm, err := group2.GetPermission(groupName2)
		if err != nil {
			t.Fatal(err)
		}
		if perm != acl.PermissionWrite {
			t.Fatal("permission does not match")
		}

		perm1, err := group.GetPermission(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		if perm1 != acl.PermissionWrite {
			t.Fatal("permission does not match")
		}
	})

	t.Run("group-member-check-permission", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}

		acc2 := account.New(logger)
		_, _, err = acc2.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		addr := acc2.GetUserAccountInfo().GetAddress()
		addrStr := addr.Hex()
		ref, err := group.AddMember(groupName1, common.HexToAddress(addrStr), acc2.GetUserAccountInfo().GetPublicKey(), acl.PermissionRead)
		if err != nil {
			t.Fatal(err)
		}
		fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)

		group2 := pod.NewGroup(mockClient, fd2, acc2, mockAcl, logger)
		err = group2.AcceptGroupInvite(ref)
		if err != nil {
			t.Fatal(err)
		}

		_, err = group2.OpenGroup(groupName1)
		if err != nil {
			t.Fatal(err)
		}

		perm, err := group2.GetPermission(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		if perm != acl.PermissionRead {
			t.Fatal("permission not read")
		}

		err = group.UpdatePermission(groupName1, common.HexToAddress(addrStr), acl.PermissionWrite)
		if err != nil {
			t.Fatal(err)
		}
		perm, err = group2.GetPermission(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		if perm != acl.PermissionWrite {
			t.Fatal("permission not write")
		}
	})

	t.Run("group-member-upload-files", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}

		acc2 := account.New(logger)
		_, _, err = acc2.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		addr := acc2.GetUserAccountInfo().GetAddress()
		addrStr := addr.Hex()
		ref, err := group.AddMember(groupName1, common.HexToAddress(addrStr), acc2.GetUserAccountInfo().GetPublicKey(), acl.PermissionRead)
		if err != nil {
			t.Fatal(err)
		}
		fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)

		group2 := pod.NewGroup(mockClient, fd2, acc2, mockAcl, logger)
		err = group2.AcceptGroupInvite(ref)
		if err != nil {
			t.Fatal(err)
		}

		g, err := group2.OpenGroup(groupName1)
		if err != nil {
			t.Fatal(err)
		}

		fileName, _ := utils.GetRandString(100)
		compression := ""
		fileSize := int64(1000)
		blockSize := file.MinBlockSize
		_, err = uploadFile(t, g.GetFile(), "/", fileName, compression, g.GetPodPassword(), fileSize, blockSize)
		if !errors.Is(err, feed.ErrReadOnlyFeed) {
			t.Fatal(err)
		}

		err = group.UpdatePermission(groupName1, common.HexToAddress(addrStr), acl.PermissionWrite)
		if err != nil {
			t.Fatal(err)
		}

		// reopen the group to reload feed with new permission
		err = group2.CloseGroup(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		g, err = group2.OpenGroup(groupName1)
		if err != nil {
			t.Fatal(err)
		}

		_, err = uploadFile(t, g.GetFile(), "/", fileName, compression, g.GetPodPassword(), fileSize, blockSize)
		if err != nil {
			t.Fatal(err)
		}
		err = g.GetDirectory().AddEntryToDir("/", g.GetPodPassword(), fileName, true)
		if err != nil {
			t.Fatal(err)
		}

		dirInode, err := g.GetDirectory().GetInode(g.GetPodPassword(), "/")
		if err != nil {
			t.Fatal(err)
		}
		if len(dirInode.FileOrDirNames) != 1 {
			t.Fatal("files not present")
		}
		if dirInode.FileOrDirNames[0] != "_F_"+fileName {
			t.Fatal("file name not correct")
		}
	})

	t.Run("group-member-remove", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}

		acc2 := account.New(logger)
		_, _, err = acc2.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		addr := acc2.GetUserAccountInfo().GetAddress()
		addrStr := addr.Hex()
		ref, err := group.AddMember(groupName1, common.HexToAddress(addrStr), acc2.GetUserAccountInfo().GetPublicKey(), acl.PermissionRead)
		if err != nil {
			t.Fatal(err)
		}
		fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)

		group2 := pod.NewGroup(mockClient, fd2, acc2, mockAcl, logger)
		err = group2.AcceptGroupInvite(ref)
		if err != nil {
			t.Fatal(err)
		}

		err = group.RemoveMember(groupName1, common.HexToAddress(addrStr))
		if err != nil {
			t.Fatal(err)
		}
		_, err = group2.OpenGroup(groupName1)
		if !errors.Is(err, pod.ErrPermissionDenied) {
			t.Fatal(err)
		}
	})

	t.Run("group-remove", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}

		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}

		acc2 := account.New(logger)
		_, _, err = acc2.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		addr := acc2.GetUserAccountInfo().GetAddress()
		addrStr := addr.Hex()
		ref, err := group.AddMember(groupName1, common.HexToAddress(addrStr), acc2.GetUserAccountInfo().GetPublicKey(), acl.PermissionRead)
		if err != nil {
			t.Fatal(err)
		}
		fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)

		group2 := pod.NewGroup(mockClient, fd2, acc2, mockAcl, logger)
		err = group2.AcceptGroupInvite(ref)
		if err != nil {
			t.Fatal(err)
		}

		err = group.RemoveGroup(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		_, err = group2.OpenGroup(groupName1)
		if !errors.Is(err, pod.ErrPermissionDenied) {
			t.Fatal(err)
		}
		_, err = group2.OpenGroup(groupName1)
		if !errors.Is(err, pod.ErrGroupDoesNotExist) {
			t.Fatal(err)
		}
	})

	t.Run("group-add-multiple-member", func(t *testing.T) {
		fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
		mockAcl := acl.NewACL(mockClient, fd, logger)
		group := pod.NewGroup(mockClient, fd, acc, mockAcl, logger)
		groupName1, _ := utils.GetRandString(10)
		_, err = group.CreateGroup(groupName1)
		if err != nil {
			t.Fatalf("error creating group %s: %s", groupName1, err.Error())
		}

		_, err = group.ListGroup()
		if err != nil {
			t.Fatalf("error getting groups")
		}
		_, err = group.OpenGroup(groupName1)
		if err != nil {
			t.Fatalf("error opening group %s: %s", groupName1, err.Error())
		}
		userCount := 10
		for i := 0; i < userCount; i++ {
			acc2 := account.New(logger)
			_, _, err = acc2.CreateUserAccount("")
			if err != nil {
				t.Fatal(err)
			}
			addr := acc2.GetUserAccountInfo().GetAddress()
			addrStr := addr.Hex()
			_, err = group.AddMember(groupName1, common.HexToAddress(addrStr), acc2.GetUserAccountInfo().GetPublicKey(), acl.PermissionWrite)
			if err != nil {
				t.Fatal(err)
			}
		}
		users, err := group.GetGroupMembers(groupName1)
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != userCount+1 {
			t.Fatal("users not added")
		}
	})

}
