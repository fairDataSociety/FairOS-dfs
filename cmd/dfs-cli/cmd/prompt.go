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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/fairdatasociety/fairOS-dfs/pkg/utils"
	"github.com/google/shlex"
	"golang.org/x/term"
)

const (
	defaultPrompt   = "dfs"
	userSeperator   = ">>>"
	podSeperator    = ">>"
	promptSeperator = "> "
	apiVersion      = "/v1"
	apiVersionV2    = "/v2"
)

var (
	currentUser      string
	currentPod       string
	currentPrompt    string
	currentDirectory string
	fdfsAPI          *fdfsClient
)

const (
	apiUserIsLoggedin  = apiVersion + "/user/isloggedin"
	apiUserLogout      = apiVersion + "/user/logout"
	apiUserStat        = apiVersion + "/user/stat"
	apiPodNew          = apiVersion + "/pod/new"
	apiPodOpen         = apiVersion + "/pod/open"
	apiPodClose        = apiVersion + "/pod/close"
	apiPodSync         = apiVersion + "/pod/sync"
	apiPodDelete       = apiVersion + "/pod/delete"
	apiPodLs           = apiVersion + "/pod/ls"
	apiPodStat         = apiVersion + "/pod/stat"
	apiPodShare        = apiVersion + "/pod/share"
	apiPodReceive      = apiVersion + "/pod/receive"
	apiPodReceiveInfo  = apiVersion + "/pod/receiveinfo"
	apiDirIsPresent    = apiVersion + "/dir/present"
	apiDirMkdir        = apiVersion + "/dir/mkdir"
	apiDirRmdir        = apiVersion + "/dir/rmdir"
	apiDirLs           = apiVersion + "/dir/ls"
	apiDirStat         = apiVersion + "/dir/stat"
	apiFileDownload    = apiVersion + "/file/download"
	apiFileUpload      = apiVersion + "/file/upload"
	apiFileShare       = apiVersion + "/file/share"
	apiFileReceive     = apiVersion + "/file/receive"
	apiFileReceiveInfo = apiVersion + "/file/receiveinfo"
	apiFileDelete      = apiVersion + "/file/delete"
	apiFileStat        = apiVersion + "/file/stat"
	apiKVCreate        = apiVersion + "/kv/new"
	apiKVList          = apiVersion + "/kv/ls"
	apiKVOpen          = apiVersion + "/kv/open"
	apiKVDelete        = apiVersion + "/kv/delete"
	apiKVCount         = apiVersion + "/kv/count"
	apiKVEntryPut      = apiVersion + "/kv/entry/put"
	apiKVEntryGet      = apiVersion + "/kv/entry/get"
	apiKVEntryDelete   = apiVersion + "/kv/entry/del"
	apiKVLoadCSV       = apiVersion + "/kv/loadcsv"
	apiKVSeek          = apiVersion + "/kv/seek"
	apiKVSeekNext      = apiVersion + "/kv/seek/next"
	apiDocCreate       = apiVersion + "/doc/new"
	apiDocList         = apiVersion + "/doc/ls"
	apiDocOpen         = apiVersion + "/doc/open"
	apiDocCount        = apiVersion + "/doc/count"
	apiDocDelete       = apiVersion + "/doc/delete"
	apiDocFind         = apiVersion + "/doc/find"
	apiDocEntryPut     = apiVersion + "/doc/entry/put"
	apiDocEntryGet     = apiVersion + "/doc/entry/get"
	apiDocEntryDel     = apiVersion + "/doc/entry/del"
	apiDocLoadJson     = apiVersion + "/doc/loadjson"
	apiDocIndexJson    = apiVersion + "/doc/indexjson"

	apiUserSignupV2  = apiVersionV2 + "/user/signup"
	apiUserLoginV2   = apiVersionV2 + "/user/login"
	apiUserPresentV2 = apiVersionV2 + "/user/present"
	apiUserDeleteV2  = apiVersionV2 + "/user/delete"
)

func newPrompt() {
	var err error
	fdfsAPI, err = newFdfsClient(fdfsServer)
	if err != nil {
		fmt.Println("could not create fdfs client")
		os.Exit(1)
	}
	if !fdfsAPI.CheckConnection() {
		fmt.Println("could not connect to fdfs server")
		os.Exit(2)
	}
}

func initPrompt() {
	currentPrompt = defaultPrompt + " " + userSeperator
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix(currentPrompt),
		prompt.OptionLivePrefix(changeLivePrefix),
		prompt.OptionTitle("dfs"),
	)
	p.Run()
}

func changeLivePrefix() (string, bool) {
	return currentPrompt, true
}

var userSuggestions = []prompt.Suggest{
	{Text: "new", Description: "create a new user (v2)"},
	{Text: "del", Description: "delete a existing user (v2)"},
	{Text: "login", Description: "login to a existing user (v2)"},
	{Text: "logout", Description: "logout from a logged-in user"},
	{Text: "present", Description: "is user present (v2)"},
	{Text: "stat", Description: "shows information about a user"},
}

var podSuggestions = []prompt.Suggest{
	{Text: "new", Description: "create a new pod for a user"},
	{Text: "del", Description: "delete a existing pod of a user"},
	{Text: "open", Description: "open to a existing pod of a user"},
	{Text: "close", Description: "close a already opened pod of a user"},
	{Text: "ls", Description: "list all the existing pods of a user"},
	{Text: "stat", Description: "show the metadata of a pod of a user"},
	{Text: "sync", Description: "sync the pod from swarm"},
}

var kvSuggestions = []prompt.Suggest{
	{Text: "new", Description: "create new key value store"},
	{Text: "delete", Description: "delete the  key value store"},
	{Text: "ls", Description: "lists all the key value stores"},
	{Text: "open", Description: "open already created key value store"},
	{Text: "get", Description: "get value from key"},
	{Text: "put", Description: "put key and value in kv store"},
	{Text: "del", Description: "delete key and value from the store"},
	{Text: "loadcsv", Description: "loads the csv file in to kv store"},
	{Text: "seek", Description: "seek to the given start prefix"},
	{Text: "getnext", Description: "get the next element"},
}

var docSuggestions = []prompt.Suggest{
	{Text: "new", Description: "creates a new document store"},
	{Text: "delete", Description: "deletes a document store"},
	{Text: "open", Description: "open the document store"},
	{Text: "ls", Description: "list all document dbs"},
	{Text: "count", Description: "count the docs in the table satisfying the expression"},
	{Text: "find", Description: "find the docs in the table satisfying the expression and limit"},
	{Text: "put", Description: "insert a json document in to document store"},
	{Text: "get", Description: "get the document having the id from the store"},
	{Text: "del", Description: "delete the document having the id from the store"},
	{Text: "loadjson", Description: "load the json file in to the newly created document db"},
}

var suggestions = []prompt.Suggest{
	{Text: "user new", Description: "create a new user"},
	{Text: "user del", Description: "delete a existing user"},
	{Text: "user login", Description: "login to a existing user"},
	{Text: "user logout", Description: "logout from a logged-in user"},
	{Text: "user present", Description: "is user present"},
	{Text: "user export ", Description: "exports the user"},
	{Text: "user import ", Description: "imports the user"},
	{Text: "user stat ", Description: "shows information about a user"},
	{Text: "pod new", Description: "create a new pod for a user"},
	{Text: "pod del", Description: "delete a existing pod of a user"},
	{Text: "pod open", Description: "open to a existing pod of a user"},
	{Text: "pod close", Description: "close a already opened pod of a user"},
	{Text: "pod ls", Description: "list all the existing pods of a user"},
	{Text: "pod stat", Description: "show the metadata of a pod of a user"},
	{Text: "pod sync", Description: "sync the pod from swarm"},
	{Text: "kv new", Description: "create new key value store"},
	{Text: "kv delete", Description: "delete the  key value store"},
	{Text: "kv ls", Description: "lists all the key value stores"},
	{Text: "kv open", Description: "open already created key value store"},
	{Text: "kv get", Description: "get value from key"},
	{Text: "kv put", Description: "put key and value in kv store"},
	{Text: "kv del", Description: "delete key and value from the store"},
	{Text: "kv count", Description: "number of records in the store"},
	{Text: "kv loadcsv", Description: "loads the csv file in to kv store"},
	{Text: "kv seek", Description: "seek to the given start prefix"},
	{Text: "kv getnext", Description: "get the next element"},
	{Text: "doc new", Description: "creates a new document store"},
	{Text: "doc delete", Description: "deletes a document store"},
	{Text: "doc open", Description: "open the document store"},
	{Text: "doc ls", Description: "list all document dbs"},
	{Text: "doc count", Description: "count the docs in the table satisfying the expression"},
	{Text: "doc find", Description: "find the docs in the table satisfying the expression and limit"},
	{Text: "doc put", Description: "insert a json document in to document store"},
	{Text: "doc get", Description: "get the document having the id from the store"},
	{Text: "doc del", Description: "delete the document having the id from the store"},
	{Text: "doc loadjson", Description: "load the json file in to the newly created document db"},
	{Text: "cd", Description: "change path"},
	{Text: "download", Description: "download file from dfs to local machine"},
	{Text: "upload", Description: "upload file from local machine to dfs"},
	{Text: "uploadDir", Description: "upload a dir from local machine to dfs"},
	{Text: "downloadDir", Description: "download dir from dfs to local machine"},
	{Text: "share", Description: "share file with another user"},
	{Text: "receive", Description: "receive a shared file"},
	{Text: "exit", Description: "exit dfs-prompt"},
	{Text: "help", Description: "show usage"},
	{Text: "ls", Description: "list all the file and directories in the current path"},
	{Text: "mkdir", Description: "make a new directory"},
	{Text: "rmdir", Description: "remove a existing directory"},
	{Text: "pwd", Description: "show the current working directory"},
	{Text: "rm", Description: "remove a file"},
}

func completer(in prompt.Document) []prompt.Suggest {
	w := in.Text
	if w == "" || len(strings.Split(w, " ")) >= 3 {
		return []prompt.Suggest{}
	}

	if strings.HasPrefix(in.TextBeforeCursor(), "user") {
		return prompt.FilterHasPrefix(userSuggestions, in.GetWordBeforeCursor(), true)
	} else if strings.HasPrefix(in.TextBeforeCursor(), "pod") {
		return prompt.FilterHasPrefix(podSuggestions, in.GetWordBeforeCursor(), true)
	} else if strings.HasPrefix(in.TextBeforeCursor(), "kv") {
		return prompt.FilterHasPrefix(kvSuggestions, in.GetWordBeforeCursor(), true)
	} else if strings.HasPrefix(in.TextBeforeCursor(), "doc") {
		return prompt.FilterHasPrefix(docSuggestions, in.GetWordBeforeCursor(), true)
	}
	return prompt.FilterHasPrefix(suggestions, w, true)
}

func executor(in string) {
	in = strings.TrimSpace(in)
	blocks, err := shlex.Split(in)
	if err != nil {
		fmt.Println("unable to parse command")
		return
	}
	if len(blocks) == 0 {
		return
	}
	switch blocks[0] {
	case "help":
		help()
	case "exit":
		os.Exit(0)
	case "user":
		if len(blocks) < 2 {
			log.Println("invalid command.")
			help()
			return
		}
		switch blocks[1] {
		case "new":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"userName\" argument")
				return
			}
			userName := blocks[2]
			mnemonic := ""
			if len(blocks) > 4 {
				if len(blocks) < 15 {
					fmt.Println("invalid command. Missing arguments")
					return
				}
				for i := 3; i < 15; i++ {
					mnemonic = mnemonic + " " + blocks[i]
				}
				mnemonic = strings.TrimPrefix(mnemonic, " ")
			}
			userNew(userName, mnemonic)
			currentPod = ""
			currentDirectory = ""
			currentPrompt = getCurrentPrompt()
		case "login":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"userName\" argument")
				return
			}
			userName := blocks[2]
			userLogin(userName, apiUserLoginV2)
			currentPod = ""
			currentDirectory = ""
			currentPrompt = getCurrentPrompt()
		case "present":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"userName\" argument")
				return
			}
			userName := blocks[2]
			presentUser(userName, apiUserPresentV2)
			currentPrompt = getCurrentPrompt()
		case "del":
			if currentUser == "" {
				fmt.Println("please login as  user to do the operation")
				return
			}
			deleteUser(apiUserDeleteV2)
			currentUser = ""
			currentPod = ""
			currentDirectory = ""
			currentPrompt = getCurrentPrompt()
		case "logout":
			if currentUser == "" {
				fmt.Println("please login as  user to do the operation")
				return
			}
			logoutUser()
			currentUser = ""
			currentPod = ""
			currentDirectory = ""
			currentPrompt = getCurrentPrompt()
		case "loggedin":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"userName\" argument")
				return
			}
			userName := blocks[2]
			isUserLoggedIn(userName)
			currentPrompt = getCurrentPrompt()
		case "stat":
			if currentUser == "" {
				fmt.Println("please login as user to do the operation")
				return
			}
			statUser()
			currentPrompt = getCurrentPrompt()
		default:
			fmt.Println("invalid user command")
		}
	case "pod":
		if currentUser == "" {
			fmt.Println("login as a user to execute these commands")
			return
		}
		if len(blocks) < 2 {
			log.Println("invalid command.")
			help()
			return
		}
		switch blocks[1] {
		case "new":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"podName\" argument")
				return
			}
			podName := blocks[2]
			podNew(podName)
			currentPrompt = getCurrentPrompt()
		case "del":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"podName\" argument")
				return
			}
			podName := blocks[2]
			deletePod(podName)
			currentPod = ""
			currentDirectory = ""
			currentPrompt = getCurrentPrompt()
		case "open":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"podName\" argument")
				return
			}
			podName := blocks[2]
			openPod(podName)
			currentPrompt = getCurrentPrompt()
		case "close":
			if !isPodOpened() {
				return
			}
			closePod(currentPod)
			currentPod = ""
			currentDirectory = ""
			currentPrompt = getCurrentPrompt()
		case "stat":
			if !isPodOpened() {
				return
			}
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"podName\" argument")
				return
			}
			podName := blocks[2]
			podStat(podName)
			currentPrompt = getCurrentPrompt()
		case "sync":
			if !isPodOpened() {
				return
			}
			syncPod(currentPod)
			currentPrompt = getCurrentPrompt()
		case "ls":
			listPod()
			currentPrompt = getCurrentPrompt()
		case "share":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"podName\" argument")
				return
			}
			podName := blocks[2]
			sharePod(podName)
			currentPrompt = getCurrentPrompt()
		case "receive":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"reference\" argument")
				return
			}
			podSharingReference := blocks[2]
			receive(podSharingReference)
			currentPrompt = getCurrentPrompt()
		case "receiveinfo":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"reference\" argument")
				return
			}
			podSharingReference := blocks[2]
			receiveInfo(podSharingReference)
			currentPrompt = getCurrentPrompt()

		default:
			fmt.Println("invalid pod command!!")
			help()
		} // end of pod commands
	case "kv":
		if currentUser == "" {
			fmt.Println("login as a user to execute these commands")
			return
		}
		if len(blocks) < 2 {
			log.Println("invalid command.")
			help()
			return
		}
		if !isPodOpened() {
			return
		}
		switch blocks[1] {
		case "new":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			indexType := ""
			if len(blocks) > 3 {
				indexType = blocks[3]
			}
			kvNew(currentPod, tableName, indexType)

			currentPrompt = getCurrentPrompt()
		case "delete":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			kvDelete(currentPod, tableName)
			currentPrompt = getCurrentPrompt()
		case "ls":
			kvList(currentPod)
			currentPrompt = getCurrentPrompt()
		case "open":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			kvOpen(currentPod, tableName)
			currentPrompt = getCurrentPrompt()
		case "count":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			kvCount(currentPod, tableName)
			currentPrompt = getCurrentPrompt()
		case "put":
			if len(blocks) < 5 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			key := blocks[3]
			value := blocks[4]
			kvPut(currentPod, tableName, key, value)
			currentPrompt = getCurrentPrompt()
		case "get":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing \"tableName\" or \"key\" argument")
				return
			}
			tableName := blocks[2]
			key := blocks[3]
			kvget(currentPod, tableName, key)
			currentPrompt = getCurrentPrompt()
		case "del":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing \"tableName\" or \"key\" argument")
				return
			}
			tableName := blocks[2]
			key := blocks[3]
			kvDel(currentPod, tableName, key)
			currentPrompt = getCurrentPrompt()
		case "loadcsv":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more argumentss")
				return
			}
			tableName := blocks[2]
			fileName := filepath.Base(blocks[3])
			localCsvFile := blocks[3]
			memory := ""
			if len(blocks) > 4 {
				memory = blocks[4]
			}
			loadcsv(currentPod, tableName, fileName, localCsvFile, memory)
			currentPrompt = getCurrentPrompt()
		case "seek":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			var start string
			var end string
			var limit string
			if len(blocks) >= 4 {
				start = blocks[3]
			}
			if len(blocks) >= 5 {
				end = blocks[4]
			}

			if len(blocks) >= 6 {
				limit = blocks[5]
			}
			kvSeek(currentPod, tableName, start, end, limit)
			currentPrompt = getCurrentPrompt()
		case "getnext":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			kvGetNext(currentPod, tableName)
			currentPrompt = getCurrentPrompt()
		default:
			fmt.Println("invalid kv command!!")
			help()
		}
	case "doc":
		if currentUser == "" {
			fmt.Println("login as a user to execute these commands")
			return
		}
		if len(blocks) < 2 {
			log.Println("invalid command.")
			help()
			return
		}
		if !isPodOpened() {
			return
		}
		switch blocks[1] {
		case "new":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			si := ""
			mutable := ""
			if len(blocks) >= 4 {
				if blocks[3] == "none" {
					si = ""
				} else {
					si = blocks[3]
				}
			}
			if len(blocks) == 5 {
				mutable = blocks[4]
			}
			docNew(currentPod, tableName, si, mutable)
			currentPrompt = getCurrentPrompt()
		case "ls":
			docList()
			currentPrompt = getCurrentPrompt()
		case "open":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			docOpen(currentPod, tableName)
			currentPrompt = getCurrentPrompt()
		case "count":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			expr := ""
			if len(blocks) == 4 {
				expr = blocks[3]
			}
			docCount(currentPod, tableName, expr)
			currentPrompt = getCurrentPrompt()
		case "delete":
			if len(blocks) < 3 {
				fmt.Println("invalid command. Missing \"tableName\" argument")
				return
			}
			tableName := blocks[2]
			docDelete(currentPod, tableName)
			currentPrompt = getCurrentPrompt()
		case "find":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more arguments")
				return
			}
			tableName := blocks[2]
			expr := blocks[3]
			limit := "10"
			if len(blocks) == 5 {
				limit = blocks[4]
			}
			docFind(currentPod, tableName, expr, limit)
			currentPrompt = getCurrentPrompt()
		case "put":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more arguments")
				return
			}
			tableName := blocks[2]
			value := blocks[3]
			docPut(currentPod, tableName, value)
			currentPrompt = getCurrentPrompt()
		case "get":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more arguments")
				return
			}
			tableName := blocks[2]
			idValue := blocks[3]
			docGet(currentPod, tableName, idValue)
			currentPrompt = getCurrentPrompt()
		case "del":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more arguments")
				return
			}
			tableName := blocks[2]
			idValue := blocks[3]
			docDel(currentPod, tableName, idValue)
			currentPrompt = getCurrentPrompt()
		case "loadjson":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more arguments")
				return
			}
			tableName := blocks[2]
			fileName := filepath.Base(blocks[3])
			localJsonFile := blocks[3]
			docLoadJson(currentPod, localJsonFile, tableName, fileName)
			currentPrompt = getCurrentPrompt()
		case "indexjson":
			if len(blocks) < 4 {
				fmt.Println("invalid command. Missing one or more arguments")
				return
			}
			tableName := blocks[2]
			podJsonFile := blocks[3]
			docIndexJson(currentPod, tableName, podJsonFile)
			currentPrompt = getCurrentPrompt()
		default:
			fmt.Println("Invalid doc coammand")
			currentPrompt = getCurrentPrompt()
		}
	case "cd":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		dirTocd := blocks[1]

		// if cd'ing to previous dir, just do it
		if dirTocd == ".." && currentDirectory != utils.PathSeparator {
			currentDirectory = filepath.Dir(currentDirectory)
			currentPrompt = getCurrentPrompt()
			return
		}

		// if cd'ing to root dir, just do it
		if dirTocd == utils.PathSeparator {
			currentDirectory = utils.PathSeparator
			currentPrompt = getCurrentPrompt()
			return
		}

		// if cd'ing forward, we have to check if that dir is present
		if dirTocd != utils.PathSeparator {
			if currentDirectory == utils.PathSeparator {
				dirTocd = currentDirectory + dirTocd
			} else {
				dirTocd = currentDirectory + utils.PathSeparator + dirTocd
			}
		}

		present := isDirectoryPresent(currentPod, dirTocd)
		if present {
			currentDirectory = dirTocd
		}
		currentPrompt = getCurrentPrompt()
	case "ls":
		if !isPodOpened() {
			return
		}
		_, err = listFileAndDirectories(currentPod, currentDirectory)
		if err != nil {
			fmt.Println("ls failed: ", err)
		}
		currentPrompt = getCurrentPrompt()
	case "mkdir":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		dirToMk := blocks[1]
		if dirToMk == "" {
			fmt.Println("invalid dir")
			return
		}
		if !strings.HasPrefix(dirToMk, utils.PathSeparator) {
			// then this path is not from root
			dirToMk = utils.PathSeparator + dirToMk
			if currentDirectory != utils.PathSeparator {
				dirToMk = currentDirectory + utils.PathSeparator + dirToMk
			}
		}
		mkdir(currentPod, dirToMk)
		currentPrompt = getCurrentPrompt()
	case "rmdir":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		dirToRm := blocks[1]
		if dirToRm == "" {
			fmt.Println("invalid dir")
			return
		}
		if !strings.HasPrefix(dirToRm, utils.PathSeparator) {
			// then this path is not from root
			if currentDirectory == utils.PathSeparator {
				dirToRm = currentDirectory + dirToRm
			} else {
				dirToRm = currentDirectory + utils.PathSeparator + dirToRm
			}
		}
		rmDir(currentPod, dirToRm)
		currentPrompt = getCurrentPrompt()
	case "uploadDir":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 4 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		dirName := blocks[1]
		podDir := blocks[2]
		if podDir == "." {
			podDir = currentDirectory
		}
		blockSize := blocks[3]
		compression := ""
		if len(blocks) >= 5 {
			compression = blocks[4]
			if compression != "snappy" && compression != "gzip" {
				fmt.Println("invalid value for \"compression\", should either be \"snappy\" or \"gzip\"")
				return
			}
		}
		toUpload, err := findFilesToUpload(dirName)
		if err != nil {
			fmt.Println("Failed to list files to upload at: ", dirName, err)
			return
		}
		for _, item := range toUpload.filesToUpload {
			if toUpload.rootDirectory != item {
				itemStats, err := os.Stat(item)
				if err != nil {
					fmt.Println("Failed to read stats for: ", item, err)
					continue
				}
				fmt.Println("Handling item: ", removeParentDirectory(toUpload.rootDirectory, item))
				if itemStats.IsDir() {
					dirToMk := filepath.ToSlash(filepath.Join(podDir, removeParentDirectory(toUpload.rootDirectory, item)))
					mkdir(currentPod, dirToMk)
				} else {
					filePath := removeParentDirectory(toUpload.rootDirectory, item)
					uploadFile(filepath.Base(filePath), currentPod, item, filepath.ToSlash(filepath.Join(podDir, filepath.Dir(filePath))), blockSize, compression)
				}
			}
		}
		currentPrompt = getCurrentPrompt()
	case "downloadDir":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 3 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		localDir := blocks[1]
		dirStat, err := os.Stat(localDir)
		if err != nil {
			fmt.Println("local path is not a present: ", err)
			return
		}

		if !dirStat.IsDir() {
			fmt.Println("local path is not a directory")
			return
		}

		podPath := blocks[2]
		if !strings.HasPrefix(podPath, utils.PathSeparator) {
			if currentDirectory == utils.PathSeparator {
				podPath = currentDirectory + podPath
			} else {
				podPath = currentDirectory + utils.PathSeparator + podPath
			}
		}
		downloadDir(currentPod, localDir, podPath)
		currentPrompt = getCurrentPrompt()
	case "upload":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 4 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		fileName := filepath.Base(blocks[1])
		podDir := blocks[2]
		if podDir == "." {
			podDir = currentDirectory
		}
		blockSize := blocks[3]
		compression := ""
		if len(blocks) >= 5 {
			compression = blocks[4]
			if compression != "snappy" && compression != "gzip" {
				fmt.Println("invalid value for \"compression\", should either be \"snappy\" or \"gzip\"")
				return
			}
		}
		uploadFile(fileName, currentPod, blocks[1], podDir, blockSize, compression)
		currentPrompt = getCurrentPrompt()
	case "download":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 3 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		localDir := blocks[1]
		dirStat, err := os.Stat(localDir)
		if err != nil {
			fmt.Println("local path is not a present: ", err)
			return
		}

		if !dirStat.IsDir() {
			fmt.Println("local path is not a directory")
			return
		}

		loalFile := filepath.Join(localDir + utils.PathSeparator + filepath.Base(blocks[2]))
		podFile := blocks[2]
		if !strings.HasPrefix(podFile, utils.PathSeparator) {
			if currentDirectory == utils.PathSeparator {
				podFile = currentDirectory + podFile
			} else {
				podFile = currentDirectory + utils.PathSeparator + podFile
			}
		}

		downloadFile(currentPod, loalFile, podFile)
		currentPrompt = getCurrentPrompt()
	case "stat":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		statElement := blocks[1]
		if statElement == "" {
			return
		}
		if !strings.HasPrefix(statElement, utils.PathSeparator) {
			if currentDirectory == utils.PathSeparator {
				statElement = currentDirectory + statElement
			} else {
				statElement = currentDirectory + utils.PathSeparator + statElement
			}
		}
		statFileOrDirectory(currentPod, statElement)
		currentPrompt = getCurrentPrompt()
	case "pwd":
		if !isPodOpened() {
			return
		}
		fmt.Println(currentDirectory)
		currentPrompt = getCurrentPrompt()
	case "rm":
		if !isPodOpened() {
			return
		}
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		rmFile := blocks[1]
		if rmFile == "" {
			return
		}
		if !strings.HasPrefix(rmFile, utils.PathSeparator) {
			if currentDirectory == utils.PathSeparator {
				rmFile = currentDirectory + rmFile
			} else {
				rmFile = currentDirectory + utils.PathSeparator + rmFile
			}
		}
		deleteFile(currentPod, rmFile)
		currentPrompt = getCurrentPrompt()
	case "share":
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		podFile := blocks[1]

		if podFile == "" {
			return
		}
		if !strings.HasPrefix(podFile, utils.PathSeparator) {
			if currentDirectory == utils.PathSeparator {
				podFile = currentDirectory + podFile
			} else {
				podFile = currentDirectory + utils.PathSeparator + podFile
			}
		}
		fileShare(currentPod, podFile, "TODO: add dest. user address")
		currentPrompt = getCurrentPrompt()
	case "receive":
		if len(blocks) < 3 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		sharingRefString := blocks[1]
		podDir := blocks[2]
		fileReceive(currentPod, sharingRefString, podDir)
		currentPrompt = getCurrentPrompt()
	case "receiveinfo":
		if len(blocks) < 2 {
			fmt.Println("invalid command. Missing one or more arguments")
			return
		}
		sharingRefString := blocks[1]
		fileReceiveInfo(currentPod, sharingRefString)
		currentPrompt = getCurrentPrompt()
	default:
		fmt.Println("invalid command")
	}
}

func help() {
	fmt.Println("Usage: <command> <sub-command> (args1) (args2) ...")
	fmt.Println("commands:")
	fmt.Println(" - user <new> (user-name) (mnemonic) - create a new user and login as that user")
	fmt.Println(" - user <del> - deletes a logged-in user")
	fmt.Println(" - user <login> (user-name) - login as a given user")
	fmt.Println(" - user <logout> - logout a logged-in user")
	fmt.Println(" - user <present> (user-name) - returns true if the user is present, false otherwise")
	fmt.Println(" - user <stat> - shows information about a user")

	fmt.Println(" - pod <new> (pod-name) - create a new pod for the logged-in user and opens the pod")
	fmt.Println(" - pod <del> (pod-name) - deletes a already created pod of the user")
	fmt.Println(" - pod <open> (pod-name) - open a already created pod")
	fmt.Println(" - pod <stat> (pod-name) - display meta information about a pod")
	fmt.Println(" - pod <sync> (pod-name) - sync the contents of a logged-in pod from Swarm")
	fmt.Println(" - pod <close>  - close a opened pod")
	fmt.Println(" - pod <ls> - lists all the pods created for this account")
	fmt.Println(" - pod <share> (pod-name) - share a pod and get sharing reference")
	fmt.Println(" - pod <receive> (sharing-reference) - add a pod with the given sharing reference")
	fmt.Println(" - pod <receiveinfo> (sharing-reference) - check a pod metadata with the given sharing reference")

	fmt.Println(" - kv <new> (table-name) - creates a new key value store")
	fmt.Println(" - kv <delete> (table-name) - deletes the key value store")
	fmt.Println(" - kv <open> (table-name) - open the key value store")
	fmt.Println(" - kv <ls>  - list all collections")
	fmt.Println(" - kv <put> (table-name) (key) (value) - insert key and value in to kv store")
	fmt.Println(" - kv <get> (table-name) (key) - get the value of the given key from the store")
	fmt.Println(" - kv <del> (table-name) (key) - remove the key and value from the store")
	fmt.Println(" - kv <loadcsv> (table-name) (local csv file) - load the csv file in to a newly created table")
	fmt.Println(" - kv <seek> (table-name) (start-key) (end-key) (limit) - seek nearest to start key")
	fmt.Println(" - kv <getnext> (table-name) - get the next element after seek")
	fmt.Println(" - kv <count> (table-name) - number of records in the store")

	fmt.Println(" - doc <new> (table-name) (si=indexes) - creates a new document store")
	fmt.Println(" - doc <delete> (table-name) - deletes a document store")
	fmt.Println(" - doc <open> (table-name) - open the document store")
	fmt.Println(" - doc <ls>  - list all document dbs")
	fmt.Println(" - doc <count> (table-name) (expr) - count the docs in the table satisfying the expression")
	fmt.Println(" - doc <find> (table-name) (expr) (limit)- find the docs in the table satisfying the expression and limit")
	fmt.Println(" - doc <put> (table-name) (json) - insert a json document in to document store")
	fmt.Println(" - doc <get> (table-name) (id) - get the document having the id from the store")
	fmt.Println(" - doc <del> (table-name) (id) - delete the document having the id from the store")
	fmt.Println(" - doc <loadjson> (table-name) (local json file) - load the json file in to the newly created document db")
	fmt.Println(" - doc <indexjson> (table-name) (pod json file) - Index the json file in pod to the document db")

	fmt.Println(" - cd <directory name>")
	fmt.Println(" - ls ")
	fmt.Println(" - download <destination dir in local fs> <relative path of source file in pod>")
	fmt.Println(" - upload <source file in local fs> <destination directory in pod> <block size (ex: 1Mb, 64Mb)>, <compression (snappy/gzip)>")
	fmt.Println(" - uploadDir <source location in local fs> <destination directory in pod> <block size (ex: 1Mb, 64Mb)>, <compression (snappy/gzip)>")
	fmt.Println(" - downloadDir <destination location in local fs> <source directory in pod>")
	fmt.Println(" - share <file name> -  shares a file with another user")
	fmt.Println(" - receive <sharing reference> <pod dir> - receives a file from another user")
	fmt.Println(" - receiveinfo <sharing reference> - shows the received file info before accepting the receive")
	fmt.Println(" - mkdir <directory name>")
	fmt.Println(" - rmdir <directory name>")
	fmt.Println(" - rm <file name>")
	fmt.Println(" - pwd - show present working directory")
	fmt.Println(" - stat <file name or directory name> - shows the information about a file or directory")
	fmt.Println(" - help - display this help")
	fmt.Println(" - exit - exits from the prompt")

}

func getCurrentPrompt() string {
	currPrompt := getUserPrompt()
	podPrompt := getPodPrompt()
	if podPrompt != "" {
		currPrompt = currPrompt + " " + podPrompt + " " + podSeperator
	}
	dirPrompt := currentDirectory
	if dirPrompt != "" {
		currPrompt = currPrompt + " " + dirPrompt + " " + promptSeperator
	}
	return currPrompt
}

func isPodOpened() bool {
	if currentPod == "" {
		fmt.Println("open the pod to do the operation")
		return false
	}
	return true
}

func getUserPrompt() string {
	if currentUser == "" {
		return defaultPrompt + " " + userSeperator
	} else {
		return defaultPrompt + "@" + currentUser + " " + userSeperator
	}
}

func getPodPrompt() string {
	if currentPod != "" {
		return currentPod
	} else {
		return ""
	}
}

func getPassword() (password string) {
	fmt.Print("Please enter your password: ")
	bytePassword, err := term.ReadPassword(0)
	if err != nil {
		log.Fatalf("error reading password")
		return
	}
	fmt.Println("")
	passwd := string(bytePassword)
	password = strings.TrimSpace(passwd)
	return password
}

type files struct {
	filesToUpload []string
	rootDirectory string
}

func findFilesToUpload(searchPath string) (*files, error) {
	var searchResults []string
	rawSearchResults, err := filepath.Glob(searchPath)
	if err != nil {
		return nil, err
	}

	for _, searchResult := range rawSearchResults {
		searchResults = append(searchResults, searchResult)
		fileStats, err := os.Stat(searchResult)
		if err != nil {
			return nil, err
		}

		if fileStats.IsDir() {
			toUpload, err := findFilesToUpload(filepath.Join(searchResult, "*"))
			if err == nil {
				searchResults = append(searchResults, toUpload.filesToUpload...)
			}
		}
	}

	if len(searchResults) == 1 && searchPath == searchResults[0] {
		return &files{
			filesToUpload: searchResults,
			rootDirectory: filepath.Dir(searchResults[0]),
		}, nil
	}

	return &files{
		filesToUpload: searchResults,
		rootDirectory: searchPath,
	}, nil
}

func removeParentDirectory(parentPath, childPath string) string {
	relativePath, err := filepath.Rel(parentPath, childPath)
	if err != nil {
		panic(err)
	}
	if relativePath == "" {
		return "/"
	}
	if strings.HasPrefix(relativePath, "..") {
		return ""
	}
	return "/" + relativePath
}

func downloadDir(currentPod, localDir, podPath string) {
	stat, err := listFileAndDirectories(currentPod, podPath)
	if err != nil {
		fmt.Println("failed to get contents of dir: ", podPath, err)
		return
	}
	for _, dir := range stat.Directories {
		err := os.Mkdir(filepath.Join(localDir, dir.Name), os.FileMode(dir.Mode))
		if err != nil && !errors.Is(err, os.ErrExist) {
			fmt.Println("failed to create local dir: ", filepath.Join(localDir, dir.Name), err)
			continue
		}
		downloadDir(currentPod, filepath.Join(localDir, dir.Name), filepath.ToSlash(filepath.Join(podPath, dir.Name)))
	}
	for _, file := range stat.Files {
		loalFile := filepath.Join(localDir, file.Name)
		downloadFile(currentPod, loalFile, filepath.ToSlash(filepath.Join(podPath, file.Name)))
	}
}
