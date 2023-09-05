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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

var (
	// MetaVersion is the version of the meta data
	MetaVersion uint8 = 2

	// ErrDeletedFeed is returned when the feed is deleted
	ErrDeletedFeed = errors.New("deleted feed")

	// ErrUnknownFeed is returned when the feed is unknown
	ErrUnknownFeed = errors.New("unknown value in feed")
)

// MetaData is the structure of the file metadata
type MetaData struct {
	Version          uint8  `json:"version"`
	Path             string `json:"filePath"`
	Name             string `json:"fileName"`
	Size             uint64 `json:"fileSize"`
	BlockSize        uint32 `json:"blockSize"`
	ContentType      string `json:"contentType"`
	Compression      string `json:"compression"`
	CreationTime     int64  `json:"creationTime"`
	AccessTime       int64  `json:"accessTime"`
	ModificationTime int64  `json:"modificationTime"`
	InodeAddress     []byte `json:"fileInodeReference"`
	Mode             uint32 `json:"mode"`
}

// LoadFileMeta is used in syncing
func (f *File) LoadFileMeta(fileNameWithPath, podPassword string) error {
	meta, err := f.GetMetaFromFileName(fileNameWithPath, podPassword, f.userAddress)
	if err != nil { // skipcq: TCV-001
		if err == ErrDeletedFeed {
			return nil
		}
		return err
	}
	f.AddToFileMap(fileNameWithPath, meta)
	f.logger.Infof(fileNameWithPath)
	return nil
}

func (f *File) handleMeta(meta *MetaData, podPassword string) error {
	// check if meta is present.
	err := f.uploadMeta(meta, podPassword)
	if err != nil {
		return f.updateMeta(meta, podPassword)
	}
	return nil
}

func (f *File) uploadMeta(meta *MetaData, podPassword string) error {
	// marshall the meta structure
	fileMetaBytes, err := json.Marshal(meta)
	if err != nil { // skipcq: TCV-001
		return err
	}

	// put the file meta as a feed
	totalPath := utils.CombinePathAndFile(meta.Path, meta.Name)
	topic := utils.HashString(totalPath)
	_, err = f.fd.CreateFeed(f.userAddress, topic, fileMetaBytes, []byte(podPassword))
	return err
}

func (f *File) deleteMeta(meta *MetaData, podPassword string) error {
	totalPath := utils.CombinePathAndFile(meta.Path, meta.Name)
	topic := utils.HashString(totalPath)
	// update with utils.DeletedFeedMagicWord
	_, err := f.fd.UpdateFeed(f.userAddress, topic, []byte(utils.DeletedFeedMagicWord), []byte(podPassword), false)
	if err != nil { // skipcq: TCV-001
		return err
	}

	return nil
}

func (f *File) updateMeta(meta *MetaData, podPassword string) error {
	// marshall the meta structure
	fileMetaBytes, err := json.Marshal(meta)
	if err != nil { // skipcq: TCV-001
		return err
	}

	// put the file meta as a feed
	totalPath := utils.CombinePathAndFile(meta.Path, meta.Name)
	topic := utils.HashString(totalPath)
	_, err = f.fd.UpdateFeed(f.userAddress, topic, fileMetaBytes, []byte(podPassword), false)
	if err != nil { // skipcq: TCV-001
		return err
	}

	return nil
}

// BackupFromFileName is used to backup a file
func (f *File) BackupFromFileName(fileNameWithPath, podPassword string) (*MetaData, error) {
	p, err := f.GetMetaFromFileName(fileNameWithPath, podPassword, f.userAddress)
	if err != nil {
		return nil, err
	}

	err = f.deleteMeta(p, podPassword)
	if err != nil {
		return nil, err
	}

	// change previous meta.Name
	p.Name = fmt.Sprintf("%d_%s", time.Now().Unix(), p.Name)
	p.ModificationTime = time.Now().Unix()

	// upload PreviousMeta
	err = f.uploadMeta(p, podPassword)
	if err != nil {
		return nil, err
	}

	// add file to map
	f.AddToFileMap(utils.CombinePathAndFile(p.Path, p.Name), p)
	return p, nil
}

// RenameFromFileName is used to rename a file
func (f *File) RenameFromFileName(fileNameWithPath, newFileNameWithPath, podPassword string) (*MetaData, error) {
	fileNameWithPath = filepath.ToSlash(fileNameWithPath)
	newFileNameWithPath = filepath.ToSlash(newFileNameWithPath)
	p := f.GetInode(podPassword, fileNameWithPath)
	if p == nil {
		return nil, ErrFileNotFound
	}

	// remove old meta and from file map
	err := f.deleteMeta(p, podPassword)
	if err != nil {
		return nil, err
	}
	f.RemoveFromFileMap(fileNameWithPath)

	newFileName := filepath.Base(newFileNameWithPath)
	newPrnt := filepath.ToSlash(filepath.Dir(newFileNameWithPath))

	// change previous meta.Name
	p.Name = newFileName
	p.Path = newPrnt
	p.ModificationTime = time.Now().Unix()

	// upload meta
	err = f.handleMeta(p, podPassword)
	if err != nil {
		return nil, err
	}

	// add file to map
	f.AddToFileMap(newFileNameWithPath, p)
	return p, nil
}

// GetMetaFromFileName is used to get meta from file name
func (f *File) GetMetaFromFileName(fileNameWithPath, podPassword string, userAddress utils.Address) (*MetaData, error) {
	topic := utils.HashString(fileNameWithPath)
	_, metaBytes, err := f.fd.GetFeedData(topic, userAddress, []byte(podPassword), false)
	if err != nil {
		return nil, err
	}

	if string(metaBytes) == utils.DeletedFeedMagicWord {
		f.logger.Errorf("found deleted feed for %s\n", fileNameWithPath)
		return nil, ErrDeletedFeed
	}
	var meta *MetaData
	err = json.Unmarshal(metaBytes, &meta)
	if err != nil { // skipcq: TCV-001
		return nil, ErrUnknownFeed
	}

	return meta, nil
}

// PutMetaForFile is used to put meta for a file
func (f *File) PutMetaForFile(meta *MetaData, podPassword string) error {
	return f.updateMeta(meta, podPassword)
}
