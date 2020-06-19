# LeifDB

[![Go Report Card][report-card-badge]][report-card]
[![License][license-badge]][license]
[![Build Status][build-badge]][build]

LeifDb a clustered K-V store application that implements [Raft] for consistency, in Go, based on the [short Raft paper]. It has an [OpenAPIv2.0]-compatible HTTP interface for client interaction, and serves the schema for the client interface at the root HTTP endpoint to allow clients to discover and use endpoints programatically.

The aim of this project is to build a distributed, consistent, fault-tolerant database along the lines of [etcd], which backs [Kubernetes]; [Consul], which backs [Vault] and other HashiCorp tools; or [ZooKeeper], which backs most Hadoop-related projects. (etcd and Consul use Raft, ZooKeeper uses a similar algorithm called [Zab], and there are others that use other algorithms such as [Paxos])

Contributions are welcome! Check out the [Contributing Guide] for more info on how to make feature requests, subtmit bug reports, or create pull requests.

## Install

This project requires Go 1.14.x and the `swag` cli tool, and modifying some elements requires protobuf. If you do not have these installed, see the instructions in the [Contributing Guide].

Install the `swag` cli:

```
make install
```

or manually:

```
go get -u github.com/swaggo/swag/cmd/swag
```

## Build and run

The simplest way to build and test the application is to enter:

```
make
```

This will clean, build, and test the code (make tasks may not currently work on Windows without [Windows Subsystem for Linux]). To automatically run (after clean and build), use:

```
make run
```

To provide flags (see [Configuration](#configuration)) or other options:

```
make run run_opts='-raftport 16991'
```

To build and run the app manually on Linux/Unix:

```
go clean
go build -tags=unit,mgmttest -o leifdb
```

On Windows:
```
go clean
go build -tags=unit,mgmttest -o leifdb.exe
```

Responses to client endpoints are string-formatted.

To manually run the test suite:

```
go test -tags=unit ./...
go test -tags=mgmttest ./...
```

After building the binary, to find out what command line parameters are available:

```
./leifdb -h
```

To run the server with default parameters, do:

```
./leifdb
```

Or on Windows:

```
./leifdb.exe
```

## UI

There's a very basic front end! It's capable to connecting to a server, and doing read/write/delete actions. Check out the [readme](./ui/README.md) in that directory for directions on installing and running it.

## Endpoints

### Database requests

After starting the server, you'll be able to interact with the database via the HTTP client interface. The Swagger/OpenAPIv2.0 schema describes the endpoints in the HTTP interface. First, start the server (assuming default HTTP port of 8080 for examples) with `make run` or by invoking the binary directly. Then, to get a copy of the Swagger JSON schema:

```
curl -i localhost:8080/
```

You can also view and interact with endpoints via the auto-generated [Swagger API page](http://localhost:8080/swagger/index.html). This has a section for each endpoint with descriptions, parameters, and an interactive query-runner to make it easy to manually test the server.

CORS is enabled, and you can double-check to make sure that [preflight requests] are handled correctly by doing:

```
curl -X OPTIONS -D - -H 'Origin: http://foo.com' -H 'Access-Control-Request-Method: PUT' localhost:8080/db/testKey?value=something
```

Server should respond with roughly:

```
HTTP/1.1 200 OK
Access-Control-Allow-Methods: PUT
Access-Control-Allow-Origin: *
Vary: Origin
Vary: Access-Control-Request-Method
Vary: Access-Control-Request-Headers
Date: Fri, 05 Jun 2020 20:47:51 GMT
Content-Length: 0

```

### Raft requests

Messages used for managing Raft state use protobuf. See test cases for examples of how to construct message bodies. For more info on creating valid values for fields, see the [short Raft paper].

## Configuration

### HTTP interface

The HTTP interface is used for client interactions with the database. It can be specified using the `-httpport` flag with an integer value. If no value is provided, port 8080 is used.

### gPRC interface

The gRPC interface is used for interactions between members of the Raft cluster. It can be specified using the `-raftport` flag with an integer value. If no value is provided, port 16990 is used.

### Data directory

The persistent data directory is used for storing configuration files and non-volatile server state, and can be specified using the `-data` flag with a path. The path may point to a non-existent location, but cannot exactly match a extant file (an extant directory is fine). If no value is provided, "$HOME/.leifdb/<addr_hash>" is used, where "<addr_hash>" is a non-cryptographic hash of the gRPC interface for the server (such that configration is consistent for a server as long as it is deployed with the same IP address and same port specified by `-raftport`)

### Cluster configuration

In order to interact with other members of a raft cluster, each node must know the addresses for other members. Currently, this is not determined dynamically. In order to create a multi-node deployment, there must be a file in the [persistent data directory](#data-directory) named "config.<ext>", where "<ext>" is one of "json", "yaml", "toml", or other format supported by [spf13/viper]. For instructions on how to write the config file, see the [example config file]. If you want to use a format other than TOML, the [Viper docs on nested keys] demonstrate use with JSON, and other sections include examples of yaml and other formats.

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

To run a cluster on one machine, make 3 directories, and put a copy of the configuration above into each directory (using "/data/a", "/data/b", and "/data/c" for examples below--replace with your chosen directories). Replace "localhost" with your computer's preferred IP (can get it from `ifconfig` on Unix/Linux or `ipconfig` on Windows, or from an error message by running a server with the config file as written--better methods forthcoming). Then open three terminal windows and execute these in each:

```
./leifdb -raftport 16990 -httpport 8080 -data /data/a
```

```
./leifdb -raftport 16991 -httpport 8081 -data /data/b
```

```
./leifdb -raftport 16992 -httpport 8082 -data /data/c
```

(keep track of which window is which, since you'll need to figure out what the ports are for the one that becomes the leader, but you generally can't control which one it will be)

The output will have a *lot* of chatter (unless you change the log level in the `init` function in "main.go"), but should reach a steady state where one of the nodes is the leader. The leader node will be logging a stream of messages like:

```
2020-06-04T07:40:16-04:00 DBG Number needed for append: 2
2020-06-04T07:40:16-04:00 DBG Appended to 3 nodes
2020-06-04T07:40:16-04:00 DBG Need to apply message to 2 nodes
2020-06-04T07:40:16-04:00 DBG Checking for update to commit index commitIndex=1 lastIndex=1
2020-06-04T07:40:16-04:00 DBG Applying records to database lastApplied=1
```

Follower nodes will be streaming messages like:

```
2020-06-04T07:40:16-04:00 DBG apply commits current=1 leader=1
2020-06-04T07:40:16-04:00 DBG Received append request: term:105 leaderId:"192.168.1.21:16991" prevLogIndex:1 prevLogTerm:97 leaderCommit:1
```

Determine which ports correspond to the leader (let's say it's the one with an HTTP service bound to port 8080), then you can issue writes to the leader node, followed by reads to any node. See [Database requests](#database-requests) for writing read/write requests.

## Todo

Raft basics (everything from the [short Raft paper]):
- add log comparison check to vote handler (election restriction)

Raft complete (additional functionality in the [full Raft paper]):
- log compaction
- changes in cluster membership

General application:
- Add scripts for starting a cluster / changing membership (probably something to the tune of [Docker] + [Kubernetes] + [Terraform])
- Performance benchmarking (see the "Measurement" section of [Paxos Made Live] for a couple of ways to set up benchmarks) (also, compare performance with differing levels of debug logging turned on)

## Prior art

Aside from the Raft papers themselves, here are some related resources:
- [The Secret Lives of Data]
- [Eli Bendersky's blog post]
- A [talk on Raft] from the [Consul] team
- [Jay Kreps' article on Logs]
- [Distributed systems for fun and profit]

[Raft]: https://raft.github.io/
[short Raft paper]: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf
[full Raft paper]: https://raft.github.io/raft.pdf
[The Secret Lives of Data]: http://thesecretlivesofdata.com/raft/
[Eli Bendersky's blog post]: https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/
[talk on Raft]: https://www.hashicorp.com/resources/raft-consul-consensus-protocol-explained/
[Jay Kreps' article on Logs]: https://engineering.linkedin.com/distributed-systems/log-what-every-software-engineer-should-know-about-real-time-datas-unifying
[Distributed systems for fun and profit]: http://book.mixu.net/distsys/
[Paxos Made Live]: https://dl.acm.org/doi/10.1145/1281100.1281103
[OpenAPIv2.0]: http://spec.openapis.org/oas/v2.0

[Docker]: https://www.docker.com/
[etcd]: https://etcd.io
[Kubernetes]: https://kubernetes.io/
[Consul]: https://www.consul.io/
[Vault]: https://www.vaultproject.io/
[Terraform]: https://www.terraform.io/
[ZooKeeper]: https://zookeeper.apache.org/
[Zab]: https://www.cs.cornell.edu/courses/cs6452/2012sp/papers/zab-ieee.pdf
[Paxos]: http://research.microsoft.com/users/lamport/pubs/paxos-simple.pdf

[gin-gonic/gin]: https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview
[spf13/viper]: https://github.com/spf13/viper
[Viper docs on nested keys]: https://github.com/spf13/viper#accessing-nested-keys
[preflight requests]: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS

[Windows Subsystem for Linux]: https://docs.microsoft.com/en-us/windows/wsl/about

[report-card]: https://goreportcard.com/report/github.com/btmorr/leifdb
[report-card-badge]: https://goreportcard.com/badge/github.com/btmorr/leifdb
[license]: https://github.com/btmorr/leifdb/LICENSE
[license-badge]: https://img.shields.io/github/license/btmorr/leifdb.svg
[build]: https://travis-ci.com/btmorr/leifdb
[build-badge]: https://travis-ci.com/btmorr/leifdb.svg?branch=master

[Contributing Guide]: ./CONTRIBUTING.md
[example config file]: ./config/default_config.toml
