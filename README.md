# LeifDB

[![Go Report Card][report-card-badge]][report-card]
[![License][license-badge]][license]
[![Build Status][build-badge]][build]

This is an attempt to create a clustered K-V store application that implements [Raft] for consistency, in Go, based on the [short Raft paper]--something along the lines of [etcd], which backs [Kubernetes]; [Consul], which backs [Vault] and other HashiCorp tools; or [ZooKeeper], which backs most Hadoop-related projects. (etcd and Consul use Raft, ZooKeeper uses a similar algorithm called [Zab], and there are others that use other algorithms such as [Paxos])

Contributions are welcome! Check out the [Contributing Guide] for more info on how to make feature requests, subtmit bug reports, or create pull requests.

## Build and run

This project requires Go 1.14.x, and modifying some elements requires protobuf. If you do not have these installed, see the instructions in the [Contributing Guide].

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

Responses to client endpoints are string-formatted.

To manually run the test suite:

```
go test
```

After building the binary, to find out what command line parameters are available:

```
./app -h
```

To run the server with default parameters, do:

```
./app
```

Or on Windows:

```
./app.exe
```

## Configuration

### HTTP interface

The HTTP interface is used for client interactions with the database. It can be specified using the `-httpport` flag with an integer value. If no value is provided, port 8080 is used.

### gPRC interface

The gRPC interface is used for interactions between members of the Raft cluster. It can be specified using the `-raftport` flag with an integer value. If no value is provided, port 16990 is used.

### Data directory

The persistent data directory is used for storing configuration files and non-volatile server state, and can be specified using the `-data` flag with a path. The path may point to a non-existent location, but cannot exactly match a extant file (an extant directory is fine). If no value is provided, "$HOME/.leifdb/<addr_hash>" is used, where "<addr_hash>" is a non-cryptographic hash of the gRPC interface for the server (such that configration is consistent for a server as long as it is deployed with the same IP address and same port specified by `-raftport`)

### Cluster configuration

In order to interact with other members of a raft cluster, each node must know the addresses for other members. Currently, this is not determined dynamically. In order to create a multi-node deployment, there must be a file in the [persistent data directory](#data-directory) named "config.<ext>", where "<ext>" is one of "json", "yaml", "toml", or other format supported by [spf13/viper]. For instructions on how to write the config file, see the [example config file](./config/default_config.toml). If you want to use a format other than TOML, the [Viper docs on nested keys] demonstrate use with JSON, and other sections include examples of yaml and other formats.

This config in TOML:

```
[configuration]
mode = "multi"
members = ["server1", "server2", "server3"]

[server1]
host = "localhost"
port = 16990

[server2]
host = "localhost"
port = 16991

[server3]
host = "localhost"
port = 16992
```

is equivalent to this in JSON:

```
{
    "configuration": {
        "mode": "multi",
        "members": ["server1", "server2", "server3"],
        "server1": {
            "host": "localhost",
            "port": 16990
        },
        "server2": {
            "host": "localhost",
            "port": 16991
        },
        "server3": {
            "host": "localhost",
            "port": 16992
        }
    }
}
```

or this in YAML:

```
configuration:
    mode: multi
    members:
      - server1
      - server2
      - server3
    server1:
        host: localhost
        port: 16990
    server2:
        host: localhost
        port: 16991
    server3:
        host: localhost
        port: 16992
```

## Endpoints

### Database requests

To create/update a key-value pair (key `somekey`):

```
curl -i -X POST localhost:8080/db/somekey -d '{"value": "test"}'
```

To read the current value for a key `somekey`:

```
curl -i -X GET localhost:8080/db/somekey
```

To remove/delete a key `somekey` from the database:

```
curl -i -X DELETE localhost:8080/db/somekey
```

### Raft requests

Messages used for managing Raft state use protobuf. See test cases for examples of how to construct message bodies. For more info on creating valid values for fields, see the [short Raft paper].

### Other requests

To check the health status of the server:

```
curl -i localhost:8080/health
```

Currently, the return code of this endpoint is the main indicator of health (200 for healthy, anything else indicates not healthy).

## Todo

Raft basics (everything from the [short Raft paper]):
- leader index volatile state
- leader keep track of log index for each other node, send append-logs requests to each based on last known log index, and update commit index for logs appended to a majority of nodes
- add log comparison check to vote handler (election restriction)
- add more checking on most recently seen term
- modify append request job to keep track of logs on other nodes and update commit

Raft complete (additional functionality in the [full Raft paper]):
- log compaction
- changes in cluster membership

General application:
- Add configuration options (probably use "flag")
- Add scripts for starting a cluster / changing membership
- OpenAPI compatibility for HTTP API?
- Dashboard for visualization / management of a cluster?

## Prior art

Aside from the Raft papers themselves, here are some related resources:
- [The Secret Lives of Data]
- [Eli Bendersky's blog post] [which I'm explicitly not reading until after I get an initial version of my own done so that I can muddle along and figure things out the hard way, but leaving this note here for later / for others' benefit]
- A [talk on Raft] from the [Consul] team

[Raft]: https://raft.github.io/
[short Raft paper]: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf
[full Raft paper]: https://raft.github.io/raft.pdf
[The Secret Lives of Data]: http://thesecretlivesofdata.com/raft/
[Eli Bendersky's blog post]: https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/
[talk on Raft]: https://www.hashicorp.com/resources/raft-consul-consensus-protocol-explained/

[etcd]: https://etcd.io
[Kubernetes]: https://kubernetes.io/
[Consul]: https://hashicorp.com/products/consul
[Vault]: https://hashicorp.com/products/vault
[ZooKeeper]: https://zookeeper.apache.org/
[Zab]: https://www.cs.cornell.edu/courses/cs6452/2012sp/papers/zab-ieee.pdf
[Paxos]: http://research.microsoft.com/users/lamport/pubs/paxos-simple.pdf

[gin-gonic/gin]: https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview
[spf13/viper]: https://github.com/spf13/viper
[Viper docs on nested keys]: https://github.com/spf13/viper#accessing-nested-keys

[report-card]: https://goreportcard.com/report/github.com/btmorr/leifdb
[report-card-badge]: https://goreportcard.com/badge/github.com/btmorr/leifdb
[license]: https://github.com/btmorr/leifdb/LICENSE
[license-badge]: https://img.shields.io/github/license/btmorr/leifdb.svg
[build]: https://travis-ci.com/btmorr/leifdb
[build-badge]: https://travis-ci.com/btmorr/leifdb.svg?branch=master

[Contributing Guide]: ./CONTRIBUTING.md
