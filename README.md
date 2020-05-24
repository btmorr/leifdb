# LeifDB

[![Go Report Card][report-card-badge]][report-card]
[![License][license-badge]][license]
[![Build Status][build-badge]][build]

This is an attempt to create a clustered K-V store application that implements [Raft] for consistency, in Go, based on the [short Raft paper]--something along the lines of [etcd], which backs [Kubernetes]; [Consul], which backs [Vault] and other HashiCorp tools; or [ZooKeeper], which backs most [Hadoop]-related projects. (etcd and Consul use Raft, ZooKeeper uses a similar algorithm called [Zab], and there are others that use other algorithms such as [Paxos])

Contributions are welcome! Check out the [Contributing Guide] for more info on how to make feature requests, subtmit bug reports, or create pull requests.

## Build and run

Currently, the database stores a single value. Once that value is correctly updated via Raft, then the database portion of the app can be made more complex for full K-V storage and other functions (which will require modifying the root endpoint).

The simplest way to build and test the application is to enter:

```
make
```

This will clean, build, and test the code. To build and run the app manually on Linux/Unix:

```
go clean
go build -o app
```

On Windows:
```
go clean
go build -o app.exe
```

All requests respond with `application/json`. All error bodies contain the "error" key with a readable message.

To manually run the test suite:

```
go test
```

To run the server, do:

```
make run
```

Or, to run manually on Linux/Unix:

```
./app
```

Or on Windows:

```
./app.exe
```

## Endpoints

### Database requests

To create/update a value:

```
curl -i -X POST localhost:8080/ -d '{"key": "someKey", "value": "test"}'
```

To read the current value:

```
curl -i -X GET localhost:8080/?key=someKey
```

To remove/delete a key from the database:

```
curl -i -X DELETE localhost:8080/?key=someKey
```

### Raft requests

For more info on creating valid values for fields, see the [short Raft paper].

To request a vote (when becoming a candidate / starting an election):

```
curl -i -X POST localhost:8080/vote -d '{"term": 1, "candidateId": "localhost:12345", "lastLogIndex": 0, "lastLogTerm": 0}'
```

To send an append-logs message (when acting as leader):

```
curl -i -X POST localhost:8080/append -d '{"term": 1, "leaderId": "localhost:12345", "prevLogIndex": 0, "prevLogTerm": 0, "entries": [{"term": 1, "value": "test"}], leaderCommit: 0}'
```

### Other requests

To check the health status of the server:

```
curl -i localhost:8080/health
```

Currently, the return code of this endpoint is the main indicator of health (200 for healthy, anything else indicates not healthy).

## Todo

Raft basics (everything from the [short Raft paper]):
- leader index volatile state
- leader keep track of log index for each other node, send append-logs requests to each based on last known log index
- add log comparison check to vote handler (election restriction)
- add more checking on most recently seen term
- Add commit/applied logic
- Separate db into own class, and expand capabilities beyond a single value

Raft complete (additional functionality in the [full Raft paper]):
- log compaction
- modifications

General application:
- Add configuration options (cli? config file?)
- Add scripts for starting a cluster / changing membership
- OpenAPI compatibility for HTTP API?
- Dashboard for visualization / management of a cluster?

## Prior art

Aside from the Raft papers themselves, here are some related resources:
- [The Secret Lives of Data](http://thesecretlivesofdata.com/raft/)
- [Eli Bendersky's blog post](https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/) [which I'm explicitly not reading until after I get an initial version of my own done so that I can muddle along and figure things out the hard way, but leaving this note here for later / for others' benefit]
- A [talk on Raft](https://www.hashicorp.com/resources/raft-consul-consensus-protocol-explained/) from the [Consul] team

[Raft]: https://raft.github.io/
[short Raft paper]: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf
[full Raft paper]: https://raft.github.io/raft.pdf

[etcd]: https://etcd.io
[Kubernetes]: https://kubernetes.io/
[Consul]: https://hashicorp.com/products/consul
[Vault]: https://hashicorp.com/products/vault
[ZooKeeper]: https://zookeeper.apache.org/
[Zab]: https://www.cs.cornell.edu/courses/cs6452/2012sp/papers/zab-ieee.pdf
[Paxos]: http://research.microsoft.com/users/lamport/pubs/paxos-simple.pdf

[gin-gonic/gin]: https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview

[report-card]: https://goreportcard.com/report/github.com/btmorr/leifdb
[report-card-badge]: https://goreportcard.com/badge/github.com/btmorr/leifdb
[license]: https://github.com/btmorr/leifdb/LICENSE
[license-badge]: https://img.shields.io/github/license/btmorr/leifdb.svg
[build]: https://travis-ci.com/btmorr/leifdb
[build-badge]: https://travis-ci.com/btmorr/leifdb.svg?branch=master

[Contributing Guide]: ./CONTRIBUTING.md
