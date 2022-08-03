package user

import (
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
)

// MigrateUser migrates a user credential from local storage to the Swarm network.
// Deletes local information. It also deletes previous mnemonic and stores it in secondary location
// Logs him out if he is logged in.
func (u *Users) MigrateUser(oldUsername, newUsername, dataDir, password, sessionId string, client blockstore.Client, ui *Info) error {
	// check if session id and user address present in map
	if !u.IsUserLoggedIn(sessionId) {
		return ErrUserNotLoggedIn
	}
	if newUsername == "" {
		newUsername = oldUsername
	}
	// username availability
	if !u.IsUsernameAvailable(oldUsername, dataDir) {
		return ErrInvalidUserName
	}

	// username availability for v2
	if u.IsUsernameAvailableV2(newUsername) {
		return ErrUserAlreadyPresent
	}

	// check for valid password
	userInfo := u.getUserFromMap(sessionId)
	acc := userInfo.account
	if !acc.Authorise(password) { // skipcq: TCV-001
		return ErrInvalidPassword
	}
	accountInfo := acc.GetUserAccountInfo()

	// create ens subdomain and store mnemonic
	_, err := u.createENS(newUsername, accountInfo)
	if err != nil { // skipcq: TCV-001
		return err
	}
	// load address from userName
	address, err := u.getAddressFromUserName(oldUsername, dataDir)
	if err != nil {
		return err
	}

	fd := feed.New(accountInfo, client, u.logger)
	encryptedMnemonic, err := u.getEncryptedMnemonic(oldUsername, address, fd)
	if err != nil { // skipcq: TCV-001
		return err
	}
	err = acc.LoadUserAccount(password, encryptedMnemonic)
	if err != nil { // skipcq: TCV-001
		return err
	}

	seed, err := acc.GetWallet().LoadSeedFromMnemonic(password)
	if err != nil {
		return err
	}
	key, err := accountInfo.PadSeed(seed, password)
	if err != nil { // skipcq: TCV-001
		return err
	}
	if err := u.uploadPortableAccount(accountInfo, newUsername, password, key, fd); err != nil { // skipcq: TCV-001
		return err
	}

	// Logout user
	err = u.Logout(sessionId)
	if err != nil { // skipcq: TCV-001
		return err
	}

	err = u.deleteMnemonic(oldUsername, accountInfo.GetAddress(), ui.GetFeed(), u.client)
	if err != nil { // skipcq: TCV-001
		return err
	}

	return u.deleteUserMapping(oldUsername, dataDir)
}
