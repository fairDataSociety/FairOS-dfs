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

	"github.com/fairdatasociety/fairOS-dfs/pkg/dfs"

	"resenje.org/jsonhttp"

	"github.com/fairdatasociety/fairOS-dfs/pkg/cookie"
)

func (h *Handler) FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != jsonContentType {
		h.logger.Errorf("file delete: invalid request body type")
		jsonhttp.BadRequest(w, "file delete: invalid request body type")
		return
	}

	decoder := json.NewDecoder(r.Body)
	var fsReq common.FileSystemRequest
	err := decoder.Decode(&fsReq)
	if err != nil {
		h.logger.Errorf("file delete: could not decode arguments")
		jsonhttp.BadRequest(w, "file delete: could not decode arguments")
		return
	}

	podFileWithPath := fsReq.FilePath
	if podFileWithPath == "" {
		h.logger.Errorf("file delete: \"file_path\" argument missing")
		jsonhttp.BadRequest(w, "file delete: \"file_path\" argument missing")
		return
	}

	// get values from cookie
	sessionId, err := cookie.GetSessionIdFromCookie(r)
	if err != nil {
		h.logger.Errorf("file delete: invalid cookie: %v", err)
		jsonhttp.BadRequest(w, ErrInvalidCookie)
		return
	}
	if sessionId == "" {
		h.logger.Errorf("file delete: \"cookie-id\" parameter missing in cookie")
		jsonhttp.BadRequest(w, "file delete: \"cookie-id\" parameter missing in cookie")
		return
	}

	// delete file
	err = h.dfsAPI.DeleteFile(podFileWithPath, sessionId)
	if err != nil {
		if err == dfs.ErrPodNotOpen {
			h.logger.Errorf("file delete: %v", err)
			jsonhttp.BadRequest(w, "file delete: "+err.Error())
			return
		}
		h.logger.Errorf("file delete: %v", err)
		jsonhttp.InternalServerError(w, "file delete: "+err.Error())
		return
	}

	jsonhttp.OK(w, "file deleted successfully")
}
