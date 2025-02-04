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

package file_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/asabya/swarm-blockstore/bee"
	"github.com/asabya/swarm-blockstore/bee/mock"
	"github.com/ethersphere/bee/v2/pkg/file/redundancy"
	mockpost "github.com/ethersphere/bee/v2/pkg/postage/mock"
	mockstorer "github.com/ethersphere/bee/v2/pkg/storer/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/file"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileReader(t *testing.T) {
	storer := mockstorer.New()
	beeUrl := mock.NewTestBeeServer(t, mock.TestServerOptions{
		Storer:          storer,
		PreventRedirect: true,
		Post:            mockpost.New(mockpost.WithAcceptAll()),
	})

	mockClient := bee.NewBeeClient(beeUrl, bee.WithStamp(mock.BatchOkStr), bee.WithRedundancy(fmt.Sprintf("%d", redundancy.NONE)), bee.WithPinning(true))

	t.Run("read-entire-file-shorter-than-block", func(t *testing.T) {
		fileSize := uint64(15)
		blockSize := uint32(20)

		_, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		_, err := reader.Seek(10, 0)
		require.NoError(t, err)

		outputBytes := make([]byte, 3)
		n, err := reader.Read(outputBytes)
		require.NoError(t, err)

		assert.Equal(t, n, 3)
	})

	t.Run("read-entire-file-shorter-than-block-2", func(t *testing.T) {
		fileSize := uint64(15)
		blockSize := uint32(20)

		_, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		_, err := reader.Seek(10, 0)
		require.NoError(t, err)

		outputBytes := make([]byte, 10)
		_, err = reader.Read(outputBytes)
		if !errors.Is(err, io.EOF) {
			t.Fatal("should be EOF")
		}
	})

	t.Run("read-entire-file-shorter-than-block-3", func(t *testing.T) {
		fileSize := uint64(15)
		blockSize := uint32(20)

		_, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		_, err := reader.Seek(10, 0)
		require.NoError(t, err)

		outputBytes := make([]byte, 5)
		n, err := reader.Read(outputBytes)
		require.NoError(t, err)

		assert.Equal(t, n, 5)
	})

	t.Run("read-seek", func(t *testing.T) {
		fileSize := uint64(15)
		blockSize := uint32(20)

		_, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		_, err := reader.Seek(16, 0)
		if !errors.Is(err, file.ErrInvalidOffset) {
			t.Fatal("offset is invalid")
		}
	})

	t.Run("read-seek-offset-zero", func(t *testing.T) {
		fileSize := uint64(15)
		blockSize := uint32(20)

		_, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		_, err := reader.Seek(0, 0)
		require.NoError(t, err)

		outputBytes := make([]byte, 15)
		n, err := reader.Read(outputBytes)
		require.NoError(t, err)

		assert.Equal(t, n, 15)
	})

	t.Run("read-entire-file", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)

		b, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()

		outputBytes := readFileContents(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-file-with-last-block-shorter", func(t *testing.T) {
		fileSize := uint64(93)
		blockSize := uint32(10)

		b, fileInode := createFile(t, fileSize, blockSize, "", mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		outputBytes := readFileContents(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-gzip-file", func(t *testing.T) {
		fileSize := uint64(1638500)
		blockSize := uint32(163850)
		compression := "gzip"

		b, fileInode := createFile(t, fileSize, blockSize, compression, mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, false)
		defer reader.Close()
		outputBytes := readFileContents(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-gzip-file-with-last-block-shorter", func(t *testing.T) {
		fileSize := uint64(1999000)
		blockSize := uint32(200000)
		compression := "gzip"

		b, fileInode := createFile(t, fileSize, blockSize, compression, mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, false)
		defer reader.Close()
		outputBytes := readFileContents(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-snappy-file", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)
		compression := "snappy"

		b, fileInode := createFile(t, fileSize, blockSize, compression, mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, false)
		defer reader.Close()
		outputBytes := readFileContents(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-snappy-file-with-last-block-shorter", func(t *testing.T) {
		fileSize := uint64(93)
		blockSize := uint32(10)
		compression := "snappy"

		b, fileInode := createFile(t, fileSize, blockSize, compression, mockClient)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, false)
		defer reader.Close()
		outputBytes := readFileContents(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-lines", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)

		b, fileInode, _, _, _, _ := createFileWithNewlines(t, fileSize, blockSize, "", mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		outputBytes := readFileContentsUsingReadline(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-lines-with-last-block-shorter", func(t *testing.T) {
		fileSize := uint64(97)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)

		b, fileInode, _, _, _, _ := createFileWithNewlines(t, fileSize, blockSize, "", mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		outputBytes := readFileContentsUsingReadline(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("read-lines-with-last-block-shorter-and-compressed", func(t *testing.T) {
		fileSize := uint64(97)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)
		compression := "snappy"

		b, fileInode, _, _, _, _ := createFileWithNewlines(t, fileSize, blockSize, compression, mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, false)
		defer reader.Close()
		outputBytes := readFileContentsUsingReadline(t, fileSize, reader)
		assert.Equal(t, b, outputBytes)
	})

	t.Run("seek-and-read-line", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)

		_, fileInode, lineStart, line, _, _ := createFileWithNewlines(t, fileSize, blockSize, "", mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		seekN, err := reader.Seek(int64(lineStart), 0)
		require.NoError(t, err)
		assert.Equal(t, seekN, int64(lineStart))

		buf, err := reader.ReadLine()
		require.NoError(t, err)
		assert.Equal(t, buf, line)
	})

	t.Run("seek-and-read-line-spanning-block-boundary", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)

		_, fileInode, _, _, lineStart, line := createFileWithNewlines(t, fileSize, blockSize, "", mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, "", false)
		defer reader.Close()
		seekN, err := reader.Seek(int64(lineStart), 0)
		require.NoError(t, err)
		assert.Equal(t, seekN, int64(lineStart))

		buf, err := reader.ReadLine()
		require.NoError(t, err)
		assert.Equal(t, buf, line)
	})

	t.Run("seek-and-read-line-spanning-block-boundary-with-compression", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)
		compression := "snappy"

		_, fileInode, _, _, lineStart, line := createFileWithNewlines(t, fileSize, blockSize, compression, mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, false)
		defer reader.Close()
		seekN, err := reader.Seek(int64(lineStart), 0)
		require.NoError(t, err)
		assert.Equal(t, seekN, int64(lineStart))

		buf, err := reader.ReadLine()
		require.NoError(t, err)
		assert.Equal(t, buf, line)
	})

	t.Run("seek-and-read-line-spanning-block-boundary-with-compression-with-cache", func(t *testing.T) {
		fileSize := uint64(100)
		blockSize := uint32(10)
		linesPerBlock := uint32(2)
		compression := "snappy"

		_, fileInode, _, _, lineStart, line := createFileWithNewlines(t, fileSize, blockSize, compression, mockClient, linesPerBlock)
		reader := file.NewReader(fileInode, mockClient, fileSize, blockSize, compression, true)
		defer reader.Close()
		seekN, err := reader.Seek(int64(lineStart), 0)
		require.NoError(t, err)
		assert.Equal(t, seekN, int64(lineStart))

		buf, err := reader.ReadLine()
		require.NoError(t, err)
		assert.Equal(t, buf, line)

		// this should come from cache
		seekN, err = reader.Seek(int64(lineStart), 0)
		require.NoError(t, err)
		assert.Equal(t, seekN, int64(lineStart))

		buf, err = reader.ReadLine()
		require.NoError(t, err)
		assert.Equal(t, buf, line)
	})
}

func createFile(t *testing.T, fileSize uint64, blockSize uint32, compression string, mockClient *bee.Client) ([]byte, file.INode) {
	var fileBlocks []*file.BlockInfo
	noOfBlocks := fileSize / uint64(blockSize)
	if fileSize%uint64(blockSize) != 0 {
		noOfBlocks += 1
	}
	var content []byte
	bytesRemaining := fileSize
	for i := uint64(0); i < noOfBlocks; i++ {
		bytesToWrite := blockSize
		if bytesRemaining < uint64(blockSize) {
			bytesToWrite = uint32(bytesRemaining)
		}
		buf := make([]byte, bytesToWrite)
		_, err := rand.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		content = append(content, buf...)
		if compression != "" {
			compressedData, err := file.Compress(buf, compression, bytesToWrite)
			if err != nil {
				t.Fatal(err)
			}
			buf = compressedData
		}

		addr, err := mockClient.UploadBlob(0, "", "0", false, true, bytes.NewReader(buf))
		if err != nil {
			t.Fatal(err)
		}
		fileBlock := &file.BlockInfo{
			Size:           bytesToWrite,
			CompressedSize: uint32(len(buf)),
			Reference:      utils.NewReference(addr.Bytes()),
		}
		fileBlocks = append(fileBlocks, fileBlock)
		bytesRemaining -= uint64(bytesToWrite)
	}

	return content, file.INode{
		Blocks: fileBlocks,
	}
}

func createFileWithNewlines(t *testing.T, fileSize uint64, blockSize uint32, compression string, mockClient *bee.Client, linesPerBlock uint32) ([]byte, file.INode, int, []byte, int, []byte) { // skipcq: GO-C4008
	var fileBlocks []*file.BlockInfo
	noOfBlocks := fileSize / uint64(blockSize)
	if fileSize%uint64(blockSize) != 0 {
		noOfBlocks += 1
	}
	bytesRemaining := fileSize

	randomLineStartPoint := 0
	var randomLine []byte

	borderCrossingLineStartingPoint := 0
	var borderCrossingLine []byte

	bytesWritten := 0
	var content []byte

	for i := uint64(0); i < noOfBlocks; i++ {
		bytesToWrite := blockSize
		if bytesRemaining < uint64(blockSize) {
			bytesToWrite = uint32(bytesRemaining)
		}
		buf := make([]byte, bytesToWrite)
		_, err := rand.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		for j := uint32(0); j < linesPerBlock; j++ {
			bi, err := rand.Int(rand.Reader, big.NewInt(int64(bytesToWrite)))
			if err != nil {
				t.Fatal(err)
			}
			idx := bi.Int64()
			if buf[idx] == '\n' {
				bi, err = rand.Int(rand.Reader, big.NewInt(int64(bytesToWrite)))
				if err != nil {
					t.Fatal(err)
				}
				idx = bi.Int64()
			}
			buf[idx] = '\n'
		}
		if buf[int64(bytesToWrite)-1] == 10 {
			buf[int64(bytesToWrite)-1] = 11
		}
		if i == 2 {
			start := false
			startIndex := 0
			endIndex := 0
			for k, ch := range buf {
				if ch == '\n' && start {
					endIndex = k + 1
					break
				}
				if ch == '\n' && !start {
					startIndex = k + 1
					randomLineStartPoint = (int(blockSize) * int(i)) + startIndex
					start = true
				}
			}
			if startIndex > endIndex {
				startIndex, endIndex = endIndex, startIndex
				randomLineStartPoint = (int(blockSize) * int(i)) + startIndex
			}
			randomLine = append(randomLine, buf[startIndex:endIndex]...)
		}

		gotFromFirstBlock := false
		if i >= 4 && borderCrossingLineStartingPoint == 0 && buf[int(blockSize)-1] != '\n' {
			gotFirstNewLine := false
			startIndex := 0
			for k, ch := range buf {
				if ch == '\n' {
					if !gotFirstNewLine {
						gotFirstNewLine = true
						continue
					}
					startIndex = k + 1
					borderCrossingLineStartingPoint = (int(blockSize) * int(i)) + startIndex
				}
			}
			if borderCrossingLineStartingPoint != 0 {
				borderCrossingLine = append(borderCrossingLine, buf[startIndex:]...)
				gotFromFirstBlock = true
			}
		}

		if i >= 4 && !gotFromFirstBlock && borderCrossingLine != nil && borderCrossingLine[len(borderCrossingLine)-1] != '\n' {
			endIndex := 0
			for k, ch := range buf {
				if ch == '\n' {
					endIndex = k + 1
					borderCrossingLine = append(borderCrossingLine, buf[:endIndex]...)
					break
				}
			}
		}
		content = append(content, buf...)
		if compression != "" {
			compressedData, err := file.Compress(buf, compression, bytesToWrite)
			if err != nil {
				t.Fatal(err)
			}
			buf = compressedData
		}

		addr, err := mockClient.UploadBlob(0, "", "0", false, true, bytes.NewReader(buf))
		if err != nil {
			t.Fatal(err)
		}
		fileBlock := &file.BlockInfo{
			Size:           bytesToWrite,
			CompressedSize: uint32(len(buf)),
			Reference:      utils.NewReference(addr.Bytes()),
		}
		fileBlocks = append(fileBlocks, fileBlock)
		bytesRemaining -= uint64(bytesToWrite)
		bytesWritten += int(bytesToWrite)
	}
	return content, file.INode{
		Blocks: fileBlocks,
	}, randomLineStartPoint, randomLine, borderCrossingLineStartingPoint, borderCrossingLine
}

func readFileContents(t *testing.T, fileSize uint64, reader *file.Reader) []byte {
	outputBytes := make([]byte, fileSize)
	count := uint64(0)
	for count < fileSize {
		n, err := reader.Read(outputBytes[count:])
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
		}
		count += uint64(n)
	}
	return outputBytes
}

func readFileContentsUsingReadline(t *testing.T, fileSize uint64, reader *file.Reader) []byte {
	var outputBytes []byte
	count := uint64(0)
	for count < fileSize {
		buf, err := reader.ReadLine()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
		}
		count += uint64(len(buf))
		outputBytes = append(outputBytes, buf...)
	}
	return outputBytes
}
