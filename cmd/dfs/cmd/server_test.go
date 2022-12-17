package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fairdatasociety/fairOS-dfs/cmd/common"
	"github.com/fairdatasociety/fairOS-dfs/pkg/api"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/dfs"
	mock2 "github.com/fairdatasociety/fairOS-dfs/pkg/ensm/eth/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
	"github.com/fairdatasociety/fairOS-dfs/pkg/user"
	"github.com/gorilla/websocket"
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

func TestApis(t *testing.T) {
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
	handler = api.NewMockHandler(dfsApi, logger, []string{"http://localhost:3000"})
	httpPort = ":9090"
	pprofPort = ":9091"
	base := "localhost:9090"
	basev1 := "http://localhost:9090/v1"
	basev2 := "http://localhost:9090/v2"
	go startHttpService(logger)

	// wait 10 seconds for the server to start
	<-time.After(time.Second * 10)
	t.Run("login-fail-test", func(t *testing.T) {
		c := http.Client{Timeout: time.Duration(1) * time.Minute}
		userRequest := &common.UserSignupRequest{
			UserName: randStringRunes(16),
			Password: randStringRunes(8),
		}
		userBytes, err := json.Marshal(userRequest)
		if err != nil {
			t.Fatal(err)
		}
		userLoginHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserLogin)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)

		}
		userLoginHttpReq.Header.Add("Content-Type", "application/json")
		userLoginHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		userLoginResp, err := c.Do(userLoginHttpReq)
		if err != nil {
			t.Fatal(err)
		}
		err = userLoginResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if userLoginResp.StatusCode == http.StatusOK {
			t.Fatal("user should not be able to login")
		}
	})

	t.Run("signup-login", func(t *testing.T) {
		c := http.Client{Timeout: time.Duration(1) * time.Minute}
		userRequest := &common.UserSignupRequest{
			UserName: randStringRunes(16),
			Password: randStringRunes(8),
		}

		userBytes, err := json.Marshal(userRequest)
		if err != nil {
			t.Fatal(err)
		}

		signupRequestDataHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserSignup)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)
		}
		signupRequestDataHttpReq.Header.Add("Content-Type", "application/json")
		signupRequestDataHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		signupRequestResp, err := c.Do(signupRequestDataHttpReq)
		if err != nil {
			t.Fatal(err)
		}

		err = signupRequestResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if signupRequestResp.StatusCode != http.StatusCreated {
			t.Fatal("Signup failed", signupRequestResp.StatusCode)
		}

		userLoginHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserLogin)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)

		}
		userLoginHttpReq.Header.Add("Content-Type", "application/json")
		userLoginHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		userLoginResp, err := c.Do(userLoginHttpReq)
		if err != nil {
			t.Fatal(err)
		}
		err = userLoginResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if userLoginResp.StatusCode != http.StatusOK {
			t.Fatal("user should be able to login")
		}
	})

	t.Run("signup-login-logout-loggedin", func(t *testing.T) {
		c := http.Client{Timeout: time.Duration(1) * time.Minute}
		userRequest := &common.UserSignupRequest{
			UserName: randStringRunes(16),
			Password: randStringRunes(8),
		}

		userBytes, err := json.Marshal(userRequest)
		if err != nil {
			t.Fatal(err)
		}

		signupRequestDataHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserSignup)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)
		}
		signupRequestDataHttpReq.Header.Add("Content-Type", "application/json")
		signupRequestDataHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		signupRequestResp, err := c.Do(signupRequestDataHttpReq)
		if err != nil {
			t.Fatal(err)
		}

		err = signupRequestResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if signupRequestResp.StatusCode != http.StatusCreated {
			t.Fatal("Signup failed", signupRequestResp.StatusCode)
		}

		userLoginHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserLogin)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)

		}
		userLoginHttpReq.Header.Add("Content-Type", "application/json")
		userLoginHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		userLoginResp, err := c.Do(userLoginHttpReq)
		if err != nil {
			t.Fatal(err)
		}
		err = userLoginResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if userLoginResp.StatusCode != http.StatusOK {
			t.Fatal("user should be able to login")
		}

		cookie := userLoginResp.Header["Set-Cookie"]
		userLogoutHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserLogout)), http.NoBody)
		if err != nil {
			t.Fatal(err)

		}
		userLogoutHttpReq.Header.Set("Cookie", cookie[0])
		userLogoutResp, err := c.Do(userLogoutHttpReq)
		if err != nil {
			t.Fatal(err)
		}

		err = userLogoutResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		userIsLoggedInHttpReq, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s%s?username%s", basev1, string(common.UserIsLoggedin), userRequest.UserName), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		userIsLoggedInResp, err := c.Do(userIsLoggedInHttpReq)
		if err != nil {
			t.Fatal(err)
		}
		useIsLoggedBodyBytes, err := io.ReadAll(userIsLoggedInResp.Body)
		if err != nil {
			t.Fatal(err)
		}

		err = userIsLoggedInResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		loggedInStatus := &api.LoginStatus{}
		err = json.Unmarshal(useIsLoggedBodyBytes, loggedInStatus)
		if err != nil {
			t.Fatal(err)
		}

		if loggedInStatus.LoggedIn {
			t.Fatal("user should be logged out")
		}
	})

	t.Run("signup-login-pod-dir-file-rename", func(t *testing.T) {
		c := http.Client{Timeout: time.Duration(1) * time.Minute}
		userRequest := &common.UserSignupRequest{
			UserName: randStringRunes(16),
			Password: randStringRunes(8),
		}

		userBytes, err := json.Marshal(userRequest)
		if err != nil {
			t.Fatal(err)
		}

		signupRequestDataHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserSignup)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)
		}
		signupRequestDataHttpReq.Header.Add("Content-Type", "application/json")
		signupRequestDataHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		signupRequestResp, err := c.Do(signupRequestDataHttpReq)
		if err != nil {
			t.Fatal(err)
		}

		err = signupRequestResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if signupRequestResp.StatusCode != http.StatusCreated {
			t.Fatal("Signup failed", signupRequestResp.StatusCode)
		}

		userLoginHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserLogin)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)

		}
		userLoginHttpReq.Header.Add("Content-Type", "application/json")
		userLoginHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))
		userLoginResp, err := c.Do(userLoginHttpReq)
		if err != nil {
			t.Fatal(err)
		}
		err = userLoginResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if userLoginResp.StatusCode != http.StatusOK {
			t.Fatal("user should be able to login")
		}

		cookie := userLoginResp.Header["Set-Cookie"]

		// pod new
		podRequest := &common.PodRequest{
			PodName:  randStringRunes(16),
			Password: userRequest.Password,
		}
		podBytes, err := json.Marshal(podRequest)
		if err != nil {
			t.Fatal(err)
		}
		podNewHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev1, string(common.PodNew)), bytes.NewBuffer(podBytes))
		if err != nil {
			t.Fatal(err)

		}
		podNewHttpReq.Header.Set("Cookie", cookie[0])
		podNewHttpReq.Header.Add("Content-Type", "application/json")
		podNewHttpReq.Header.Add("Content-Length", strconv.Itoa(len(podBytes)))
		podNewResp, err := c.Do(podNewHttpReq)
		if err != nil {
			t.Fatal(err)
		}

		err = podNewResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if podNewResp.StatusCode != 201 {
			t.Fatal("pod creation failed")
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
				mkdirRqst := common.FileSystemRequest{
					PodName:       podRequest.PodName,
					DirectoryPath: v.path,
				}
				mkDirBytes, err := json.Marshal(mkdirRqst)
				if err != nil {
					t.Fatal(err)
				}
				mkDirHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev1, string(common.DirMkdir)), bytes.NewBuffer(mkDirBytes))
				if err != nil {
					t.Fatal(err)

				}
				mkDirHttpReq.Header.Set("Cookie", cookie[0])
				mkDirHttpReq.Header.Add("Content-Type", "application/json")
				mkDirHttpReq.Header.Add("Content-Length", strconv.Itoa(len(mkDirBytes)))
				mkDirResp, err := c.Do(mkDirHttpReq)
				if err != nil {
					t.Fatal(err)
				}
				err = mkDirResp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if mkDirResp.StatusCode != 201 {
					t.Fatal("mkdir failed")
				}
			} else {
				body := new(bytes.Buffer)
				writer := multipart.NewWriter(body)
				contentLength := fmt.Sprintf("%d", v.size)

				err = writer.WriteField("podName", podRequest.PodName)
				if err != nil {
					t.Fatal(err)
				}
				err = writer.WriteField("contentLength", contentLength)
				if err != nil {
					t.Fatal(err)
				}
				err = writer.WriteField("dirPath", filepath.Dir(v.path))
				if err != nil {
					t.Fatal(err)
				}
				err = writer.WriteField("blockSize", "4kb")
				if err != nil {
					t.Fatal(err)
				}
				part, err := writer.CreateFormFile("files", filepath.Base(v.path))
				if err != nil {
					t.Fatal(err)
				}
				reader := &io.LimitedReader{R: rand.Reader, N: v.size}
				_, err = io.Copy(part, reader)
				if err != nil {
					t.Fatal(err)
				}
				err = writer.Close()
				if err != nil {
					t.Fatal(err)
				}

				uploadReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev1, string(common.FileUpload)), body)
				if err != nil {
					t.Fatal(err)

				}
				uploadReq.Header.Set("Cookie", cookie[0])
				contentType := fmt.Sprintf("multipart/form-data;boundary=%v", writer.Boundary())
				uploadReq.Header.Add("Content-Type", contentType)
				uploadResp, err := c.Do(uploadReq)
				if err != nil {
					t.Fatal(err)
				}
				err = uploadResp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if uploadResp.StatusCode != 200 {
					t.Fatal("upload failed")
				}
			}
		}

		for _, v := range entries {
			if v.isDir {
				statReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s?podName=%s&dirPath=%s", basev1, string(common.DirStat), podRequest.PodName, v.path), http.NoBody)
				if err != nil {
					t.Fatal(err)

				}
				statReq.Header.Set("Cookie", cookie[0])
				statResp, err := c.Do(statReq)
				if err != nil {
					t.Fatal(err)
				}
				err = statResp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if statResp.StatusCode != 200 {
					t.Fatal("dir stat failed")
				}
			} else {
				statReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s?podName=%s&filePath=%s", basev1, string(common.FileStat), podRequest.PodName, v.path), http.NoBody)
				if err != nil {
					t.Fatal(err)

				}
				statReq.Header.Set("Cookie", cookie[0])
				statResp, err := c.Do(statReq)
				if err != nil {
					t.Fatal(err)
				}
				err = statResp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if statResp.StatusCode != 200 {
					t.Fatal("file stat failed")
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
			renameReq := common.RenameRequest{
				PodName: podRequest.PodName,
				OldPath: v.oldPath,
				NewPath: v.newPath,
			}

			renameBytes, err := json.Marshal(renameReq)
			if err != nil {
				t.Fatal(err)
			}
			url := common.FileRename
			if v.isDir {
				url = common.DirRename
			}
			renameHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev1, string(url)), bytes.NewBuffer(renameBytes))
			if err != nil {
				t.Fatal(err)

			}
			renameHttpReq.Header.Set("Cookie", cookie[0])
			renameHttpReq.Header.Add("Content-Type", "application/json")
			renameHttpReq.Header.Add("Content-Length", strconv.Itoa(len(renameBytes)))
			renameResp, err := c.Do(renameHttpReq)
			if err != nil {
				t.Fatal(err)
			}
			err = renameResp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if renameResp.StatusCode != 200 {
				t.Fatal("rename failed", url)
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
				statReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s?podName=%s&dirPath=%s", basev1, string(common.DirStat), podRequest.PodName, v.path), http.NoBody)
				if err != nil {
					t.Fatal(err)

				}
				statReq.Header.Set("Cookie", cookie[0])
				statResp, err := c.Do(statReq)
				if err != nil {
					t.Fatal(err)
				}
				err = statResp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if statResp.StatusCode != 200 {
					t.Fatal("dir stat failed")
				}
			} else {
				statReq, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s?podName=%s&filePath=%s", basev1, string(common.FileStat), podRequest.PodName, v.path), http.NoBody)
				if err != nil {
					t.Fatal(err)

				}
				statReq.Header.Set("Cookie", cookie[0])
				statResp, err := c.Do(statReq)
				if err != nil {
					t.Fatal(err)
				}
				err = statResp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}
				if statResp.StatusCode != 200 {
					t.Fatal("file stat failed")
				}
			}
		}
	})

	t.Run("ws test", func(t *testing.T) {
		u := url.URL{Scheme: "ws", Host: base, Path: "/ws/v1/"}
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
		if err != nil {
			t.Fatal("dial:", err)
		}
		defer c.Close()

		downloadFn := func(cl string) {
			mt2, reader, err := c.NextReader()
			if mt2 != websocket.BinaryMessage {
				t.Fatal("non binary message while download")
			}
			if err != nil {
				t.Fatal("download failed", err)
			}

			fo, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%d", time.Now().Unix()))
			if err != nil {
				t.Fatal("download failed", err)
			}
			// close fo on exit and check for its returned error
			defer func() {
				if err := fo.Close(); err != nil {
					t.Fatal("download failed", err)
				}
			}()
			n, err := io.Copy(fo, reader)
			if err != nil {
				t.Fatal("download failed", err)
			}
			if fmt.Sprintf("%d", n) == cl {
				return
			}
		}

		go func() {
			for {
				mt, message, err := c.ReadMessage()
				if err != nil {
					return
				}
				if mt == 1 {
					res := &common.WebsocketResponse{}
					if err := json.Unmarshal(message, res); err != nil {
						t.Error("got error ", err)
						continue
					}
					if res.Event == common.FileDownload {
						params := res.Params.(map[string]interface{})
						cl := fmt.Sprintf("%v", params["content_length"])
						downloadFn(cl)
						continue
					}
					if res.StatusCode != 200 && res.StatusCode != 201 {
						t.Errorf("%s failed: %s\n", res.Event, res.Params)
						continue
					}
				}
			}
		}()

		userRequest := &common.UserSignupRequest{
			UserName: randStringRunes(16),
			Password: randStringRunes(8),
		}

		userBytes, err := json.Marshal(userRequest)
		if err != nil {
			t.Fatal(err)
		}

		signupRequestDataHttpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", basev2, string(common.UserSignup)), bytes.NewBuffer(userBytes))
		if err != nil {
			t.Fatal(err)
		}
		signupRequestDataHttpReq.Header.Add("Content-Type", "application/json")
		signupRequestDataHttpReq.Header.Add("Content-Length", strconv.Itoa(len(userBytes)))

		httpClient := http.Client{Timeout: time.Duration(1) * time.Minute}
		signupRequestResp, err := httpClient.Do(signupRequestDataHttpReq)
		if err != nil {
			t.Fatal(err)
		}

		err = signupRequestResp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if signupRequestResp.StatusCode != http.StatusCreated {
			t.Fatal("Signup failed", signupRequestResp.StatusCode)
		}

		// userLogin
		podName := fmt.Sprintf("%d", time.Now().UnixNano())

		login := &common.WebsocketRequest{
			Event:  common.UserLoginV2,
			Params: userRequest,
		}

		data, err := json.Marshal(login)
		if err != nil {
			t.Fatal("failed to marshal login request: ", err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal("write:", err)
		}

		// userPresent
		uPresent := &common.WebsocketRequest{
			Event: common.UserPresentV2,
			Params: common.UserSignupRequest{
				UserName: userRequest.UserName,
			},
		}
		data, err = json.Marshal(uPresent)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// userLoggedIN
		uLoggedIn := &common.WebsocketRequest{
			Event: common.UserIsLoggedin,
			Params: common.UserSignupRequest{
				UserName: userRequest.UserName,
			},
		}
		data, err = json.Marshal(uLoggedIn)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// userStat
		userStat := &common.WebsocketRequest{
			Event: common.UserStat,
		}
		data, err = json.Marshal(userStat)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// podNew
		podNew := &common.WebsocketRequest{
			Event: common.PodNew,
			Params: common.PodRequest{
				PodName:  podName,
				Password: userRequest.Password,
			},
		}
		data, err = json.Marshal(podNew)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// podLs
		podLs := &common.WebsocketRequest{
			Event: common.PodLs,
		}
		data, err = json.Marshal(podLs)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// mkdir
		mkDir := &common.WebsocketRequest{
			Event: common.DirMkdir,
			Params: common.FileRequest{
				PodName: podName,
				DirPath: "/d",
			},
		}
		data, err = json.Marshal(mkDir)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// rmDir
		rmDir := &common.WebsocketRequest{
			Event: common.DirRmdir,
			Params: common.FileRequest{
				PodName: podName,
				DirPath: "/d",
			},
		}
		data, err = json.Marshal(rmDir)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// dirLs
		dirLs := &common.WebsocketRequest{
			Event: common.DirLs,
			Params: common.FileRequest{
				PodName: podName,
				DirPath: "/",
			},
		}
		data, err = json.Marshal(dirLs)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// dirStat
		dirStat := &common.WebsocketRequest{
			Event: common.DirStat,
			Params: common.FileRequest{
				PodName: podName,
				DirPath: "/",
			},
		}
		data, err = json.Marshal(dirStat)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// dirPresent
		dirPresent := &common.WebsocketRequest{
			Event: common.DirIsPresent,
			Params: common.FileRequest{
				PodName: podName,
				DirPath: "/d",
			},
		}
		data, err = json.Marshal(dirPresent)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// Upload
		upload := &common.WebsocketRequest{
			Event: common.FileUpload,
			Params: common.FileRequest{
				PodName:   podName,
				DirPath:   "/",
				BlockSize: "1Mb",
				FileName:  "README.md",
			},
		}
		data, err = json.Marshal(upload)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Open("../../../README.md")
		if err != nil {
			panic(err)
		}
		defer file.Close()
		body := &bytes.Buffer{}
		_, err = io.Copy(body, file)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.BinaryMessage, body.Bytes())
		if err != nil {
			t.Fatal(err)
		}

		// Download
		download := &common.WebsocketRequest{
			Event: common.FileDownload,
			Params: common.FileDownloadRequest{
				PodName:  podName,
				Filepath: "/README.md",
			},
		}
		data, err = json.Marshal(download)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// stat
		stat := &common.WebsocketRequest{
			Event: common.FileStat,
			Params: common.FileSystemRequest{
				PodName:       podName,
				DirectoryPath: "/README.md",
			},
		}
		data, err = json.Marshal(stat)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		table := "kv_1"
		// kvCreate
		kvCreate := &common.WebsocketRequest{
			Event: common.KVCreate,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
				IndexType: "string",
			},
		}
		data, err = json.Marshal(kvCreate)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvList
		kvList := &common.WebsocketRequest{
			Event: common.KVList,
			Params: common.KVRequest{
				PodName: podName,
			},
		}
		data, err = json.Marshal(kvList)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvOpen
		kvOpen := &common.WebsocketRequest{
			Event: common.KVOpen,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
			},
		}
		data, err = json.Marshal(kvOpen)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvEntryPut
		kvEntryPut := &common.WebsocketRequest{
			Event: common.KVEntryPut,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
				Key:       "key1",
				Value:     "value1",
			},
		}
		data, err = json.Marshal(kvEntryPut)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvCount
		kvCount := &common.WebsocketRequest{
			Event: common.KVCount,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
			},
		}
		data, err = json.Marshal(kvCount)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvGet
		kvGet := &common.WebsocketRequest{
			Event: common.KVEntryGet,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
				Key:       "key1",
			},
		}
		data, err = json.Marshal(kvGet)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvSeek
		kvSeek := &common.WebsocketRequest{
			Event: common.KVSeek,
			Params: common.KVRequest{
				PodName:     podName,
				TableName:   table,
				StartPrefix: "key",
			},
		}
		data, err = json.Marshal(kvSeek)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvSeek
		kvSeekNext := &common.WebsocketRequest{
			Event: common.KVSeekNext,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
			},
		}
		data, err = json.Marshal(kvSeekNext)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// kvEntryDel
		kvEntryDel := &common.WebsocketRequest{
			Event: common.KVEntryDelete,
			Params: common.KVRequest{
				PodName:   podName,
				TableName: table,
				Key:       "key1",
			},
		}
		data, err = json.Marshal(kvEntryDel)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		docTable := "doc_1"
		// docCreate
		docCreate := &common.WebsocketRequest{
			Event: common.DocCreate,
			Params: common.DocRequest{
				PodName:     podName,
				TableName:   docTable,
				SimpleIndex: "first_name=string,age=number",
				Mutable:     true,
			},
		}
		data, err = json.Marshal(docCreate)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docLs
		docLs := &common.WebsocketRequest{
			Event: common.DocList,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
			},
		}
		data, err = json.Marshal(docLs)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docOpen
		docOpen := &common.WebsocketRequest{
			Event: common.DocOpen,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
			},
		}
		data, err = json.Marshal(docOpen)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docEntryPut
		docEntryPut := &common.WebsocketRequest{
			Event: common.DocEntryPut,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
				Document:  `{"id":"1", "first_name": "Hello1", "age": 11}`,
			},
		}
		data, err = json.Marshal(docEntryPut)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docEntryGet
		docEntryGet := &common.WebsocketRequest{
			Event: common.DocEntryGet,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
				ID:        "1",
			},
		}
		data, err = json.Marshal(docEntryGet)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docFind
		docFind := &common.WebsocketRequest{
			Event: common.DocFind,
			Params: common.DocRequest{
				PodName:    podName,
				TableName:  docTable,
				Expression: `age>10`,
			},
		}
		data, err = json.Marshal(docFind)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docCount
		docCount := &common.WebsocketRequest{
			Event: common.DocCount,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
			},
		}
		data, err = json.Marshal(docCount)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docEntryGet
		docEntryDel := &common.WebsocketRequest{
			Event: common.DocEntryDel,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
				ID:        "1",
			},
		}
		data, err = json.Marshal(docEntryDel)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// docDel
		docDel := &common.WebsocketRequest{
			Event: common.DocDelete,
			Params: common.DocRequest{
				PodName:   podName,
				TableName: docTable,
			},
		}
		data, err = json.Marshal(docDel)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}
		// user Logout
		uLogout := &common.WebsocketRequest{
			Event: common.UserLogout,
		}
		data, err = json.Marshal(uLogout)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		// userLoggedIN
		uLoggedIn = &common.WebsocketRequest{
			Event: common.UserIsLoggedin,
			Params: common.UserSignupRequest{
				UserName: userRequest.UserName,
			},
		}
		data, err = json.Marshal(uLoggedIn)
		if err != nil {
			t.Fatal(err)
		}
		err = c.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			t.Fatal(err)
		}

		err = c.WriteMessage(websocket.CloseMessage, []byte{})
		if err != nil {
			t.Fatal("write:", err)
		}
		// wait
		<-time.After(time.Second)
	})
}
