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

package file

import (
	"sync"

	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

// File represents a file in a pod
type File struct {
	podName     string
	userAddress utils.Address
	client      blockstore.Client
	fd          *feed.API
	fileMap     map[string]*MetaData
	fileMu      *sync.RWMutex
	logger      logging.Logger
}

// NewFile creates the base file object which has all the methods related to file manipulation.
func NewFile(podName string, client blockstore.Client, fd *feed.API, user utils.Address, logger logging.Logger) *File {
	return &File{
		podName:     podName,
		userAddress: user,
		client:      client,
		fd:          fd,
		fileMap:     make(map[string]*MetaData),
		fileMu:      &sync.RWMutex{},
		logger:      logger,
	}
}

func (f *File) getClient() blockstore.Client {
	return f.client
}

// AddToFileMap adds a file metadata into fileMap
func (f *File) AddToFileMap(filePath string, meta *MetaData) {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()
	f.fileMap[filePath] = meta
}

// RemoveFromFileMap removes a file metadata from fileMap
func (f *File) RemoveFromFileMap(filePath string) {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()
	delete(f.fileMap, filePath)
}

// GetFromFileMap gets file metadata from the fileMap
func (f *File) GetFromFileMap(filePath string) *MetaData {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()
	if meta, ok := f.fileMap[filePath]; ok {
		return meta
	}
	return nil
}

// IsFileAlreadyPresent checks if a file is present in the fileMap
func (f *File) IsFileAlreadyPresent(fileWithPath string) bool {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()
	if _, ok := f.fileMap[fileWithPath]; ok {
		return true
	}
	return false
}

// RemoveAllFromFileMap resets the fileMap
func (f *File) RemoveAllFromFileMap() {
	f.fileMu.Lock()
	defer f.fileMu.Unlock()
	f.fileMap = make(map[string]*MetaData)
}
