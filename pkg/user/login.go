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

package user

import (
	"sync"

	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore"
	"github.com/fairdatasociety/fairOS-dfs/pkg/cookie"
	d "github.com/fairdatasociety/fairOS-dfs/pkg/dir"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	f "github.com/fairdatasociety/fairOS-dfs/pkg/file"
	p "github.com/fairdatasociety/fairOS-dfs/pkg/pod"
)

// LoginUser checks if the user is present and logs in the user. It also creates the required information
// to execute user function and stores it in memory.
func (u *Users) LoginUser(userName, passPhrase string, client blockstore.Client, sessionId string) (*Info, error) {
	// check if username is available (user created)
	if !u.IsUsernameAvailable(userName) {
		return nil, ErrInvalidUserName
	}

	// create account
	acc := account.New(u.logger)
	accountInfo := acc.GetUserAccountInfo()

	// load address from userName
	address, err := u.fnm.GetOwner(userName)
	if err != nil {
		return nil, err
	}

	err = u.fnm.GetAll(userName)
	if err != nil {
		return nil, err
	}

	// load encrypted mnemonic from Swarm
	fd := feed.New(accountInfo, client, u.logger)
	encryptedMnemonic, err := u.getEncryptedMnemonic(userName, utils.Address(address), fd)
	if err != nil {
		return nil, err
	}

	err = acc.LoadUserAccount(passPhrase, encryptedMnemonic)
	if err != nil {
		if err.Error() == "mnemonic is invalid" {
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	if u.IsUserLoggedIn(sessionId) {
		return nil, ErrUserAlreadyLoggedIn
	}

	// Instantiate pod, dir & file objects
	file := f.NewFile(userName, client, fd, accountInfo.GetAddress(), u.logger)
	dir := d.NewDirectory(userName, client, fd, accountInfo.GetAddress(), file, u.logger)
	pod := p.NewPod(u.client, fd, acc, u.logger)
	if sessionId == "" {
		sessionId = cookie.GetUniqueSessionId()
	}

	ui := &Info{
		name:       userName,
		sessionId:  sessionId,
		feedApi:    fd,
		account:    acc,
		file:       file,
		dir:        dir,
		pod:        pod,
		openPods:   make(map[string]*p.Info),
		openPodsMu: &sync.RWMutex{},
	}

	// set cookie and add user to map
	return ui, u.addUserAndSessionToMap(ui)
}

func (u *Users) addUserAndSessionToMap(ui *Info) error {
	u.addUserToMap(ui)
	return nil
}

// Logout removes the user information from all the data structures and clears the cookie.
func (u *Users) Logout(sessionId string) error {
	// check if session or user present in map
	if !u.isUserPresentInMap(sessionId) {
		return ErrUserNotLoggedIn
	}

	// remove from the user map
	u.removeUserFromMap(sessionId)

	return nil
}

func (u *Users) IsUserLoggedIn(sessionId string) bool {
	return u.isUserPresentInMap(sessionId)
}

func (u *Users) GetLoggedInUserInfo(sessionId string) *Info {
	return u.getUserFromMap(sessionId)
}

func (u *Users) IsUserNameLoggedIn(userName string) bool {
	return u.isUserNameInMap(userName)
}
