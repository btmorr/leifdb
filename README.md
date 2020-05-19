# Raft practice implementation in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/btmorr/go-raft)](https://goreportcard.com/report/github.com/btmorr/go-raft)[![License](https://img.shields.io/github/license/btmorr/go-raft.svg)](https://github.com/btmorr/go-raft/LICENSE)

This is an attempt to create a clustered K-V store application that implements [Raft](https://raft.github.io/) for consistency, in Go, based on the explanation of Raft from [The Secret Lives of Data](http://thesecretlivesofdata.com/raft/) and the [short paper](https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf). If/when I get it working for the simplest case (leader accepts GET and POST requests to a specified path to read and write data respectively), then I'll think about other features, such as full OpenAPI support, something other than K-V, a dashboard, etc.

## Build and run

Currently, the server is a single node that stores a single value.

To build and run the server on Linux/Unix:

```
go build
./go-raft
```

On Windows:
```
go build
./go-raft.exe
```

All requests respond with `application/json`. All error bodies contain the "error" key with a readable message.

## Database requests

To write a value:

```
curl -X POST localhost:8080/ -d '{"value": "test"}'
```

To read the current value:

```
curl localhost:8080/
```

## Todo

Raft

- ~~basic vote handler~~
- ~~basic vote client~~
- ~~basic append handler~~
- basic append client
- leader index volatile state
- make persistent state persistent
- add log comparison check to vote handler
- add more checking on most recently seen term
- add commit/applied logic
- check logic on receiving append request when leader
- switch data write to append to leader log


General application

- swap in [gin-gonic/gin](https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview) for http router and request/response objects
- write unit and/or integration tests (will be easier to do with gin than with using net/http directly)
- separate logic out into smaller functions/modules/packages
- add configuration options (cli? config file?)
- add scripts for starting a cluster / changing membership
