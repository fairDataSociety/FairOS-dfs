package test_test

import (
	"crypto/rand"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/cmd/common"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dfs"
	mock2 "github.com/fairdatasociety/fairOS-dfs/pkg/ensm/eth/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/user"
	"github.com/sirupsen/logrus"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterRunes))))
		if err != nil {
			return string(b)
		}
		b[i] = letterRunes[num.Int64()]
	}
	return string(b)
}

func TestLiteUser(t *testing.T) {
	mockClient := mock.NewMockBeeClient()
	ens := mock2.NewMockNamespaceManager()
	logger := logging.New(io.Discard, logrus.ErrorLevel)
	dataDir, err := os.MkdirTemp("", "new")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dataDir)
	users := user.NewUsers(dataDir, mockClient, ens, logger)
	dfsApi := dfs.NewMockDfsAPI(mockClient, users, logger, dataDir)

	t.Run("signup-login-pod-dir-file-rename", func(t *testing.T) {
		userRequest := &common.UserSignupRequest{
			UserName: randStringRunes(16),
			Password: randStringRunes(8),
		}

		mnemonic, _, ui, err := dfsApi.LoadLiteUser(userRequest.UserName, userRequest.Password, "", "")
		if err != nil {
			t.Fatal(err)
		}

		sessionId := ui.GetSessionId()

		// pod new
		podRequest := &common.PodRequest{
			PodName: randStringRunes(16),
		}

		_, err = dfsApi.CreatePod(podRequest.PodName, sessionId)
		if err != nil {
			t.Fatal(err)
		}

		entries := []struct {
			path    string
			isDir   bool
			size    int64
			content []byte
		}{
			{
				path:  "/dir1",
				isDir: true,
			},
			{
				path:  "/dir2",
				isDir: true,
			},
			{
				path:  "/dir3",
				isDir: true,
			},
			{
				path: "/file1",
				size: 1024 * 1024,
			},
			{
				path: "/dir1/file11",
				size: 1024 * 512,
			},
			{
				path: "/dir1/file12",
				size: 1024 * 1024,
			},
			{
				path: "/dir3/file31",
				size: 1024 * 1024,
			},
			{
				path: "/dir3/file32",
				size: 1024 * 1024,
			},
			{
				path: "/dir3/file33",
				size: 1024,
			},
			{
				path:  "/dir2/dir4",
				isDir: true,
			},
			{
				path:  "/dir2/dir4/dir5",
				isDir: true,
			},
			{
				path: "/dir2/dir4/file241",
				size: 5 * 1024 * 1024,
			},
			{
				path: "/dir2/dir4/dir5/file2451",
				size: 10 * 1024 * 1024,
			},
		}

		for _, v := range entries {
			if v.isDir {

				err = dfsApi.Mkdir(podRequest.PodName, v.path, sessionId)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				reader := &io.LimitedReader{R: rand.Reader, N: v.size}
				err = dfsApi.UploadFile(podRequest.PodName, filepath.Base(v.path), sessionId, v.size, reader, filepath.Dir(v.path), "", 100000, false)
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		for _, v := range entries {
			if v.isDir {
				_, err := dfsApi.DirectoryStat(podRequest.PodName, v.path, sessionId)
				if err != nil {
					t.Fatal(err)

				}
			} else {
				_, err := dfsApi.FileStat(podRequest.PodName, v.path, sessionId)
				if err != nil {
					t.Fatal(err)

				}
			}
		}
		// rename  file "/dir2/dir4/dir5/file2451" => "/dir2/dir4/dir5/file24511"
		renames := []struct {
			oldPath string
			newPath string
			isDir   bool
		}{
			{
				oldPath: "/dir2/dir4/dir5/file2451",
				newPath: "/dir2/dir4/dir5/file24511",
				isDir:   false,
			},
			{
				oldPath: "/dir2/dir4/dir5/file24511",
				newPath: "/file24511",
				isDir:   false,
			},
			{
				oldPath: "/dir2",
				newPath: "/dir2020",
				isDir:   true,
			},
			{
				oldPath: "/dir2020/dir4",
				newPath: "/dir2020/dir4040",
				isDir:   true,
			}, {
				oldPath: "/dir3/file33",
				newPath: "/dir2020/file33",
				isDir:   false,
			}, {
				oldPath: "/dir1/file12",
				newPath: "/dir2020/dir4040/file12",
				isDir:   false,
			},
		}
		for _, v := range renames {
			if v.isDir {
				err = dfsApi.RenameDir(podRequest.PodName, v.oldPath, v.newPath, sessionId)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				err = dfsApi.RenameFile(podRequest.PodName, v.oldPath, v.newPath, sessionId)
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		newEntries := []struct {
			path    string
			isDir   bool
			size    int64
			content []byte
		}{
			{
				path:  "/dir1",
				isDir: true,
			},
			{
				path:  "/dir2020",
				isDir: true,
			},
			{
				path:  "/dir3",
				isDir: true,
			},
			{
				path: "/file1",
				size: 1024 * 1024,
			},
			{
				path: "/dir1/file11",
				size: 1024 * 512,
			},
			{
				path: "/dir2020/dir4040/file12",
				size: 1024 * 1024,
			},
			{
				path: "/dir3/file31",
				size: 1024 * 1024,
			},
			{
				path: "/dir3/file32",
				size: 1024 * 1024,
			},
			{
				path: "/dir2020/file33",
				size: 1024,
			},
			{
				path:  "/dir2020/dir4040",
				isDir: true,
			},
			{
				path:  "/dir2020/dir4040/dir5",
				isDir: true,
			},
			{
				path: "/dir2020/dir4040/file241",
				size: 5 * 1024 * 1024,
			},
			{
				path: "/file24511",
				size: 10 * 1024 * 1024,
			},
		}
		for _, v := range newEntries {
			if v.isDir {
				_, err := dfsApi.DirectoryStat(podRequest.PodName, v.path, sessionId)
				if err != nil {
					t.Fatal(err)

				}
			} else {
				_, err := dfsApi.FileStat(podRequest.PodName, v.path, sessionId)
				if err != nil {
					t.Fatal(err)

				}
			}
		}

		err = dfsApi.LogoutUser(sessionId)
		if err != nil {
			t.Fatal(err)
		}

		<-time.After(time.Second)

		for i := 0; i < 20; i++ {
			_, _, ui, err = dfsApi.LoadLiteUser(userRequest.UserName, userRequest.Password, mnemonic, "")
			if err != nil {
				t.Fatal(err)
			}

			sessionId = ui.GetSessionId()

			_, err = dfsApi.OpenPod(podRequest.PodName, sessionId)
			if err != nil {
				t.Fatal(err)
			}
			for _, v := range newEntries {
				if v.isDir {
					_, err := dfsApi.DirectoryStat(podRequest.PodName, v.path, sessionId)
					if err != nil {
						t.Fatal(err)

					}
				} else {
					_, err := dfsApi.FileStat(podRequest.PodName, v.path, sessionId)
					if err != nil {
						t.Fatal(err)

					}
				}
			}

			err = dfsApi.LogoutUser(sessionId)
			if err != nil {
				t.Fatal(err)
			}

			<-time.After(time.Second)
		}
	})
}
