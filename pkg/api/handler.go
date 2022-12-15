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

package api

import (
	"github.com/fairdatasociety/fairOS-dfs/pkg/contracts"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dfs"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
)

type Handler struct {
	dfsAPI *dfs.API
	logger logging.Logger

	whitelistedOrigins []string
	cookieDomain       string
}

// New handler is the main handler for fairOS-dfs
func New(beeApi, cookieDomain, postageBlockId string, whitelistedOrigins []string, isGatewayProxy bool, ensConfig *contracts.Config, logger logging.Logger) (*Handler, error) {
	api, err := dfs.NewDfsAPI(beeApi, postageBlockId, isGatewayProxy, ensConfig, logger)
	if err != nil {
		return nil, err
	}
	return &Handler{
		dfsAPI:             api,
		logger:             logger,
		whitelistedOrigins: whitelistedOrigins,
		cookieDomain:       cookieDomain,
	}, nil
}

// NewMockHandler is used for tests only
func NewMockHandler(dfsAPI *dfs.API, logger logging.Logger, whitelistedOrigins []string) *Handler {
	return &Handler{
		dfsAPI:             dfsAPI,
		logger:             logger,
		whitelistedOrigins: whitelistedOrigins,
	}
}
