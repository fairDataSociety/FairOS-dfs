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

package user_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/user"
)

func TestExport(t *testing.T) {
	mockClient := mock.NewMockBeeClient()
	logger := logging.New(ioutil.Discard, 0)

	t.Run("export-user", func(t *testing.T) {
		dataDir, err := ioutil.TempDir("", "export")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dataDir)

		//create user
		userObject := user.NewUsers(dataDir, mockClient, "", logger)
		userAddressString, _, ui, err := userObject.CreateNewUser("user1", "password1", "", nil, "")
		if err != nil {
			t.Fatal(err)
		}

		// export user
		userName, address, err := userObject.ExportUser(ui)
		if err != nil {
			t.Fatal(err)
		}

		//validate
		if userName != "user1" {
			t.Fatalf("invalid user name")
		}
		if address != userAddressString {
			t.Fatalf("invalid user address")
		}

	})
}
