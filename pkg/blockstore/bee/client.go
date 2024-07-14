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

package bee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ethersphere/bee/v2/pkg/swarm"
	bmtlegacy "github.com/ethersphere/bmt/legacy"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

const (
	maxIdleConnections        = 20
	maxConnectionsPerHost     = 256
	requestTimeout            = 6000
	chunkCacheSize            = 1024
	uploadBlockCacheSize      = 100
	downloadBlockCacheSize    = 100
	healthUrl                 = "/health"
	chunkUploadDownloadUrl    = "/chunks"
	bytesUploadDownloadUrl    = "/bytes"
	bzzUrl                    = "/bzz"
	tagsUrl                   = "/tags"
	pinsUrl                   = "/pins/"
	feedsUrl                  = "/feeds/"
	swarmPinHeader            = "Swarm-Pin"
	swarmEncryptHeader        = "Swarm-Encrypt"
	swarmPostageBatchId       = "Swarm-Postage-Batch-Id"
	swarmDeferredUploadHeader = "Swarm-Deferred-Upload"
	SwarmErasureCodingHeader  = "Swarm-Redundancy-Level"
	swarmTagHeader            = "Swarm-Tag"
	contentTypeHeader         = "Content-Type"
)

// Client is a bee http client that satisfies blockstore.Client
type Client struct {
	url                string
	client             *http.Client
	hasher             *bmtlegacy.Hasher
	chunkCache         *lru.LRU[string, []byte]
	uploadBlockCache   *lru.LRU[string, []byte]
	downloadBlockCache *lru.LRU[string, []byte]
	postageBlockId     string
	logger             logging.Logger
	isProxy            bool
	shouldPin          bool
	redundancyLevel    uint8
}

func hashFunc() hash.Hash {
	return sha3.NewLegacyKeccak256()
}

type bytesPostResponse struct {
	Reference swarm.Address `json:"reference"`
}

type tagPostRequest struct {
	Address string `json:"address"`
}

type tagPostResponse struct {
	UID       uint32    `json:"uid"`
	StartedAt time.Time `json:"startedAt"`
	Total     int64     `json:"total"`
	Processed int64     `json:"processed"`
	Synced    int64     `json:"synced"`
}

type beeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewBeeClient creates a new client which connects to the Swarm bee node to access the Swarm network.
func NewBeeClient(apiUrl, postageBlockId string, shouldPin bool, redundancyLevel uint8, logger logging.Logger) *Client {
	p := bmtlegacy.NewTreePool(hashFunc, swarm.Branches, bmtlegacy.PoolSize)
	cache := lru.NewLRU(chunkCacheSize, func(key string, value []byte) {}, 0)
	uploadBlockCache := lru.NewLRU(uploadBlockCacheSize, func(key string, value []byte) {}, 0)
	downloadBlockCache := lru.NewLRU(downloadBlockCacheSize, func(key string, value []byte) {}, 0)
	return &Client{
		url:                apiUrl,
		client:             createHTTPClient(),
		hasher:             bmtlegacy.New(p),
		chunkCache:         cache,
		uploadBlockCache:   uploadBlockCache,
		downloadBlockCache: downloadBlockCache,
		postageBlockId:     postageBlockId,
		logger:             logger,
		shouldPin:          shouldPin,
		redundancyLevel:    redundancyLevel,
	}
}

type chunkAddressResponse struct {
	Reference swarm.Address `json:"reference"`
}

func socResource(owner, id, sig string) string {
	return fmt.Sprintf("/soc/%s/%s?sig=%s", owner, id, sig)
}

// CheckConnection is used to check if the bee client is up and running.
func (s *Client) CheckConnection() bool {
	// check if node is standalone bee
	matchString := "Ethereum Swarm Bee\n"
	data, _ := s.checkBee(false)
	if data == matchString {
		return true
	}

	// check if node is gateway-proxy
	data, err := s.checkBee(true)
	if err != nil {
		return false
	}
	matchString = "OK"
	s.isProxy = data == matchString

	return s.isProxy
}

func (s *Client) checkBee(isProxy bool) (string, error) {
	url := s.url
	if isProxy {
		url += healthUrl
	}
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Close = true
	// skipcq: GO-S2307
	response, err := s.Do(req)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Do dispatches the HTTP request to the network
func (s *Client) Do(req *http.Request) (*http.Response, error) {
	return s.client.Do(req)
}

// UploadSOC is used construct and send a Single Owner Chunk to the Swarm bee client.
func (s *Client) UploadSOC(owner, id, signature string, data []byte) (address []byte, err error) {
	to := time.Now()
	socResStr := socResource(owner, id, signature)
	fullUrl := fmt.Sprintf(s.url + socResStr)

	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Close = true

	// the postage block id to store the SOC chunk
	req.Header.Set(swarmPostageBatchId, s.postageBlockId)
	req.Header.Set(contentTypeHeader, "application/octet-stream")
	req.Header.Set(swarmDeferredUploadHeader, "false")

	// TODO change this in the future when we have some alternative to pin SOC
	// This is a temporary fix to force soc pinning
	if s.shouldPin {
		req.Header.Set(swarmPinHeader, "true")
	}

	response, err := s.Do(req)
	if err != nil {
		return nil, err
	}

	// skipcq: GO-S2307
	defer response.Body.Close()

	addrData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error uploading data")
	}

	if response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(addrData, &beeErr)
		if err != nil {
			return nil, errors.New(string(addrData))
		}
		return nil, errors.New(beeErr.Message)
	}

	var addrResp *chunkAddressResponse
	err = json.Unmarshal(addrData, &addrResp)
	if err != nil {
		return nil, err
	}

	fields := logrus.Fields{
		"reference": addrResp.Reference.String(),
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "upload soc: ")
	return addrResp.Reference.Bytes(), nil
}

// UploadChunk uploads a chunk to Swarm network.
func (s *Client) UploadChunk(ch swarm.Chunk) (address []byte, err error) {
	to := time.Now()
	fullUrl := fmt.Sprintf(s.url + chunkUploadDownloadUrl)
	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(ch.Data()))
	if err != nil {
		return nil, err
	}
	req.Close = true

	if s.shouldPin {
		req.Header.Set(swarmPinHeader, "true")
	}

	req.Header.Set(contentTypeHeader, "application/octet-stream")

	// the postage block id to store the chunk
	req.Header.Set(swarmPostageBatchId, s.postageBlockId)

	req.Header.Set(swarmDeferredUploadHeader, "true")

	response, err := s.Do(req)
	if err != nil {
		return nil, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	addrData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error uploading data")
	}

	if response.StatusCode != http.StatusOK {
		var beeErr *beeError
		err = json.Unmarshal(addrData, &beeErr)
		if err != nil {
			return nil, errors.New(string(addrData))
		}
		return nil, errors.New(beeErr.Message)
	}

	var addrResp *chunkAddressResponse
	err = json.Unmarshal(addrData, &addrResp)
	if err != nil {
		return nil, err
	}

	fields := logrus.Fields{
		"reference": ch.Address().String(),
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "upload chunk: ")

	return addrResp.Reference.Bytes(), nil
}

// DownloadChunk downloads a chunk with given address from the Swarm network
func (s *Client) DownloadChunk(ctx context.Context, address []byte) (data []byte, err error) {
	to := time.Now()
	addrString := swarm.NewAddress(address).String()

	path := chunkUploadDownloadUrl + "/" + addrString
	fullUrl := fmt.Sprintf(s.url + path)
	req, err := http.NewRequest(http.MethodGet, fullUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Close = true

	req = req.WithContext(ctx)

	response, err := s.Do(req)
	if err != nil {
		return nil, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("error downloading data")
	}

	data, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error downloading data")
	}
	fields := logrus.Fields{
		"reference": addrString,
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "download chunk: ")
	return data, nil
}

// UploadBlob uploads a binary blob of data to Swarm network. It also optionally pins and encrypts the data.
func (s *Client) UploadBlob(data []byte, tag uint32, encrypt bool) (address []byte, err error) {
	to := time.Now()

	// return the ref if this data is already in swarm
	if s.inBlockCache(s.uploadBlockCache, string(data)) {
		return s.getFromBlockCache(s.uploadBlockCache, string(data)), nil
	}

	fullUrl := s.url + bytesUploadDownloadUrl
	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Close = true

	req.Header.Set(contentTypeHeader, "application/octet-stream")
	req.Header.Set(SwarmErasureCodingHeader, strconv.Itoa(int(s.redundancyLevel)))

	if s.shouldPin {
		req.Header.Set(swarmPinHeader, "true")
	}

	if encrypt {
		req.Header.Set(swarmEncryptHeader, "true")
	}

	if tag > 0 {
		req.Header.Set(swarmTagHeader, fmt.Sprintf("%d", tag))
	}

	// the postage block id to store the blob
	req.Header.Set(swarmPostageBatchId, s.postageBlockId)

	req.Header.Set(swarmDeferredUploadHeader, "true")

	response, err := s.Do(req)
	if err != nil {
		return nil, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error uploading blob")
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return nil, errors.New(string(respData))
		}
		return nil, errors.New(beeErr.Message)
	}

	var resp bytesPostResponse
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response")
	}
	fields := logrus.Fields{
		"reference": resp.Reference.String(),
		"size":      len(data),
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "upload blob: ")

	// add the data in cache
	s.addToBlockCache(s.uploadBlockCache, string(data), resp.Reference.Bytes())

	return resp.Reference.Bytes(), nil
}

// DownloadBlob downloads a blob of binary data from the Swarm network.
func (s *Client) DownloadBlob(address []byte) ([]byte, int, error) {
	to := time.Now()

	// return the data if this address is already in cache
	addrString := swarm.NewAddress(address).String()
	if s.inBlockCache(s.downloadBlockCache, addrString) {
		return s.getFromBlockCache(s.downloadBlockCache, addrString), 200, nil
	}

	fullUrl := s.url + bytesUploadDownloadUrl + "/" + addrString
	req, err := http.NewRequest(http.MethodGet, fullUrl, http.NoBody)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	req.Close = true

	response, err := s.Do(req)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, errors.New("error downloading blob")
	}

	if response.StatusCode != http.StatusOK {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return nil, response.StatusCode, errors.New(string(respData))
		}
		return nil, response.StatusCode, errors.New(beeErr.Message)
	}

	fields := logrus.Fields{
		"reference": addrString,
		"size":      len(respData),
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "download blob: ")

	// add the data and ref if it is not in cache
	if !s.inBlockCache(s.downloadBlockCache, addrString) {
		s.addToBlockCache(s.downloadBlockCache, addrString, respData)
	}
	return respData, response.StatusCode, nil
}

// UploadBzz uploads a file through bzz api
func (s *Client) UploadBzz(data []byte, fileName string) (address []byte, err error) {
	to := time.Now()

	fullUrl := s.url + bzzUrl + "?name=" + fileName
	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Close = true

	req.Header.Set(swarmPostageBatchId, s.postageBlockId)
	req.Header.Set(contentTypeHeader, "application/json")
	req.Header.Set(SwarmErasureCodingHeader, strconv.Itoa(int(s.redundancyLevel)))

	response, err := s.Do(req)
	if err != nil {
		return nil, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error downloading bzz")
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return nil, errors.New(string(respData))
		}
		return nil, errors.New(beeErr.Message)
	}

	var resp bytesPostResponse
	err = json.Unmarshal(respData, &resp)

	fields := logrus.Fields{
		"reference": resp.Reference.String(),
		"size":      len(respData),
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "upload bzz: ")

	// add the data and ref if it is not in cache
	if !s.inBlockCache(s.downloadBlockCache, resp.Reference.String()) {
		s.addToBlockCache(s.downloadBlockCache, resp.Reference.String(), data)
	}
	return resp.Reference.Bytes(), nil
}

// DownloadBzz downloads bzz data from the Swarm network.
func (s *Client) DownloadBzz(address []byte) ([]byte, int, error) {
	to := time.Now()

	// return the data if this address is already in cache
	addrString := swarm.NewAddress(address).String()
	if s.inBlockCache(s.downloadBlockCache, addrString) {
		return s.getFromBlockCache(s.downloadBlockCache, addrString), 200, nil
	}

	fullUrl := s.url + bzzUrl + "/" + addrString
	req, err := http.NewRequest(http.MethodGet, fullUrl, http.NoBody)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	req.Close = true

	response, err := s.Do(req)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, errors.New("error downloading bzz")
	}

	if response.StatusCode != http.StatusOK {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return nil, response.StatusCode, errors.New(string(respData))
		}
		return nil, response.StatusCode, errors.New(beeErr.Message)
	}

	fields := logrus.Fields{
		"reference": addrString,
		"size":      len(respData),
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "download bzz: ")

	// add the data and ref if it is not in cache
	if !s.inBlockCache(s.downloadBlockCache, addrString) {
		s.addToBlockCache(s.downloadBlockCache, addrString, respData)
	}
	return respData, response.StatusCode, nil
}

// DeleteReference unpins a reference so that it will be garbage collected by the Swarm network.
func (s *Client) DeleteReference(address []byte) error {
	if !s.shouldPin {
		return nil
	}
	to := time.Now()
	addrString := swarm.NewAddress(address).String()

	fullUrl := s.url + pinsUrl + addrString
	req, err := http.NewRequest(http.MethodDelete, fullUrl, http.NoBody)
	if err != nil {
		return err
	}
	req.Close = true

	response, err := s.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		respData, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to unpin reference : %s", respData)
	} else {
		_, _ = io.Copy(io.Discard, response.Body)
	}

	fields := logrus.Fields{
		"reference": addrString,
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "delete chunk: ")

	return nil
}

// CreateTag creates a tag for given address
func (s *Client) CreateTag(address []byte) (uint32, error) {
	// gateway proxy does not have tags api exposed
	if s.isProxy {
		return 0, nil
	}
	to := time.Now()

	fullUrl := s.url + tagsUrl
	var data []byte
	var err error
	if len(address) > 0 {
		addrString := swarm.NewAddress(address).String()
		b := &tagPostRequest{Address: addrString}
		data, err = json.Marshal(b)
		if err != nil {
			return 0, err
		}
	}
	req, err := http.NewRequest(http.MethodPost, fullUrl, bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	req.Close = true

	response, err := s.Do(req)
	if err != nil {
		return 0, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, errors.New("error create tag")
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return 0, errors.New(string(respData))
		}
		return 0, errors.New(beeErr.Message)
	}

	var resp tagPostResponse
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling response")
	}
	fields := logrus.Fields{
		"reference": address,
		"tag":       resp.UID,
		"duration":  time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "create tag: ")

	return resp.UID, nil
}

func (s *Client) CreateFeedManifest(owner, topic string) (swarm.Address, error) {
	to := time.Now()

	fullUrl := s.url + feedsUrl + owner + "/" + topic
	fmt.Println("fullUrl: ", fullUrl)
	req, err := http.NewRequest(http.MethodPost, fullUrl, nil)
	if err != nil {
		return swarm.ZeroAddress, err
	}
	req.Close = true

	req.Header.Set(swarmPostageBatchId, s.postageBlockId)

	response, err := s.Do(req)
	if err != nil {
		return swarm.ZeroAddress, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return swarm.ZeroAddress, errors.New("error create feed manifest")
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return swarm.ZeroAddress, errors.New(string(respData))
		}
		return swarm.ZeroAddress, errors.New(beeErr.Message)
	}

	var resp bytesPostResponse
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return swarm.ZeroAddress, fmt.Errorf("error unmarshalling response")
	}
	fields := logrus.Fields{
		"owner":    owner,
		"topic":    topic,
		"duration": time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "create feed manifest: ")
	return resp.Reference, nil
}

func (s *Client) GetLatestFeedManifest(owner, topic string) ([]byte, string, string, error) {
	to := time.Now()

	fullUrl := s.url + feedsUrl + owner + "/" + topic
	fmt.Println("get ullUrl: ", fullUrl)

	req, err := http.NewRequest(http.MethodGet, fullUrl, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Close = true

	response, err := s.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", "", errors.New("error getting latest feed manifest")
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return nil, "", "", errors.New(string(respData))
		}
		return nil, "", "", errors.New(beeErr.Message)
	}

	var resp bytesPostResponse
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return nil, "", "", fmt.Errorf("error unmarshalling response")
	}
	fields := logrus.Fields{
		"owner":    owner,
		"topic":    topic,
		"duration": time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "get latest feed manifest: ")

	return resp.Reference.Bytes(), response.Header.Get("swarm-feed-index"), response.Header.Get("swarm-feed-index-next"), nil
}

// GetTag gets sync status of a given tag
func (s *Client) GetTag(tag uint32) (int64, int64, int64, error) {
	// gateway proxy does not have tags api exposed
	if s.isProxy {
		return 0, 0, 0, nil
	}

	to := time.Now()

	fullUrl := s.url + tagsUrl + fmt.Sprintf("/%d", tag)

	req, err := http.NewRequest(http.MethodGet, fullUrl, http.NoBody)
	if err != nil {
		return 0, 0, 0, err
	}
	req.Close = true

	response, err := s.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	// skipcq: GO-S2307
	defer response.Body.Close()

	respData, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, 0, 0, errors.New("error getting tag")
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var beeErr *beeError
		err = json.Unmarshal(respData, &beeErr)
		if err != nil {
			return 0, 0, 0, errors.New(string(respData))
		}
		return 0, 0, 0, errors.New(beeErr.Message)
	}

	var resp tagPostResponse
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error unmarshalling response")
	}
	fields := logrus.Fields{
		"tag":      resp.UID,
		"duration": time.Since(to).String(),
	}
	s.logger.WithFields(fields).Log(logrus.DebugLevel, "get tag: ")

	return resp.Total, resp.Processed, resp.Synced, nil
}

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: time.Second * requestTimeout,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: maxIdleConnections,
			MaxConnsPerHost:     maxConnectionsPerHost,
		},
	}
	return client
}

func (*Client) addToBlockCache(cache *lru.LRU[string, []byte], key string, value []byte) {
	if cache != nil {
		cache.Add(key, value)
	}
}

func (*Client) inBlockCache(cache *lru.LRU[string, []byte], key string) bool {
	_, in := cache.Get(key)
	return in
}

func (*Client) getFromBlockCache(cache *lru.LRU[string, []byte], key string) []byte {
	if cache != nil {
		value, ok := cache.Get(key)
		if ok {
			return value
		}
		return nil
	}
	return nil
}
