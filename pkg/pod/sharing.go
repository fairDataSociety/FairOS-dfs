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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethersphere/bee/v2/pkg/swarm"

	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

// ShareInfo is the structure of the share info
type ShareInfo struct {
	PodName     string `json:"podName"`
	Address     string `json:"podAddress"`
	Password    string `json:"password"`
	UserAddress string `json:"userAddress"`
}

// PodShare makes a pod public by exporting all the pod related information and its
// address. it does this by creating a sharing reference which points to the information
// required to import this pod.
func (p *Pod) PodShare(podName, sharedPodName string) (string, error) {
	// check if pods is present and get the index of the pod
	podList, err := p.PodList()
	if err != nil { // skipcq: TCV-001
		return "", err
	}
	if !p.checkIfPodPresent(podList, podName) {
		return "", ErrInvalidPodName
	}

	index, podPassword := p.getIndexPassword(podList, podName)
	if index == -1 { // skipcq: TCV-001
		return "", fmt.Errorf("pod does not exist")
	}

	// Create pod account and get the address
	accountInfo, err := p.acc.CreatePodAccount(index, false)
	if err != nil { // skipcq: TCV-001
		return "", err
	}

	address := accountInfo.GetAddress()
	userAddress := p.acc.GetUserAccountInfo().GetAddress()
	if sharedPodName == "" {
		sharedPodName = podName
	}
	shareInfo := &ShareInfo{
		PodName:     sharedPodName,
		Password:    podPassword,
		Address:     address.String(),
		UserAddress: userAddress.String(),
	}

	data, err := json.Marshal(shareInfo)
	if err != nil { // skipcq: TCV-001
		return "", err
	}
	ref, err := p.client.UploadBlob(0, "", "0", false, false, bytes.NewReader(data))
	if err != nil { // skipcq: TCV-001
		return "", err
	}

	return ref.String(), nil
}

// GetPodSharingInfo returns the raw shareInfo
func (p *Pod) GetPodSharingInfo(podName string) (*ShareInfo, error) {
	// check if pods is present and get the index of the pod
	podList, err := p.PodList()
	if err != nil { // skipcq: TCV-001
		return nil, err
	}
	if !p.checkIfPodPresent(podList, podName) {
		return nil, ErrInvalidPodName
	}

	index, podPassword := p.getIndexPassword(podList, podName)
	if index == -1 { // skipcq: TCV-001
		return nil, fmt.Errorf("pod does not exist")
	}

	// Create pod account and get the address
	accountInfo, err := p.acc.CreatePodAccount(index, false)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}

	address := accountInfo.GetAddress()
	userAddress := p.acc.GetUserAccountInfo().GetAddress()

	return &ShareInfo{
		PodName:     podName,
		Password:    podPassword,
		Address:     address.String(),
		UserAddress: userAddress.String(),
	}, nil
}

// ReceivePodInfo returns the shareInfo from the reference
func (p *Pod) ReceivePodInfo(ref utils.Reference) (*ShareInfo, error) {
	r, resp, err := p.client.DownloadBlob(swarm.NewAddress(ref.Bytes()))
	if err != nil { // skipcq: TCV-001
		return nil, err
	}

	if resp != http.StatusOK { // skipcq: TCV-001
		return nil, fmt.Errorf("ReceivePodInfo: could not download blob")
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}
	var shareInfo ShareInfo
	err = json.Unmarshal(data, &shareInfo)
	if err != nil {
		return nil, err
	}

	return &shareInfo, nil
}

// ReceivePod imports a pod by creating a new pod with the same name and password
func (p *Pod) ReceivePod(sharedPodName string, ref utils.Reference) (*Info, error) {
	r, resp, err := p.client.DownloadBlob(swarm.NewAddress(ref.Bytes()))
	if err != nil { // skipcq: TCV-001
		return nil, err
	}
	if resp != http.StatusOK { // skipcq: TCV-001
		return nil, fmt.Errorf("receivePod: could not download blob")
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}
	var shareInfo ShareInfo
	err = json.Unmarshal(data, &shareInfo)
	if err != nil { // skipcq: TCV-001
		return nil, err
	}

	if sharedPodName != "" {
		shareInfo.PodName = sharedPodName
	}
	return p.CreatePod(shareInfo.PodName, shareInfo.Address, shareInfo.Password)
}
