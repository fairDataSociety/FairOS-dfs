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
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	mockpost "github.com/ethersphere/bee/v2/pkg/postage/mock"
	mockstorer "github.com/ethersphere/bee/v2/pkg/storer/mock"

	"github.com/fairdatasociety/fairOS-dfs/pkg/file"
	"github.com/sirupsen/logrus"

	mock2 "github.com/fairdatasociety/fairOS-dfs/pkg/subscriptionManager/rpc/mock"

	"github.com/plexsysio/taskmanager"

	"github.com/asabya/swarm-blockstore/bee"
	"github.com/asabya/swarm-blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

func TestShare(t *testing.T) {
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
	fd := feed.New(acc.GetUserAccountInfo(), mockClient, -1, 0, logger)
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	defer func() {
		_ = tm.Stop(context.Background())
	}()
	sm := mock2.NewMockSubscriptionManager()

	pod1 := pod.NewPod(mockClient, fd, acc, tm, sm, -1, 0, logger)
	podName1 := "test1"

	acc2 := account.New(logger)
	_, _, err = acc2.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}
	fd2 := feed.New(acc2.GetUserAccountInfo(), mockClient, -1, 0, logger)
	pod2 := pod.NewPod(mockClient, fd2, acc2, tm, sm, -1, 0, logger)
	podName2 := "test2"

	acc3 := account.New(logger)
	_, _, err = acc3.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}
	fd3 := feed.New(acc3.GetUserAccountInfo(), mockClient, -1, 0, logger)
	pod3 := pod.NewPod(mockClient, fd3, acc3, tm, sm, -1, 0, logger)
	podName3 := "test3"

	acc4 := account.New(logger)
	_, _, err = acc4.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}
	fd4 := feed.New(acc4.GetUserAccountInfo(), mockClient, -1, 0, logger)
	pod4 := pod.NewPod(mockClient, fd4, acc4, tm, sm, -1, 0, logger)
	podName4 := "test4"

	acc5 := account.New(logger)
	_, _, err = acc5.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}
	fd5 := feed.New(acc5.GetUserAccountInfo(), mockClient, -1, 0, logger)
	pod5 := pod.NewPod(mockClient, fd5, acc5, tm, sm, -1, 0, logger)
	podName5 := "test5"

	acc6 := account.New(logger)
	_, _, err = acc6.CreateUserAccount("")
	if err != nil {
		t.Fatal(err)
	}
	fd6 := feed.New(acc6.GetUserAccountInfo(), mockClient, -1, 0, logger)
	pod6 := pod.NewPod(mockClient, fd6, acc6, tm, sm, -1, 0, logger)
	podName6 := "test6"

	t.Run("share-pod", func(t *testing.T) {
		_, err := pod1.PodShare(podName1, "")
		if err == nil {
			t.Fatal("pod share should fail, not exists")
		}
		// create a pod
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		info, err := pod1.CreatePod(podName1, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName1)
		}

		// make root dir so that other directories can be added
		err = info.GetDirectory().MkRootDir("pod1", podPassword, info.GetPodAddress(), info.GetFeed())
		if err != nil {
			t.Fatal(err)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod1, podName1, podPassword)

		// share pod
		sharingRef, err := pod1.PodShare(podName1, "")
		if err != nil {
			t.Fatal(err)
		}

		// verify if pod is shared
		if sharingRef == "" {
			t.Fatalf("could not share pod")
		}
	})

	t.Run("share-pod-with-new-name", func(t *testing.T) {
		// create a pod
		podName01 := "test_1"
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		info, err := pod1.CreatePod(podName01, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName01)
		}

		// make root dir so that other directories can be added
		err = info.GetDirectory().MkRootDir("", podPassword, info.GetPodAddress(), info.GetFeed())
		if err != nil {
			t.Fatal(err)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod1, podName01, podPassword)

		// share pod
		sharedPodName := "test01"
		sharingRef, err := pod1.PodShare(podName01, sharedPodName)
		if err != nil {
			t.Fatal(err)
		}

		// verify if pod is shared
		if sharingRef == "" {
			t.Fatalf("could not share pod")
		}

		// receive pod info for name validation
		ref, err := utils.ParseHexReference(sharingRef)
		if err != nil {
			t.Fatal(err)
		}
		sharingInfo, err := pod1.ReceivePodInfo(ref)
		if err != nil {
			t.Fatal(err)
		}

		// verify the pod info
		if sharingInfo == nil {
			t.Fatalf("could not receive sharing info")
		}
		if sharingInfo.PodName != sharedPodName {
			t.Fatalf("invalid pod name received")
		}
	})

	t.Run("receive-pod-info", func(t *testing.T) {
		// create a pod
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		info, err := pod2.CreatePod(podName2, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName2)
		}

		// make root dir so that other directories can be added
		err = info.GetDirectory().MkRootDir("pod1", podPassword, info.GetPodAddress(), info.GetFeed())
		if err != nil {
			t.Fatal(err)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod2, podName2, podPassword)

		// share pod
		sharingRef, err := pod2.PodShare(podName2, "")
		if err != nil {
			t.Fatal(err)
		}

		// receive pod info
		ref, err := utils.ParseHexReference(sharingRef)
		if err != nil {
			t.Fatal(err)
		}
		sharingInfo, err := pod2.ReceivePodInfo(ref)
		if err != nil {
			t.Fatal(err)
		}

		// verify the pod info
		if sharingInfo == nil {
			t.Fatalf("could not receive sharing info")
		}
		if sharingInfo.PodName != podName2 {
			t.Fatalf("invalid pod name received")
		}
	})

	t.Run("receive-pod", func(t *testing.T) {
		// create sending pod and receiving pod
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		info, err := pod3.CreatePod(podName3, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName3)
		}
		_, err = pod4.GetAccountInfo(podName4)
		if err == nil {
			t.Fatalf("GetAccountInfo for pod4 should fail")
		}
		pi4, err := pod4.CreatePod(podName4, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName4)
		}

		pod4Present := pod4.IsPodPresent(podName4)
		if !pod4Present {
			t.Fatal("pod4 should be present")
		}
		pod5Present := pod4.IsPodPresent(podName5)
		if pod5Present {
			t.Fatal("pod5 should not be present")
		}
		pi, err := pod4.GetAccountInfo(podName4)
		if err != nil {
			t.Fatalf("error getting info of pod %s", podName4)
		}
		if pi.GetAddress() != pi4.GetAccountInfo().GetAddress() {
			t.Fatalf("pod4 address does not match")
		}
		// make root dir so that other directories can be added
		err = info.GetDirectory().MkRootDir("", podPassword, info.GetPodAddress(), info.GetFeed())
		if err != nil {
			t.Fatal(err)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod3, podName3, podPassword)

		// share pod
		sharingRef, err := pod3.PodShare(podName3, "")
		if err != nil {
			t.Fatal(err)
		}

		// receive pod info
		ref, err := utils.ParseHexReference(sharingRef)
		if err != nil {
			t.Fatal(err)
		}
		podInfo, err := pod4.ReceivePod("", ref)
		if err != nil {
			t.Fatal(err)
		}
		pod3Present := pod4.IsPodPresent(podName3)
		if !pod3Present {
			t.Fatal("pod3 should be present")
		}

		podInfo2, err := pod4.OpenPod(podName3)
		if err != nil {
			t.Fatal(err)
		}
		if podInfo.GetPodName() != podInfo2.GetPodName() {
			t.Fatal("pod infos do not match")
		}
		// verify the pod info
		if podInfo == nil {
			t.Fatalf("could not receive sharing info")
		}
		if podInfo.GetPodName() != podName3 {
			t.Fatalf("invalid pod name received")
		}

		pods, sharedPods, err := pod4.ListPods()
		if err != nil {
			t.Fatal(err)
		}
		if pods == nil {
			t.Fatalf("invalid pods")
		}
		if len(pods) != 1 && pods[0] != podName1 {
			t.Fatalf("invalid pod name")
		}
		if sharedPods == nil {
			t.Fatalf("invalid shared pods")
		}
		if len(sharedPods) != 1 && sharedPods[0] != podName4 {
			t.Fatalf("invalid pod name")
		}
		podPassword, _ = utils.GetRandString(pod.PasswordLength)
		_, err = pod4.CreatePod(podName4, ref.String(), podPassword)
		if !errors.Is(err, pod.ErrPodAlreadyExists) {
			t.Fatal("pod should exist")
		}
		podPassword, _ = utils.GetRandString(pod.PasswordLength)
		_, err = pod4.CreatePod(podName3, ref.String(), podPassword)
		if !errors.Is(err, pod.ErrPodAlreadyExists) {
			t.Fatal("shared pod should exist")
		}
		podPassword, _ = utils.GetRandString(pod.PasswordLength)
		_, err = pod4.CreatePod(podName4, "", podPassword)
		if !errors.Is(err, pod.ErrPodAlreadyExists) {
			t.Fatal("pod should exist")
		}
		podPassword, _ = utils.GetRandString(pod.PasswordLength)
		_, err = pod4.CreatePod(podName3, "", podPassword)
		if !errors.Is(err, pod.ErrPodAlreadyExists) {
			t.Fatal("shared pod should exist")
		}

		err = pod4.DeleteSharedPod(podName3)
		if err != nil {
			t.Fatal(err)
		}

		err = pod4.DeleteSharedPod(podName3)
		if err == nil {
			t.Fatal("pod should have been deleted")
		}
	})

	t.Run("receive-pod-with-new-name", func(t *testing.T) {
		// create sending pod and receiving pod
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		info, err := pod5.CreatePod(podName5, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName5)
		}
		podPassword, _ = utils.GetRandString(pod.PasswordLength)
		_, err = pod6.CreatePod(podName6, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName6)
		}

		// make root dir so that other directories can be added
		err = info.GetDirectory().MkRootDir("", podPassword, info.GetPodAddress(), info.GetFeed())
		if err != nil {
			t.Fatal(err)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod5, podName5, podPassword)

		// share pod
		sharingRef, err := pod5.PodShare(podName5, "")
		if err != nil {
			t.Fatal(err)
		}

		// receive pod info
		ref, err := utils.ParseHexReference(sharingRef)
		if err != nil {
			t.Fatal(err)
		}
		sharedPodName2 := "test02"
		podInfo, err := pod6.ReceivePod(sharedPodName2, ref)
		if err != nil {
			t.Fatal(err)
		}

		// verify the pod info
		if podInfo == nil {
			t.Fatalf("could not receive sharing info")
		}
		if podInfo.GetPodName() != sharedPodName2 {
			t.Fatalf("invalid pod name received")
		}

		pods, sharedPods, err := pod6.ListPods()
		if err != nil {
			t.Fatal(err)
		}
		if pods == nil {
			t.Fatalf("invalid pods")
		}
		if len(pods) != 1 && pods[0] != podName6 {
			t.Fatalf("invalid pod name")
		}
		if sharedPods == nil {
			t.Fatalf("invalid shared pods")
		}
		if len(sharedPods) != 1 {
			t.Fatalf("invalid shared pod count")
		}
		if sharedPods[0] != sharedPodName2 {
			t.Fatalf("invalid shared pod name")
		}
	})

	t.Run("check-updates-on-received-pod", func(t *testing.T) {
		acc7 := account.New(logger)
		_, _, err = acc7.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		fd7 := feed.New(acc7.GetUserAccountInfo(), mockClient, -1, 0, logger)
		pod7 := pod.NewPod(mockClient, fd7, acc7, tm, sm, -1, 0, logger)
		podName7 := "test7"

		acc8 := account.New(logger)
		_, _, err = acc8.CreateUserAccount("")
		if err != nil {
			t.Fatal(err)
		}
		fd8 := feed.New(acc8.GetUserAccountInfo(), mockClient, -1, 0, logger)
		pod8 := pod.NewPod(mockClient, fd8, acc8, tm, sm, -1, 0, logger)

		// create sending pod and receiving pod
		podPassword, _ := utils.GetRandString(pod.PasswordLength)
		info, err := pod7.CreatePod(podName7, "", podPassword)
		if err != nil {
			t.Fatalf("error creating pod %s", podName7)
		}

		// make root dir so that other directories can be added
		err = info.GetDirectory().MkRootDir("", podPassword, info.GetPodAddress(), info.GetFeed())
		if err != nil {
			t.Fatal(err)
		}

		// create some dir and files
		addFilesAndDirectories(t, info, pod7, podName7, podPassword)

		// share pod
		sharingRef, err := pod7.PodShare(podName7, "")
		if err != nil {
			t.Fatal(err)
		}

		pod7.GetFeed().CommitFeeds()
		info.GetFeed().CommitFeeds()
		// receive pod info
		ref, err := utils.ParseHexReference(sharingRef)
		if err != nil {
			t.Fatal(err)
		}

		podInfo, err := pod8.ReceivePod("", ref)
		if err != nil {
			t.Fatal(err)
		}

		// verify the pod info
		if podInfo == nil {
			t.Fatalf("could not receive sharing info")
		}
		if podInfo.GetPodName() != podName7 {
			t.Fatalf("invalid pod name received")
		}

		// now change original pod content

		// open the pod ths triggers sync too
		gotInfo, err := pod7.OpenPod(podName7)
		if err != nil {
			t.Fatal(err)
		}

		// validate if the directory and files are synced
		dirObject := gotInfo.GetDirectory()
		err = dirObject.RenameDir("/parentDir/subDir1", "/parentDir/newSubDir1", podPassword)
		if err != nil {
			t.Fatal(err)
		}
		fileObject := gotInfo.GetFile()
		_, err = fileObject.RenameFromFileName("/parentDir/file1", "/parentDir/renamedFile1", podPassword)
		if err != nil {
			t.Fatal(err)
		}

		pod7.GetFeed().CommitFeeds()
		gotInfo.GetFeed().CommitFeeds()
		// check shared pod entry
		gotSharedPodInfo, err := pod8.OpenPod(podName7)
		if err != nil {
			t.Fatal(err)
		}
		dirObject8 := gotSharedPodInfo.GetDirectory()
		dirInode1, _ := dirObject8.GetInode(podPassword, "/parentDir/subDir1")
		if dirInode1 != nil {
			t.Fatalf("invalid dir entry")
		}
		dirInode1, _ = dirObject8.GetInode(podPassword, "/parentDir/newSubDir1")
		if dirInode1 == nil {
			t.Fatalf("invalid dir entry")
		}
		if dirInode1.Meta.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if dirInode1.Meta.Name != "newSubDir1" {
			t.Fatalf("invalid dir entry")
		}
		dirInode2, _ := dirObject8.GetInode(podPassword, "/parentDir/subDir2")
		if dirInode2 == nil {
			t.Fatalf("invalid dir entry")
		}
		if dirInode2.Meta.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if dirInode2.Meta.Name != "subDir2" {
			t.Fatalf("invalid dir entry")
		}

		fileObject8 := gotInfo.GetFile()
		fileMeta1 := fileObject8.GetInode(podPassword, "/parentDir/file1")
		if fileMeta1 != nil {
			t.Fatalf("invalid file meta")
		}
		fileMeta1 = fileObject8.GetInode(podPassword, "/parentDir/renamedFile1")
		if fileMeta1 == nil {
			t.Fatalf("invalid file meta")
		}
		if fileMeta1.Path != "/parentDir" {
			t.Fatalf("invalid path entry")
		}
		if fileMeta1.Name != "renamedFile1" {
			t.Fatalf("invalid file entry")
		}
		if fileMeta1.Size != uint64(100) {
			t.Fatalf("invalid file size")
		}
		if fileMeta1.BlockSize != file.MinBlockSize {
			t.Fatalf("invalid block size")
		}
		fileMeta2 := fileObject.GetInode(podPassword, "/parentDir/file2")
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
		if fileMeta2.BlockSize != file.MinBlockSize {
			t.Fatalf("invalid block size")
		}
	})
}
