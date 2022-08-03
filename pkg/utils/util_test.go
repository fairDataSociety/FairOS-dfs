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

package utils

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestAddress(t *testing.T) {
	buf := make([]byte, 4096)
	_, err := rand.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	ch, err := NewChunkWithSpan(buf)
	if err != nil {
		t.Fatal(err)
	}

	refBytes := ch.Address().Bytes()
	ref := NewReference(refBytes)
	refHexString := ref.String()
	newRef, err := ParseHexReference(refHexString)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(refBytes, newRef.Bytes()) {
		t.Fatalf("bytes are not equal")
	}
}

func TestChunkLength(t *testing.T) {
	buf := make([]byte, 5000)
	_, err := rand.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewChunkWithSpan(buf)
	if err != nil && err.Error() != "max chunk size exceeded" {
		t.Fatal("error should be \"max chunk size exceeded\"")
	}
}
