package test_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	mockpost "github.com/ethersphere/bee/pkg/postage/mock"
	mockstorer "github.com/ethersphere/bee/pkg/storer/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	mock2 "github.com/fairdatasociety/fairOS-dfs/pkg/ensm/eth/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	mock3 "github.com/fairdatasociety/fairOS-dfs/pkg/subscriptionManager/rpc/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/user"
	"github.com/plexsysio/taskmanager"
	"github.com/sirupsen/logrus"
)

func TestLite(t *testing.T) {
	storer := mockstorer.New()
	beeUrl := mock.NewTestBeeServer(t, mock.TestServerOptions{
		Storer:          storer,
		PreventRedirect: true,
		Post:            mockpost.New(mockpost.WithAcceptAll()),
	})

	logger := logging.New(io.Discard, logrus.DebugLevel)
	mockClient := bee.NewBeeClient(beeUrl, mock.BatchOkStr, true, logger)
	tm := taskmanager.New(1, 10, time.Second*15, logger)
	defer func() {
		_ = tm.Stop(context.Background())
	}()
	sm := mock3.NewMockSubscriptionManager()

	t.Run("new-blank-username", func(t *testing.T) {
		ens := mock2.NewMockNamespaceManager()

		// create user
		userObject := user.NewUsers(mockClient, ens, logger)
		_, _, _, err := userObject.LoadLiteUser("", "password1", "", "", tm, sm)
		if !errors.Is(err, user.ErrInvalidUserName) {
			t.Fatal(err)
		}
	})

	t.Run("new-user", func(t *testing.T) {
		ens := mock2.NewMockNamespaceManager()

		// create user
		userObject := user.NewUsers(mockClient, ens, logger)
		mnemonic, _, ui, err := userObject.LoadLiteUser("user1", "password1", "", "", tm, sm)
		if err != nil {
			t.Fatal(err)
		}

		// validate user
		if !userObject.IsUserNameLoggedIn(ui.GetUserName()) {
			t.Fatalf("user not loggin in")
		}
		if ui == nil {
			t.Fatalf("invalid user info")
		}
		if ui.GetUserName() != "user1" {
			t.Fatalf("invalid user name")
		}
		if ui.GetFeed() == nil || ui.GetAccount() == nil || ui.GetPod() == nil {
			t.Fatalf("invalid feed, account or pod")
		}
		err = ui.GetAccount().GetWallet().IsValidMnemonic(mnemonic)
		if err != nil {
			t.Fatalf("invalid mnemonic")
		}
	})
}
