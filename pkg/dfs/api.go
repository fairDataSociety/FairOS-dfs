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
	"errors"
	"fmt"
	"io"
	"time"

	blockstore "github.com/asabya/swarm-blockstore"
	"github.com/asabya/swarm-blockstore/bee"
	"github.com/fairdatasociety/fairOS-dfs/pkg/contracts"
	ethClient "github.com/fairdatasociety/fairOS-dfs/pkg/ensm/eth"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/subscriptionManager"
	"github.com/fairdatasociety/fairOS-dfs/pkg/subscriptionManager/rpc"
	"github.com/fairdatasociety/fairOS-dfs/pkg/user"
	"github.com/plexsysio/taskmanager"
)

const (
	defaultMaxWorkers = 100
)

// API is the go api for fairOS
type API struct {
	context       context.Context
	cancel        context.CancelFunc
	client        blockstore.Client
	users         *user.Users
	logger        logging.Logger
	tm            *taskmanager.TaskManager
	sm            subscriptionManager.SubscriptionManager
	feedCacheSize int
	feedCacheTTL  time.Duration
	io.Closer
}

type Options struct {
	BeeApiEndpoint     string
	Stamp              string
	EnsConfig          *contracts.ENSConfig
	SubscriptionConfig *contracts.SubscriptionConfig
	Logger             logging.Logger
	FeedCacheSize      int
	FeedCacheTTL       time.Duration
	RedundancyLevel    uint8
}

// NewDfsAPI is the main entry point for the df controller.
func NewDfsAPI(ctx context.Context, opts *Options) (*API, error) {
	logger := opts.Logger
	ens, err := ethClient.New(opts.EnsConfig, logger)
	if err != nil {
		logger.Errorf("dfs: ens initialisation failed %s", err.Error())
		if errors.Is(err, ethClient.ErrWrongChainID) {
			return nil, err
		}
		return nil, errEthClient
	}
	c := bee.NewBeeClient(opts.BeeApiEndpoint, bee.WithStamp(opts.Stamp), bee.WithRedundancy(fmt.Sprintf("%d", opts.RedundancyLevel)))
	if !c.CheckConnection() {
		logger.Errorf("dfs: bee client initialisation failed")
		return nil, errBeeClient
	}
	// Setting cache size 0 will disable the cache. This is to change the default behaviour of lru itself.
	// We have this -1 check hard coded in the feed package. -1 will disable the feed pool off. and write directly to swarm.
	if opts.FeedCacheSize == 0 {
		opts.FeedCacheSize = -1
	}
	users := user.NewUsers(c, ens, opts.FeedCacheSize, opts.FeedCacheTTL, logger)

	var sm subscriptionManager.SubscriptionManager
	if opts.SubscriptionConfig != nil {
		logger.Infof("dfs: subscriptionManager initialisation")
		sm, err = rpc.New(opts.SubscriptionConfig, logger, c, c)
		if err != nil {
			logger.Errorf("dfs: subscriptionManager initialisation failed %s", err.Error())
			return nil, errSubManager
		}
	}

	// discard tm logs as it creates too much noise
	tmLogger := logging.New(io.Discard, 0)
	ctx2, cancel := context.WithCancel(ctx)
	return &API{
		context:       ctx2,
		cancel:        cancel,
		client:        c,
		users:         users,
		logger:        logger,
		tm:            taskmanager.New(10, defaultMaxWorkers, time.Second*15, tmLogger),
		sm:            sm,
		feedCacheSize: opts.FeedCacheSize,
		feedCacheTTL:  opts.FeedCacheTTL,
	}, nil
}

// NewMockDfsAPI is used for tests only
func NewMockDfsAPI(client blockstore.Client, users *user.Users, logger logging.Logger) *API {
	ctx, cancel := context.WithCancel(context.Background())
	return &API{
		context: ctx,
		cancel:  cancel,
		client:  client,
		users:   users,
		logger:  logger,
		tm:      taskmanager.New(1, 100, time.Second*15, logger),
	}
}

// Close stops the taskmanager
func (a *API) Close() error {
	users := a.users.GetUsersLoggedIn()
	for sessionId, v := range users {
		err := a.LogoutUser(sessionId)
		if err != nil {
			a.logger.Errorf("dfs: error logging out user %s", v.GetUserName())
		}
	}

	ctx, cancel := context.WithTimeout(a.context, time.Minute)
	defer func() {
		cancel()
		a.cancel()
	}()
	return a.tm.Stop(ctx)
}
