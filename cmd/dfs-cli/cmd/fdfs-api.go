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

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/pkg/api"
	"resenje.org/jsonhttp"
)

const (
	maxIdleConnections    = 20
	maxConnectionsPerHost = 256
	requestTimeout        = 6000
)

// fdfsClient is the http client for dfs
type fdfsClient struct {
	url    string
	client *http.Client
	cookie *http.Cookie
}

func newFdfsClient(fdfsServer string) (*fdfsClient, error) {
	client, err := createHTTPClient()
	if err != nil {
		return nil, err
	}
	return &fdfsClient{
		url:    fdfsServer,
		client: client,
	}, nil
}

func createHTTPClient() (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil { // error handling
		return nil, err
	}
	client := &http.Client{
		Timeout: time.Second * requestTimeout,
		Jar:     jar,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: maxIdleConnections,
			MaxConnsPerHost:     maxConnectionsPerHost,
		},
	}
	return client, nil
}

// CheckConnection checks if it can connect to dfs server
func (s *fdfsClient) CheckConnection() bool {
	req, err := http.NewRequest(http.MethodGet, s.url, http.NoBody)
	if err != nil {
		return false
	}

	response, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	req.Close = true

	if response.StatusCode != http.StatusOK {
		return false
	}

	_, err = io.ReadAll(response.Body)
	return err == nil
}

func (s *fdfsClient) postReq(method, urlPath string, jsonBytes []byte) ([]byte, error) {
	// prepare the  request
	fullUrl := fmt.Sprintf(s.url + urlPath)
	var req *http.Request
	var err error
	if jsonBytes != nil {
		req, err = http.NewRequest(method, fullUrl, bytes.NewBuffer(jsonBytes))
		if err != nil {
			return nil, err
		}

		// add the headers
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Content-Length", strconv.Itoa(len(jsonBytes)))
	} else {
		req, err = http.NewRequest(method, fullUrl, http.NoBody)
		if err != nil {
			return nil, err
		}
	}

	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}
	// execute the request
	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	req.Close = true

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		if response.StatusCode == http.StatusNoContent {
			return nil, errors.New("no content")
		}
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, errors.New("error downloading data")
		}
		var resp jsonhttp.StatusResponse
		err = json.Unmarshal(data, &resp)
		if err != nil {
			return nil, errors.New("error unmarshalling error response")
		}
		if response.StatusCode == http.StatusPaymentRequired {
			return data, nil
		}
		return nil, errors.New(resp.Message)
	}

	if len(response.Cookies()) > 0 {
		s.cookie = response.Cookies()[0]
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error downloading data")
	}

	var resp jsonhttp.StatusResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		errStr := fmt.Sprintf("error unmarshalling response: %d", len(data))
		return nil, errors.New(errStr)
	}

	if resp.Code == 0 {
		return data, nil
	}

	return []byte(resp.Message), nil
}

func (s *fdfsClient) getReq(urlPath, argsString string) ([]byte, error) {
	fullUrl := fmt.Sprintf(s.url + urlPath)
	var req *http.Request
	var err error
	if argsString != "" {
		fullUrl = fullUrl + "?" + argsString
		req, err = http.NewRequest(http.MethodGet, fullUrl, http.NoBody)
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequest(http.MethodGet, fullUrl, http.NoBody)
		if err != nil {
			return nil, err
		}
	}

	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}

	// execute the request
	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	req.Close = true

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		if response.StatusCode == http.StatusNoContent {
			return nil, errors.New("no content")
		}
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, errors.New("error downloading data")
		}
		var resp jsonhttp.StatusResponse
		err = json.Unmarshal(data, &resp)
		if err != nil {
			return nil, errors.New("error unmarshalling error response")
		}
		return nil, errors.New(resp.Message)
	}

	if len(response.Cookies()) > 0 {
		s.cookie = response.Cookies()[0]
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error downloading data")
	}

	var resp jsonhttp.StatusResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		errStr := fmt.Sprintf("error unmarshalling response: %d", len(data))
		return nil, errors.New(errStr)
	}
	if resp.Code == 0 {
		return data, nil
	}

	return []byte(resp.Message), nil
}

func (s *fdfsClient) uploadMultipartFile(urlPath, fileName string, fileSize int64, fd *os.File, arguments map[string]string, formFileArgument, compression string) ([]byte, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Add parameters
	for k, v := range arguments {
		err := writer.WriteField(k, v)
		if err != nil {
			return nil, err
		}
	}

	part, err := writer.CreateFormFile(formFileArgument, fileName)
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(part, fd)
	if err != nil {
		return nil, err
	}

	if n != fileSize {
		return nil, fmt.Errorf("partial write")
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	fullUrl := fmt.Sprintf(s.url + urlPath)
	req, err := http.NewRequest(http.MethodPost, fullUrl, body)
	if err != nil {
		return nil, err
	}

	contentType := fmt.Sprintf("multipart/form-data;boundary=%v", writer.Boundary())
	req.Header.Set("Content-Type", contentType)
	if compression != "" {
		compValue := strings.ToLower(compression)
		req.Header.Set(api.CompressionHeader, compValue)
	}

	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}

	// execute the request
	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	req.Close = true

	if response.StatusCode != http.StatusOK {
		errStr := fmt.Sprintf("received invalid status: %v", response.StatusCode)
		return nil, errors.New(errStr)
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("error downloading data")
	}

	return data, nil

}

func (s *fdfsClient) downloadMultipartFile(method, urlPath string, arguments map[string]string, out *os.File) (int64, error) {
	// prepare the  request
	fullUrl := fmt.Sprintf(s.url + urlPath)
	var req *http.Request
	var err error

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range arguments {
		err := writer.WriteField(k, v)
		if err != nil {
			return 0, err
		}
	}
	err = writer.Close()
	if err != nil {
		return 0, err
	}
	req, err = http.NewRequest(method, fullUrl, body)
	if err != nil {
		return 0, err
	}
	// add the headers

	contentType := fmt.Sprintf("multipart/form-data;boundary=%v", writer.Boundary())
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Content-Length", strconv.Itoa(len(body.Bytes())))

	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}

	// execute the request
	response, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	req.Close = true

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		errStr := fmt.Sprintf("received invalid status: %s", response.Status)
		return 0, errors.New(errStr)
	}

	// Write the body to file
	n, err := io.Copy(out, response.Body)
	if err != nil {
		return 0, err
	}
	return n, nil
}
