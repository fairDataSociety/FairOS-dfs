package dir

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/pkg/file"

	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

// RenameDir renames a directory
func (d *Directory) RenameDir(dirNameWithPath, newDirNameWithPath, podPassword string) error {
	dirNameWithPath = filepath.ToSlash(dirNameWithPath)
	newDirNameWithPath = filepath.ToSlash(newDirNameWithPath)
	parentPath := filepath.ToSlash(filepath.Dir(dirNameWithPath))
	dirName := filepath.Base(dirNameWithPath)

	newParentPath := filepath.ToSlash(filepath.Dir(newDirNameWithPath))
	newDirName := filepath.Base(newDirNameWithPath)

	// validation checks of the arguments
	if dirName == "" || strings.HasPrefix(dirName, utils.PathSeparator) { // skipcq: TCV-001
		return ErrInvalidDirectoryName
	}

	if len(dirName) > nameLength { // skipcq: TCV-001
		return ErrTooLongDirectoryName
	}

	if dirName == "/" {
		return fmt.Errorf("cannot rename root dir")
	}

	// check if directory exists
	_, err := d.GetInode(podPassword, dirNameWithPath)
	if err != nil { // skipcq: TCV-001
		return ErrDirectoryNotPresent
	}

	// check if parent directory exists
	_, err = d.GetInode(podPassword, parentPath)
	if err != nil { // skipcq: TCV-001
		return ErrDirectoryNotPresent
	}
	_, err = d.GetInode(podPassword, newDirNameWithPath)
	if err == nil {
		return ErrDirectoryAlreadyPresent
	}

	err = d.mapChildrenToNewPath(dirNameWithPath, newDirNameWithPath, podPassword)
	if err != nil { // skipcq: TCV-001
		return err
	}

	inode, err := d.GetInode(podPassword, dirNameWithPath)
	if err != nil { // skipcq: TCV-001
		return err
	}

	inode.Meta.Name = newDirName
	inode.Meta.Path = newParentPath
	inode.Meta.ModificationTime = time.Now().Unix()

	// upload meta
	fileMetaBytes, err := json.Marshal(inode)
	if err != nil { // skipcq: TCV-001
		return err
	}

	err = d.file.Upload(bufio.NewReader(strings.NewReader(string(fileMetaBytes))), IndexFileName, int64(len(fileMetaBytes)), file.MinBlockSize, 0, newDirNameWithPath, "gzip", podPassword)
	if err != nil { // skipcq: TCV-001
		return err
	}

	err = d.file.RmFile(utils.CombinePathAndFile(dirNameWithPath, IndexFileName), podPassword)
	if err != nil { // skipcq: TCV-001
		return err
	}

	d.RemoveFromDirectoryMap(dirNameWithPath)

	// get the parent directory entry and add this new directory to its list of children
	err = d.RemoveEntryFromDir(parentPath, podPassword, dirName, false)
	if err != nil {
		return err
	}
	err = d.AddEntryToDir(newParentPath, podPassword, newDirName, false)
	if err != nil {
		return err
	}

	err = d.SyncDirectory(parentPath, podPassword)
	if err != nil {
		return err
	}

	if parentPath != newParentPath {
		err = d.SyncDirectory(newParentPath, podPassword)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Directory) mapChildrenToNewPath(totalPath, newTotalPath, podPassword string) error {
	dirInode := d.GetDirFromDirectoryMap(totalPath)
	for _, fileOrDirName := range dirInode.FileOrDirNames {
		if strings.HasPrefix(fileOrDirName, "_F_") {
			fileName := strings.TrimPrefix(fileOrDirName, "_F_")
			filePath := utils.CombinePathAndFile(totalPath, fileName)
			newFilePath := utils.CombinePathAndFile(newTotalPath, fileName)
			topic := utils.HashString(filePath)
			_, metaBytes, err := d.fd.GetFeedData(topic, d.userAddress, []byte(podPassword), false)
			if err != nil {
				return err
			}
			if string(metaBytes) == utils.DeletedFeedMagicWord {
				continue
			}

			p := &file.MetaData{}
			err = json.Unmarshal(metaBytes, p)
			if err != nil { // skipcq: TCV-001
				return err
			}
			newTopic := utils.HashString(newFilePath)
			// change previous meta.Name
			p.Path = newTotalPath
			p.ModificationTime = time.Now().Unix()
			// upload meta
			fileMetaBytes, err := json.Marshal(p)
			if err != nil { // skipcq: TCV-001
				return err
			}

			previousAddr, _, err := d.fd.GetFeedData(newTopic, d.userAddress, []byte(podPassword), false)
			if err == nil && previousAddr != nil {
				err = d.fd.UpdateFeed(d.userAddress, newTopic, fileMetaBytes, []byte(podPassword), false)
				if err != nil { // skipcq: TCV-001
					return err
				}
			} else {
				err = d.fd.CreateFeed(d.userAddress, newTopic, fileMetaBytes, []byte(podPassword))
				if err != nil { // skipcq: TCV-001
					return err
				}
			}

			// delete old meta
			// update with utils.DeletedFeedMagicWord
			err = d.fd.UpdateFeed(d.userAddress, topic, []byte(utils.DeletedFeedMagicWord), []byte(podPassword), false)
			if err != nil { // skipcq: TCV-001
				return err
			}
		} else if strings.HasPrefix(fileOrDirName, "_D_") {
			dirName := strings.TrimPrefix(fileOrDirName, "_D_")
			pathWithDir := utils.CombinePathAndFile(totalPath, dirName)
			newPathWithDir := utils.CombinePathAndFile(newTotalPath, dirName)
			err := d.mapChildrenToNewPath(pathWithDir, newPathWithDir, podPassword)
			if err != nil { // skipcq: TCV-001
				return err
			}

			inode, err := d.GetInode(podPassword, pathWithDir)
			if err != nil { // skipcq: TCV-001
				return err
			}

			inode.Meta.Path = newTotalPath
			inode.Meta.ModificationTime = time.Now().Unix()

			err = d.SetInode(podPassword, inode)
			if err != nil { // skipcq: TCV-001
				return err
			}

			// delete old meta
			err = d.RemoveInode(podPassword, pathWithDir)
			if err != nil { // skipcq: TCV-001
				return err
			}
		}
	}
	return nil
}
