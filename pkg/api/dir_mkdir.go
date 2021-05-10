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

func (h *Handler) DirectoryMkdirHandler(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != jsonContentType {
		h.logger.Errorf("mkdir: invalid request body type")
		jsonhttp.BadRequest(w, "mkdir: invalid request body type")
		return
	}

	decoder := json.NewDecoder(r.Body)
	var fsReq common.FileSystemRequest
	err := decoder.Decode(&fsReq)
	if err != nil {
		h.logger.Errorf("mkdir: could not decode arguments")
		jsonhttp.BadRequest(w, "mkdir: could not decode arguments")
		return
	}

	dirToCreate := fsReq.DirectoryName
	if dirToCreate == "" {
		h.logger.Errorf("mkdir: \"dir\" argument missing")
		jsonhttp.BadRequest(w, "mkdir: \"dir\" argument missing")
		return
	}

	path := fsReq.DirectoryPath
	if path == "" {
		h.logger.Errorf("mkdir: \"path\" argument missing")
		jsonhttp.BadRequest(w, "mkdir: \"path\" argument missing")
		return
	}

	// get values from cookie
	sessionId, err := cookie.GetSessionIdFromCookie(r)
	if err != nil {
		h.logger.Errorf("mkdir: invalid cookie: %v", err)
		jsonhttp.BadRequest(w, ErrInvalidCookie)
		return
	}
	if sessionId == "" {
		h.logger.Errorf("mkdir: \"cookie-id\" parameter missing in cookie")
		jsonhttp.BadRequest(w, "mkdir: \"cookie-id\" parameter missing in cookie")
		return
	}

	// make directory
	err = h.dfsAPI.Mkdir(path, dirToCreate, sessionId)
	if err != nil {
		if err == dfs.ErrPodNotOpen || err == dfs.ErrUserNotLoggedIn ||
			err == p.ErrInvalidDirectory ||
			err == p.ErrTooLongDirectoryName ||
			err == p.ErrPodNotOpened {
			h.logger.Errorf("mkdir: %v", err)
			jsonhttp.BadRequest(w, "mkdir: "+err.Error())
			return
		}
		h.logger.Errorf("mkdir: %v", err)
		jsonhttp.InternalServerError(w, "mkdir: "+err.Error())
		return
	}
	jsonhttp.Created(w, "directory created successfully")
}
