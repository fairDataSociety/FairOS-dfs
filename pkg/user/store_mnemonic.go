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
	"math/rand"
	"time"
	"unsafe"

	"github.com/ethersphere/bee/pkg/swarm"
	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

func randStringBytesMaskImprSrcUnsafe(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func (u *Users) uploadEncryptedMnemonicSOC(accountInfo *account.Info, encryptedMnemonic string, fd *feed.API) ([]byte, error) {
	topic := utils.HashString(randStringBytesMaskImprSrcUnsafe(16))
	return fd.CreateFeed(topic, accountInfo.GetAddress(), []byte(encryptedMnemonic))
}

func (u *Users) uploadSecondaryLocationInformation(accountInfo *account.Info, encryptedAddress, encryptedPublicKey string, fd *feed.API) error {
	topic := utils.HashString(encryptedPublicKey)
	_, err := fd.CreateFeed(topic, accountInfo.GetAddress(), []byte(encryptedAddress))
	if err != nil {
		return err
	}
	return err
}

func (u *Users) getSecondaryLocationInformation(address utils.Address, encryptedPublicKey string, fd *feed.API) (string, error) {
	topic := utils.HashString(encryptedPublicKey)
	_, data, err := fd.GetFeedData(topic, address)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (*Users) getEncryptedMnemonic(address []byte, fd *feed.API) ([]byte, error) {
	return fd.GetFeedDataFromAddress(address)
}

func (*Users) deleteMnemonic(userName string, address utils.Address, fd *feed.API, client blockstore.Client) error {
	topic := utils.HashString(userName)
	feedAddress, _, err := fd.GetFeedData(topic, address)
	if err != nil {
		return err
	}
	return client.DeleteReference(feedAddress)
}

// toSignDigest creates a digest suitable for signing to represent the soc.
func toSignDigest(id, sum []byte) ([]byte, error) {
	h := swarm.NewHasher()
	_, err := h.Write(id)
	if err != nil {
		return nil, err
	}
	_, err = h.Write(sum)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
