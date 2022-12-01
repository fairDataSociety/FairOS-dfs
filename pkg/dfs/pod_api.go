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
	"context"
	"encoding/hex"

	"github.com/fairdatasociety/fairOS-dfs/pkg/pod"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

func (a *API) CreatePod(podName, sessionId string) (*pod.Info, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}
	podPasswordBytes, _ := utils.GetRandBytes(pod.PodPasswordLength)
	podPassword := hex.EncodeToString(podPasswordBytes)
	// create the pod
	_, err := ui.GetPod().CreatePod(podName, "", podPassword)
	if err != nil {
		return nil, err
	}

	// open the pod
	pi, err := ui.GetPod().OpenPod(podName)
	if err != nil {
		return nil, err
	}

	// create the root directory
	err = pi.GetDirectory().MkRootDir(pi.GetPodName(), podPassword, pi.GetPodAddress(), pi.GetFeed())
	if err != nil {
		return nil, err
	}

	// Add podName in the login user session
	ui.AddPodName(podName, pi)
	return pi, nil
}

// DeletePod deletes a pod
func (a *API) DeletePod(podName, sessionId string) error {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return ErrUserNotLoggedIn
	}

	// delete all the directory, files, and database tables under this pod from
	// the Swarm network.
	podInfo, _, err := ui.GetPod().GetPodInfoFromPodMap(podName)
	if err != nil {
		return err
	}
	directory := podInfo.GetDirectory()

	// check if this is a shared pod
	if podInfo.GetFeed().IsReadOnlyFeed() {
		// delete the pod and close if it is opened
		err = ui.GetPod().DeleteSharedPod(podName)
		if err != nil {
			return err
		}

		// close the pod if it is open
		if ui.IsPodOpen(podName) {
			// remove from the login session
			ui.RemovePodName(podName)
		}
		return nil
	}

	err = directory.RmRootDir(podInfo.GetPodPassword())
	if err != nil {
		return err
	}

	// delete the pod and close if it is opened
	err = ui.GetPod().DeleteOwnPod(podName)
	if err != nil {
		return err
	}

	// close the pod if it is open
	if ui.IsPodOpen(podName) {
		// remove from the login session
		ui.RemovePodName(podName)
	}

	return nil
}

func (a *API) OpenPod(podName, sessionId string) (*pod.Info, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}
	// return if pod already open
	if ui.IsPodOpen(podName) {
		podInfo, _, err := ui.GetPod().GetPodInfoFromPodMap(podName)
		if err != nil {
			return nil, err
		}
		return podInfo, nil
	}
	// open the pod
	pi, err := ui.GetPod().OpenPod(podName)
	if err != nil {
		return nil, err
	}
	err = pi.GetDirectory().AddRootDir(pi.GetPodName(), pi.GetPodPassword(), pi.GetPodAddress(), pi.GetFeed())
	if err != nil {
		return nil, err
	}
	// Add podName in the login user session
	ui.AddPodName(podName, pi)
	return pi, nil
}

func (a *API) OpenPodAsync(ctx context.Context, podName, sessionId string) (*pod.Info, error) {
	// get the logged-in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}
	// return if pod already open
	if ui.IsPodOpen(podName) {
		podInfo, _, err := ui.GetPod().GetPodInfoFromPodMap(podName)
		if err != nil {
			return nil, err
		}
		return podInfo, nil
	}
	// open the pod
	pi, err := ui.GetPod().OpenPodAsync(ctx, podName)
	if err != nil {
		return nil, err
	}
	err = pi.GetDirectory().AddRootDir(pi.GetPodName(), pi.GetPodPassword(), pi.GetPodAddress(), pi.GetFeed())
	if err != nil {
		return nil, err
	}
	// Add podName in the login user session
	ui.AddPodName(podName, pi)
	return pi, nil
}

func (a *API) ClosePod(podName, sessionId string) error {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
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

func (a *API) PodStat(podName, sessionId string) (*pod.Stat, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
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

func (a *API) SyncPod(podName, sessionId string) error {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
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

func (a *API) ListPods(sessionId string) ([]string, []string, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
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

// PodList lists all available pods in json format
func (a *API) PodList(sessionId string) (*pod.PodList, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	// list pods of a user
	return ui.GetPod().PodList()
}

func (a *API) PodShare(podName, sharedPodName, sessionId string) (string, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return "", ErrUserNotLoggedIn
	}

	// get the pod stat
	address, err := ui.GetPod().PodShare(podName, sharedPodName)
	if err != nil {
		return "", err
	}
	return address, nil
}

func (a *API) PodReceiveInfo(sessionId string, ref utils.Reference) (*pod.ShareInfo, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	return ui.GetPod().ReceivePodInfo(ref)
}

func (a *API) PodReceive(sessionId, sharedPodName string, ref utils.Reference) (*pod.Info, error) {
	// get the logged in user information
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return nil, ErrUserNotLoggedIn
	}

	return ui.GetPod().ReceivePod(sharedPodName, ref)
}

func (a *API) IsPodExist(podName, sessionId string) bool {
	ui := a.users.GetLoggedInUserInfo(sessionId)
	if ui == nil {
		return false
	}
	return ui.GetPod().IsPodPresent(podName)
}
