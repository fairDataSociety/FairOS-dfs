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
	"encoding/json"
	"net/http"

	"github.com/fairdatasociety/fairOS-dfs/cmd/common"
	"resenje.org/jsonhttp"

	"github.com/fairdatasociety/fairOS-dfs/pkg/cookie"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dfs"
	p "github.com/fairdatasociety/fairOS-dfs/pkg/pod"
)

// PodCloseHandler is the api handler to close an open pod
// it takes no arguments
func (h *Handler) PodCloseHandler(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != jsonContentType {
		h.logger.Errorf("pod close: invalid request body type")
		jsonhttp.BadRequest(w, "pod close: invalid request body type")
		return
	}

	decoder := json.NewDecoder(r.Body)
	var podReq common.PodRequest
	err := decoder.Decode(&podReq)
	if err != nil {
		h.logger.Errorf("pod close: could not decode arguments")
		jsonhttp.BadRequest(w, "pod close: could not decode arguments")
		return
	}
	podName := podReq.PodName

	// get values from cookie
	sessionId, err := cookie.GetSessionIdFromCookie(r)
	if err != nil {
		h.logger.Errorf("pod close: invalid cookie: %v", err)
		jsonhttp.BadRequest(w, ErrInvalidCookie)
		return
	}
	if sessionId == "" {
		h.logger.Errorf("pod close: \"cookie-id\" parameter missing in cookie")
		jsonhttp.BadRequest(w, "pod close: \"cookie-id\" parameter missing in cookie")
		return
	}

	// close pod
	err = h.dfsAPI.ClosePod(podName, sessionId)
	if err != nil {
		if err == dfs.ErrPodNotOpen || err == dfs.ErrUserNotLoggedIn ||
			err == p.ErrPodNotOpened {
			h.logger.Errorf("pod close: %v", err)
			jsonhttp.BadRequest(w, "pod close: "+err.Error())
			return
		}
		h.logger.Errorf("pod close: %v", err)
		jsonhttp.InternalServerError(w, "pod close: "+err.Error())
		return
	}
	jsonhttp.OK(w, "pod closed successfully")
}
