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
	"errors"
	"net/http"
	"strconv"

	"github.com/fairdatasociety/fairOS-dfs/cmd/common"

	"github.com/fairdatasociety/fairOS-dfs/pkg/collection"
	"github.com/fairdatasociety/fairOS-dfs/pkg/cookie"
	"resenje.org/jsonhttp"
)

const (
	DefaultSeekLimit = "10"
)

// KVSeekHandler is the api handler to seek to a particular key with the given prefix
// it takes four arguments, 2 mandatory and two optional
// - table_name: the name of the kv table
// - start_prefix: the prefix of the key to seek
// * end_prefix: the prefix of the end key
// * limit: the threshold for the number of keys to go when get_next is called
func (h *Handler) KVSeekHandler(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != jsonContentType {
		h.logger.Errorf("kv seek: invalid request body type")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: invalid request body type"})
		return
	}

	decoder := json.NewDecoder(r.Body)
	var kvReq common.KVRequest
	err := decoder.Decode(&kvReq)
	if err != nil {
		h.logger.Errorf("kv seek: could not decode arguments")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: could not decode arguments"})
		return
	}

	podName := kvReq.PodName
	if podName == "" {
		h.logger.Errorf("kv seek: \"pod_name\" argument missing")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: \"pod_name\" argument missing"})
		return
	}

	name := kvReq.TableName
	if name == "" {
		h.logger.Errorf("kv seek: \"table_name\" argument missing")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: \"table_name\" argument missing"})
		return
	}

	start := kvReq.StartPrefix
	if start == "" {
		h.logger.Errorf("kv seek: \"start\" argument missing")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: \"start\" argument missing"})
		return
	}

	end := kvReq.EndPrefix
	limit := kvReq.Limit
	if limit == "" {
		limit = DefaultSeekLimit
	}
	noOfRows, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		h.logger.Errorf("kv seek: invalid limit")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: invalid limit"})
		return
	}

	// get values from cookie
	sessionId, err := cookie.GetSessionIdFromCookie(r)
	if err != nil {
		h.logger.Errorf("kv seek: invalid cookie: %v", err)
		jsonhttp.BadRequest(w, &response{Message: ErrInvalidCookie.Error()})
		return
	}
	if sessionId == "" {
		h.logger.Errorf("kv seek: \"cookie-id\" parameter missing in cookie")
		jsonhttp.BadRequest(w, &response{Message: "kv seek: \"cookie-id\" parameter missing in cookie"})
		return
	}

	_, err = h.dfsAPI.KVSeek(sessionId, podName, name, start, end, noOfRows)
	if err != nil {
		h.logger.Errorf("kv seek: %v", err)
		jsonhttp.InternalServerError(w, &response{Message: "kv seek: " + err.Error()})
		return
	}
	jsonhttp.OK(w, &response{Message: "seeked closest to the start key"})
}

// KVGetNextHandler is the api handler to get the key and value from the current seek position
// it takes only oneargument
// - table_name: the name of the kv table
func (h *Handler) KVGetNextHandler(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["pod_name"]
	if !ok || len(keys[0]) < 1 {
		h.logger.Errorf("kv get_next: \"pod_name\" argument missing")
		jsonhttp.BadRequest(w, "kv get_next: \"pod_name\" argument missing")
		return
	}

	podName := keys[0]
	if podName == "" {
		h.logger.Errorf("kv get_next: \"pod_name\" argument missing")
		jsonhttp.BadRequest(w, "kv get_next: \"pod_name\" argument missing")
		return
	}

	keys, ok = r.URL.Query()["table_name"]
	if !ok || len(keys[0]) < 1 {
		h.logger.Errorf("kv get_next: \"table_name\" argument missing")
		jsonhttp.BadRequest(w, "kv get_next: \"table_name\" argument missing")
		return
	}

	name := keys[0]
	if name == "" {
		h.logger.Errorf("kv get_next: \"table_name\" argument missing")
		jsonhttp.BadRequest(w, "kv get_next: \"table_name\" argument missing")
		return
	}

	// get values from cookie
	sessionId, err := cookie.GetSessionIdFromCookie(r)
	if err != nil {
		h.logger.Errorf("kv get_next: invalid cookie: %v", err)
		jsonhttp.BadRequest(w, ErrInvalidCookie)
		return
	}
	if sessionId == "" {
		h.logger.Errorf("kv get_next: \"cookie-id\" parameter missing in cookie")
		jsonhttp.BadRequest(w, "kv get_next: \"cookie-id\" parameter missing in cookie")
		return
	}

	columns, key, data, err := h.dfsAPI.KVGetNext(sessionId, podName, name)
	if err != nil && !errors.Is(err, collection.ErrNoNextElement) {
		h.logger.Errorf("kv get_next: %v", err)
		jsonhttp.InternalServerError(w, "kv get_next: "+err.Error())
		return
	}

	if errors.Is(err, collection.ErrNoNextElement) {
		jsonhttp.Respond(w, http.StatusNoContent, nil)
		return
	}

	var resp KVResponse
	if columns != nil {
		resp.Keys = columns
	} else {
		resp.Keys = []string{key}
	}
	resp.Values = data

	w.Header().Set("Content-Type", "application/json")
	jsonhttp.OK(w, &resp)
}
