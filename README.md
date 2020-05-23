# Raft practice implementation in Go

[![Go Report Card][report-card-badge]][report-card]
[![License][license-badge]][license]
[![Build Status][build-badge]][build]

This is an attempt to create a clustered K-V store application that implements [Raft] for consistency, in Go, based on the explanation of Raft from [The Secret Lives of Data] and the [short Raft paper]. If/when I get it working for the simplest case (leader accepts GET and POST requests to a specified path to read and write data respectively), then I'll think about other features, such as full OpenAPI support, something other than K-V, a dashboard, etc.

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

To run the test suite:

```
go test
```

## Endpoints

### Database requests

To write a value:

```
curl -i -X POST localhost:8080/ -d '{"value": "test"}'
```

To read the current value:

```
curl -i localhost:8080/
```

### Raft requests

For more info on creating valid values for fields, see the [short Raft paper].

To request a vote (when becoming a candidate / starting an election):

```
curl -i -X POST localhost:8080/vote -d '{"term": 1, "candidateId": "localhost:12345", "lastLogIndex": 0, "lastLogTerm": 0}'
```

To send an append-logs message (when acting as leader):

```
curl -i -X POST localhost:8080/append -d '{"term": 1, "leaderId": "localhost:12345", "lastLogIndex": 0, "lastLogTerm": 0, "entries": [{"term": 1, "value": "test"}], leaderCommit: 0}'
```

### Other requests

To check the health status of the server:

```
curl -i localhost:8080/health
```

## Todo

Raft

- ~~basic vote handler~~
- ~~basic vote client~~
- ~~basic append handler~~
- ~~handle vote response correctly in election~~
- basic append client
- leader index volatile state
- make persistent state persistent
- add log comparison check to vote handler
- add more checking on most recently seen term
- add commit/applied logic
- check logic on receiving append request when leader
- switch data write to append to leader log
- separate db into own class, and expand capabilities beyond a single value


General application

- ~~swap in [gin-gonic/gin] for http router and request/response objects~~
- write unit and/or integration tests (will be easier to do with gin than with using net/http directly)
- separate logic out into smaller functions/modules/packages
- add configuration options (cli? config file?)
- add scripts for starting a cluster / changing membership

[Raft]: https://raft.github.io/
[The Secret Lives of Data]: http://thesecretlivesofdata.com/raft/
[short Raft paper]: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf

[gin-gonic/gin]: https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview

[report-card]: https://goreportcard.com/report/github.com/btmorr/go-raft
[report-card-badge]: https://goreportcard.com/badge/github.com/btmorr/go-raft
[license]: https://github.com/btmorr/go-raft/LICENSE
[license-badge]: https://img.shields.io/github/license/btmorr/go-raft.svg
[build]: https://travis-ci.com/btmorr/go-raft
[build-badge]: https://travis-ci.com/btmorr/go-raft.svg?branch=master
