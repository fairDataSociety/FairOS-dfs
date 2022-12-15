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

import "errors"

var (
	//ErrInvalidDirectoryName
	ErrInvalidDirectoryName = errors.New("invalid directory name")
	//ErrTooLongDirectoryName
	ErrTooLongDirectoryName = errors.New("too long directory name")
	//ErrDirectoryAlreadyPresent
	ErrDirectoryAlreadyPresent = errors.New("directory name already present")
	//ErrDirectoryNotPresent
	ErrDirectoryNotPresent = errors.New("directory not present")

	//ErrInvalidFileOrDirectoryName
	ErrInvalidFileOrDirectoryName = errors.New("invalid file or directory name")
)
