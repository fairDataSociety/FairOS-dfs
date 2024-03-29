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

package pod

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	c "github.com/fairdatasociety/fairOS-dfs/pkg/collection"
	d "github.com/fairdatasociety/fairOS-dfs/pkg/dir"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	f "github.com/fairdatasociety/fairOS-dfs/pkg/file"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

const (
	podFile   = "Pods"
	podFileV2 = "PodsV2"
)

// CreatePod creates a new pod for a given user.
func (p *Pod) CreatePod(podName, addressString, podPassword string) (*Info, error) {
	podName, err := CleanPodName(podName)
	if err != nil {
		return nil, err
	}

	// check if pods is present and get free index
	podList, err := p.PodList()
	if err != nil { // skipcq: TCV-001
		return nil, err
	}
	pods := map[int]string{}
	sharedPods := map[string]string{}
	for _, pod := range podList.Pods {
		pods[pod.Index] = pod.Name
	}
	for _, pod := range podList.SharedPods {
		sharedPods[pod.Address] = pod.Name
	}

	var accountInfo *account.Info
	var fd *feed.API
	var file *f.File
	var dir *d.Directory
	var user utils.Address
	if addressString != "" {
		if p.checkIfPodPresent(podList, podName) {
			return nil, ErrPodAlreadyExists
		}
		if p.checkIfSharedPodPresent(podList, podName) {
			return nil, ErrPodAlreadyExists
		}

		// shared pod, so add only address to the account info
		accountInfo = p.acc.GetEmptyAccountInfo()
		address := utils.HexToAddress(addressString)
		accountInfo.SetAddress(address)

		fd = feed.New(accountInfo, p.client, p.feedCacheSize, p.feedCacheTTL, p.logger)
		file = f.NewFile(podName, p.client, fd, accountInfo.GetAddress(), p.tm, p.logger)
		dir = d.NewDirectory(podName, p.client, fd, accountInfo.GetAddress(), file, p.tm, p.logger)

		// store the pod file with shared pod
		sharedPod := &SharedListItem{
			Name:     podName,
			Address:  addressString,
			Password: podPassword,
		}
		podList.SharedPods = append(podList.SharedPods, *sharedPod)
		err = p.storeUserPodsV2(podList)
		if err != nil { // skipcq: TCV-001
			return nil, err
		}

		// set the userAddress as the pod address we got from shared pod
		user = address
	} else {
		// your own pod, so create a new account with private key
		if p.checkIfPodPresent(podList, podName) {
			return nil, ErrPodAlreadyExists
		}
		if p.checkIfSharedPodPresent(podList, podName) {
			return nil, ErrPodAlreadyExists
		}
		freeId, err := p.getFreeId(pods)
		if err != nil { // skipcq: TCV-001
			return nil, err
		}
		// create a child account for the userAddress and other data structures for the pod
		accountInfo, err = p.acc.CreatePodAccount(freeId, true)
		if err != nil { // skipcq: TCV-001
			return nil, err
		}
		fd = feed.New(accountInfo, p.client, p.feedCacheSize, p.feedCacheTTL, p.logger)
		//fd.SetUpdateTracker(p.fd.GetUpdateTracker())
		file = f.NewFile(podName, p.client, fd, accountInfo.GetAddress(), p.tm, p.logger)
		dir = d.NewDirectory(podName, p.client, fd, accountInfo.GetAddress(), file, p.tm, p.logger)
		// store the pod file
		pods[freeId] = podName
		pod := &ListItem{
			Name:     podName,
			Index:    freeId,
			Password: podPassword,
		}
		podList.Pods = append(podList.Pods, *pod)
		err = p.storeUserPodsV2(podList)
		if err != nil { // skipcq: TCV-001
			return nil, err
		}
		user = p.acc.GetAddress(freeId)
	}

	kvStore := c.NewKeyValueStore(podName, fd, accountInfo, user, p.client, p.logger)
	docStore := c.NewDocumentStore(podName, fd, accountInfo, user, file, p.tm, p.client, p.logger)

	// create the pod info and store it in the podMap
	podInfo := &Info{
		podName:     podName,
		podPassword: podPassword,
		userAddress: user,
		dir:         dir,
		file:        file,
		accountInfo: accountInfo,
		feed:        fd,
		kvStore:     kvStore,
		docStore:    docStore,
	}
	p.addPodToPodMap(podName, podInfo)
	if addressString == "" {
		// create the root directory
		err = podInfo.GetDirectory().MkRootDir(podInfo.GetPodName(), podPassword, podInfo.GetPodAddress(), podInfo.GetFeed())
		if err != nil {
			return nil, err
		}
	}
	return podInfo, nil
}

func (p *Pod) loadUserPods() (*List, error) {
	// The userAddress pod file topic should be in the name of the userAddress account
	topic := utils.HashString(podFile)
	privKeyBytes := crypto.FromECDSA(p.acc.GetUserAccountInfo().GetPrivateKey())
	_, data, err := p.fd.GetFeedData(topic, p.acc.GetAddress(account.UserAccountIndex), []byte(hex.EncodeToString(privKeyBytes)), false)
	if err != nil { // skipcq: TCV-001
		if err.Error() != "feed does not exist or was not updated yet" {
			return nil, err
		}
	}
	podList := &List{
		Pods:       []ListItem{},
		SharedPods: []SharedListItem{},
	}
	if len(data) == 0 {
		return podList, nil
	}

	err = json.Unmarshal(data, podList)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}

	return podList, nil
}

func (p *Pod) loadUserPodsV2() (*List, error) {
	f2 := f.NewFile("", p.client, p.fd, p.acc.GetAddress(account.UserAccountIndex), p.tm, p.logger)
	topicString := utils.CombinePathAndFile("", podFileV2)
	privKeyBytes := crypto.FromECDSA(p.acc.GetUserAccountInfo().GetPrivateKey())
	r, _, err := f2.Download(topicString, hex.EncodeToString(privKeyBytes))
	if err != nil { // skipcq: TCV-001
		return nil, err
	}
	podList := &List{
		Pods:       []ListItem{},
		SharedPods: []SharedListItem{},
	}
	data, err := io.ReadAll(r)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}

	if len(data) == 0 {
		return podList, nil
	}

	err = json.Unmarshal(data, podList)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}

	return podList, nil
}

func (p *Pod) storeUserPodsV2(podList *List) error {
	fmt.Println("storeUserPodsV2")
	data, err := json.Marshal(podList)
	if err != nil {
		return err
	}

	// store data as file and get metadata
	// This is a very hacky way to store pod data, but it works for now
	// We create a new file object with the user account address and upload the data
	// We use the user private key to encrypt data.
	f2 := f.NewFile("", p.client, p.fd, p.acc.GetAddress(account.UserAccountIndex), p.tm, p.logger)
	privKeyBytes := crypto.FromECDSA(p.acc.GetUserAccountInfo().GetPrivateKey())
	return f2.Upload(bytes.NewReader(data), podFileV2, int64(len(data)), f.MinBlockSize, 0, "/", "gzip", hex.EncodeToString(privKeyBytes))
}

func (*Pod) getFreeId(pods map[int]string) (int, error) {
	for i := 0; i < maxPodId; i++ {
		if _, ok := pods[i]; !ok {
			if i == 0 {
				// this is the root account patch id
				continue
			}
			return i, nil
		}
	}
	return 0, ErrMaxPodsReached // skipcq: TCV-001
}

func (*Pod) checkIfPodPresent(pods *List, podName string) bool {
	for _, pod := range pods.Pods {
		if pod.Name == podName {
			return true
		}
	}
	return false
}

func (*Pod) checkIfSharedPodPresent(pods *List, podName string) bool {
	for _, pod := range pods.SharedPods {
		if pod.Name == podName {
			return true
		}
	}
	return false
}

func (p *Pod) getPodIndex(podName string) (podIndex int, err error) {
	podList, err := p.PodList()
	if err != nil {
		return -1, err
	} // skipcq: TCV-001
	podIndex = -1
	for _, pod := range podList.Pods {
		if pod.Name == podName {
			podIndex = pod.Index
			return
		}
	}
	return
}
