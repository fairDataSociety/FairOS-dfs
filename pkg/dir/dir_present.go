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

package dir

import (
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
)

// IsDirectoryPresent this function check if a given directory is present inside the pod.
func (d *Directory) IsDirectoryPresent(directoryNameWithPath string) bool {
	topic := utils.HashString(directoryNameWithPath)
	_, metaBytes, err := d.fd.GetFeedData(topic, d.userAddress)
	if string(metaBytes) == utils.DeletedFeedMagicWord {
		return false
	}
	return err == nil
}
