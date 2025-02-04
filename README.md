# FairOS-dfs

[![Go](https://github.com/fairDataSociety/fairOS-dfs/workflows/Go/badge.svg)](https://github.com/fairDataSociety/fairOS-dfs/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/fairDataSociety/fairOS-dfs.svg)](https://pkg.go.dev/github.com/fairdatasociety/fairOS-dfs)
[![Go Report Card](https://goreportcard.com/badge/github.com/fairDataSociety/fairOS-dfs)](https://goreportcard.com/report/github.com/fairDataSociety/fairOS-dfs)
[![API OpenAPI Specs](https://img.shields.io/badge/openapi-api-blue)](https://docs.fairos.fairdatasociety.org/api/index.html)
![Docker Pulls](https://img.shields.io/docker/pulls/fairdatasociety/fairos-dfs)
![GitHub all releases](https://img.shields.io/github/downloads/fairdatasociety/fairos-dfs/total)
![GitHub](https://img.shields.io/github/license/fairdatasociety/fairos-dfs)
[![DeepSource](https://deepsource.io/gh/fairDataSociety/fairOS-dfs.svg/?label=active+issues&show_trend=true&token=AVTnJET1ReJkV2-YqwexTmIQ)](https://deepsource.io/gh/fairDataSociety/fairOS-dfs/?ref=repository-badge)

Latest documentation is available at [https://docs.fairos.fairdatasociety.org/docs/](https://docs.fairos.fairdatasociety.org/docs/)

![FairOS-dfs](./docs/images/FairOS-dfs.png)

The Decentralised File System (dfs) is a file system built for the [FairOS](https://github.com/fairDataSociety/fairOS/blob/master/README.md).
It is a stateless thin layer which uses the building blocks provided by Swarm to provide high level functionalities like
- Exposing a logical file system
- Creation of logical drives
- User and Permission management
- Charging and Payments
- Mutable, Indexed data structures over immutable file system

dfs can be used for the following use cases
1) Personal data store
2) Application data store (for both Web 3.0 DApps and web 2.0 Apps)
3) Data sharing with single user and on an organizational level

## User
The first step in dfs is to create a user. Every user is associated with a 12 
word mnemonic based hd wallet. This wallet is password protected and stored in 
the blockchain as ens record. whenever a user created a pod for himself, a new key pair 
is created using this mnemonic.

## What is a pod?
A pod is a personal drive created by a user. It is used to store files and related metadata in a decentralised fashion. A pod is always under the control of the user who created it. A user can create and store any number of files or directories in a pod. 
The user can share files in his pod with any other user just like in other centralised drives like dropbox. Not only users, a pod can be used by decentralised applications (DApp's) to store data related to that user.

Pod creation is cheap. A user can create multiple pods and use it to organise his data. for ex: Personal-Pod, Applications-Pod etc.

## (NEW) Access Control Trie (ACT) Integration
### Overview
We have introduced a new feature that integrates Swarm's Access Control Trie (ACT) into fairOS-dfs to enable user-based access control. This enhancement allows for more granular permissions and secure data sharing among users.

### What is ACT?
The Access Control Trie (ACT) is a mechanism provided by Swarm for managing access permissions to resources stored on the Swarm network. It allows publishers to grant or revoke access to specific grantees.

### How is ACT Integrated into fairOS-dfs?
In the native Swarm implementation, ACT is node-based and lacks the concept of users, which is not suitable for user-centric applications like fairOS-dfs. We have integrated ACT in such a way that:

- User-Based Initialization: Access control is initialized with a user's key, tying permissions directly to user identities.
- Grantee Management: Users can be added as grantees by their public keys, allowing specific users to access shared resources.
- Secure Sharing: Instead of sharing the pod sharing reference directly, we wrap that reference using ACT and share the wrapped actRef. This ensures that only authorized users can access the shared content, even if the actRef is obtained by others.

## (NEW) What is a group? 
A group is a shared drive created by a user. It is basically a pod, but on steroids. Group Owner can add members and update permissions. Members with "write" permission can create and store any number of files or directories in a group.

## How to run FairOS-dfs?
Run the following command to download the latest release

```
curl -o- https://raw.githubusercontent.com/fairDataSociety/fairOS-dfs/master/download.sh | bash
```
```
wget -qO- https://raw.githubusercontent.com/fairDataSociety/fairOS-dfs/master/download.sh | bash
```

Or download the latest release from https://github.com/fairDataSociety/fairOS-dfs/releases.

Or use Docker to run the project https://docs.fairos.fairdatasociety.org/docs/fairOS-dfs/fairos-dfs-using-docker.

Or build the latest version with the instruction https://docs.fairos.fairdatasociety.org/docs/fairOS-dfs/manual-installation.

## Configure FairOS-dfs
To get the most out of your FairOS-dfs it is important that you configure FairOS-dfs for your specific use case!

##### Configuration for Bee
```
bee:
  bee-api-endpoint: http://localhost:1633
  postage-batch-id: ""
```

##### Configuration for FairOS-dfs
```
dfs:
  ports:
    http-port: :9090
    pprof-port: :9091
```

### ENS based Registration

##### RPC endpoint
Fairos depends on blockchain RPC to authenticate user accounts. Hence, it needs `rpc` to connect to
```
rpc: http://localhost:9545
```

##### Custom configuration for ENS based Registration
For ENS based authentication we can either use a `ens-network` configuration in the config file 
```
// define network for ens authtication
ens-network: "testnet"
```

#### Other configuration
```
cookie-domain: api.fairos.io
cors-allowed-origins: []
verbosity: trace
```

This is how a config file should look like
``` .dfs.yaml
bee:
  bee-api-endpoint: http://localhost:1633
  postage-batch-id: <BATCH>
cookie-domain: localhost
cors-allowed-origins: []
dfs:
  ports:
    http-port: :9090
    pprof-port: :9091
rpc: http://localhost:9545
network: "testnet"
verbosity: trace
```

Run `dfs config` to see all configurations

### Help for dfs
```
$ dfs server -h                                                                                                                            

  /$$$$$$          /$$            /$$$$$$   /$$$$$$                /$$  /$$$$$$         
 /$$__  $$        |__/           /$$__  $$ /$$__  $$              | $$ /$$__  $$        
| $$  \__//$$$$$$  /$$  /$$$$$$ | $$  \ $$| $$  \__/          /$$$$$$$| $$  \__//$$$$$$$
| $$$$   |____  $$| $$ /$$__  $$| $$  | $$|  $$$$$$  /$$$$$$ /$$__  $$| $$$$   /$$_____/
| $$_/    /$$$$$$$| $$| $$  \__/| $$  | $$ \____  $$|______/| $$  | $$| $$_/  |  $$$$$$ 
| $$     /$$__  $$| $$| $$      | $$  | $$ /$$  \ $$        | $$  | $$| $$     \____  $$
| $$    |  $$$$$$$| $$| $$      |  $$$$$$/|  $$$$$$/        |  $$$$$$$| $$     /$$$$$$$/
|__/     \_______/|__/|__/       \______/  \______/          \_______/|__/    |_______/


Serves all the dfs commands through an HTTP server so that the upper layers
can consume it.

Usage:
  dfs server [flags]

Flags:
      --cookieDomain string     the domain to use in the cookie (default "api.fairos.io")
      --cors-origins strings    allow CORS headers for the given origins
  -h, --help                    help for server
      --httpPort string         http port (default ":9090")
      --network string          network to use for authentication (mainnet/testnet/play)
      --postageBlockId string   the postage block used to store the data in bee
      --pprofPort string        pprof port (default ":9091")
      --rpc string              rpc endpoint for ens network. xDai for mainnet | Sepolia for testnet | local fdp-play rpc endpoint for play
      --swag                    should run swagger-ui
Global Flags:
      --beeApi string      full bee api endpoint (default "localhost:1633")
      --config string      config file (default "/Users/sabyasachipatra/.dfs.yaml")
      --verbosity string   verbosity level (default "trace")
```

### HTTP APIs

https://docs.fairos.fairdatasociety.org/docs/fairOS-dfs/api-reference

### REPL Commands in dfs-cli

https://docs.fairos.fairdatasociety.org/docs/fairOS-dfs/cli-reference

### Make binaries for all platforms

To make binaries for all platforms run this command

`./generate-exe.sh`

### Generate swagger docs

#### Install swag

```
go install github.com/swaggo/swag/cmd/swag@latest
```

#### Generate

```
make swagger
```

### Running swagger-ui

By default, swagger-ui is disabled. To run swagger-ui we run the `server` command with `--swag` flag

```
$ dfs server --swag
```

This should run the dfs server along with swagger-ui, available at `http://localhost:9090/swagger/index.html` assuming 
server is running on default `9090` port on your localhost

### Running fairOS on sepolia testnet and swarm mainnet

we need to set `network` configuration in the config file as testnet and bee configuration should point to a bee running
on mainnet 

```
network: "testnet"
bee:
  bee-api-endpoint: http://localhost:1633 # bee running on mainnet  
  postage-batch-id: <BATCH>
```

### Integrating git with dfs

To integrate git with dfs, we need to set the `git` configuration in the config file. We only need to set the credential helper for local git repositories.

We create a file `.dfs-credentials` with the following content at any given location
```
#!/bin/bash
token_file="<ABSOLUTE PATH FOR STRING ACCESS TOKEN>" # this needs to be absolute path

username="<USERNAME>"
password="<PASSWORD>"

dfs="<FAIROS-DFS SERVER URL>" # http://localhost:9090 for local running fairOS-dfs server

# Function to get the access token using the username and password
get_access_token() {
    local response=$(curl -s -X POST "$dfs/v2/user/login" -H "Content-Type: application/json" -d "{\"userName\": \"$username\", \"password\": \"$password\"}")
    access_token=$(echo "$response" | jq -r '.accessToken')
    # check if response has access token
    if [ "$access_token" == "null"  ]; then
        exit 1
    fi
    echo "$access_token" > "$token_file"
    echo "$access_token"
}

get_saved_access_token() {
    if [[ -f "$token_file" ]]; then
        local saved_token=$(sed -n '1p' "$token_file")
        if [ "$saved_token" == "null" ] || [ "$saved_token" == "" ]; then
            return 1
        fi
        local response=$(curl --write-out '%{http_code}' --silent --output /dev/null  -s -X POST "$dfs/v1/user/stat" -H "Content-Type: application/json" -H "Authorisation: Bearer $saved_token" )
        # check if response had success http code
        if [[ response -eq 200 ]]; then
            echo "$saved_token"
            return 0
        else
            rm "$token_file"
            return 1
        fi
    fi
    return 1
}

access_token=$(get_saved_access_token)
if [[ $? -ne 0 ]]; then
    access_token=$(get_access_token)
fi
echo "username=$username"
echo "password=$access_token"

exit 0
```

After creating this file, we need to set the `credential.helper` in the git configuration

```
git config --local credential.helper "<Absolute path to .dfs-credentials file>"
```

#### How to push to dfs

Currently, we only support pushing once, so its sort of archive. We can push the git repository to dfs using the following command

```
git init # initialize the git repository, run this inside the directory that you want to push to dfs
git add . # add all the files to the git repository
git commit -m "Initial commit" # commit the changes
git remote add origin <DFS SERVER>/v1/git/<USERNAME>/<PODNAME>.git # add the remote origin
git config --local credential.helper "<Absolute path to .dfs-credentials file>" # set the credential helper
git push -u origin master # push the changes to the remote origin
```

### How to clone from dfs

```
git config --local credential.helper "<Absolute path to .dfs-credentials file>"
git clone <DFS SERVER>/v1/git/<USERNAME>/<PODNAME>.git
```

NOTE: Before pushing into a pod, we have to first create a pod if it does not exist. A pod can not be used as two different repos. 
Dfs stores the files of the repo as a git pack file. So, we cannot access each file of the repo individually. although we can access other files from dfs api as expected.



