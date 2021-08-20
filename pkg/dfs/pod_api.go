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

package dfs

import (
	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

func (d *DfsAPI) CreatePod(podName, passPhrase, sessionId string) (*pod.Info, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	// create the pod
	pi, err := ui.GetPod().CreatePod(podName, passPhrase, "")
	if err != nil {
		return nil, err
	}

	// open the pod
	_, err = ui.GetPod().OpenPod(podName, passPhrase)
	if err != nil {
		return nil, err
	}

	// create the root directory
	err = ui.GetUserDirectory().MkRootDir(pi.GetPodName(), pi.GetPodAddress(), pi.GetFeed())
	if err != nil {
		return nil, err
	}

	err = ui.GetPod().SyncPod(podName)
	if err != nil {
		return nil, err
	}

	// Add podName in the login user session
	ui.AddPodName(podName, pi)
	return pi, nil
}

func (d *DfsAPI) DeletePod(podName, sessionId string) error {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return ErrUserNotLoggedIn
	}

	// delete the pod and close if it is opened
	err := ui.GetPod().DeletePod(podName)
	if err != nil {
		return err
	}

	// TODO: delete all the directory, files, and database tables under this pod from
	// the Swarm network.

	// close the pod if it is open
	if ui.IsPodOpen(podName) {
		// remove from the login session
		ui.RemovePodName(podName)
	}

	return nil
}

func (d *DfsAPI) OpenPod(podName, passPhrase, sessionId string) (*pod.Info, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	// return if pod already open
	if ui.IsPodOpen(podName) {
		return nil, ErrPodAlreadyOpen
	}

	// open the pod
	pi, err := ui.GetPod().OpenPod(podName, passPhrase)
	if err != nil {
		return nil, err
	}

	err = pi.GetDirectory().AddRootDir(pi.GetPodName(), pi.GetPodAddress(), pi.GetFeed())
	if err != nil {
		return nil, err
	}

	// Add podName in the login user session
	ui.AddPodName(podName, pi)

	return pi, nil
}

func (d *DfsAPI) ClosePod(podName, sessionId string) error {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return ErrUserNotLoggedIn
	}

	// check if pod open
	if !ui.IsPodOpen(podName) {
		return ErrPodNotOpen
	}

	// close the pod
	err := ui.GetPod().ClosePod(podName)
	if err != nil {
		return err
	}

	// delete podName in the login user session
	ui.RemovePodName(podName)
	return nil
}

func (d *DfsAPI) PodStat(podName, sessionId string) (*pod.PodStat, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	// get the pod stat
	podStat, err := ui.GetPod().PodStat(podName)
	if err != nil {
		return nil, err
	}
	return podStat, nil
}

func (d *DfsAPI) SyncPod(podName, sessionId string) error {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return ErrUserNotLoggedIn
	}

	// check if pod open
	if !ui.IsPodOpen(podName) {
		return ErrPodNotOpen
	}

	// sync the pod
	err := ui.GetPod().SyncPod(podName)
	if err != nil {
		return err
	}
	return nil
}

func (d *DfsAPI) ListPods(sessionId string) ([]string, []string, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, nil, ErrUserNotLoggedIn
	}

	// list pods of a user
	pods, sharedPods, err := ui.GetPod().ListPods()
	if err != nil {
		return nil, nil, err
	}
	return pods, sharedPods, nil
}

func (d *DfsAPI) PodShare(podName, passPhrase, sessionId string) (string, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return "", ErrUserNotLoggedIn
	}

	// get the pod stat
	address, err := ui.GetPod().PodShare(podName, passPhrase, ui.GetUserName())
	if err != nil {
		return "", err
	}
	return address, nil
}

func (d *DfsAPI) PodReceiveInfo(sessionId string, ref utils.Reference) (*pod.ShareInfo, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	return ui.GetPod().ReceivePodInfo(ref)
}

func (d *DfsAPI) PodReceive(sessionId string, ref utils.Reference) (*pod.Info, error) {
	// get the logged in user information
	ui := d.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	return ui.GetPod().ReceivePod(ref)
}
