# LeifDB

[![Go Report Card][report-card-badge]][report-card]
[![License][license-badge]][license]
[![CircleCI][build-badge]][build]

LeifDb a clustered K-V store application that implements [Raft] for consistency, in Go. It has an [OpenAPIv2.0]-compatible HTTP interface for client interaction, and serves the schema for the client interface at the root HTTP endpoint to allow clients to discover and use endpoints programmatically. In the near future, it will also employ erasure codes to improve performance and storage footprint, as described in the [CRaft] paper (check out the [milestones](https://github.com/btmorr/leifdb/milestones) to check on progress).

The aim of this project is to build a distributed, consistent, fault-tolerant K-V store providing high throughput and minimizing storage footprint at large scale.

Contributions are welcome! Check out the [Contributing Guide] for more info on how to make feature requests, submit bug reports, or create pull requests.

## Install

This project requires Go 1.14.x+ and the [swaggo/swag] cli tool, and modifying some elements requires protobuf. If you do not have these installed, see the instructions in the [Contributing Guide].

## Build and run

The simplest way to build and test the application is to enter:

```
make
```

This will clean, build, and test the code (make tasks may not currently work on Windows without [Windows Subsystem for Linux] or Git Bash terminal).

To run the database run this command, replacing "\<version>" with the current version number:

```
./leifdb-<version>
```

Use environment variables to configure options (for all options, see [Configuration](#configuration)):

```
env LEIFDB_HTTP_PORT=8081 ./leifdb-<version>
```

To build and run the app manually on Linux/Unix:

```
go clean
go build -tags=unit,mgmttest -o leifdb
env LEIFDB_HTTP_PORT=8081 ./leifdb
```

On Windows PowerShell:

```
go clean
go build -tags=unit,mgmttest -o leifdb.exe
$env:LEIFDB_HTTP_PORT="8081" ./leifdb.exe
```

For other options, such as manually running the test suite, take a look at the commands in the [Makefile](./Makefile).

## UI

There's a very basic front end! It's capable to connecting to a server, and doing read/write/delete actions. Check out the [readme](./ui) in that directory for directions on installing and running it.

## Starting a demo cluster

This repo includes a docker-compose specification for starting a demo cluster with 3 nodes and the UI application. To build and start the demo cluster, do:

```
make testcluster
```

This is a great way to see the cluster in action, try out the UI, or as an example for configuration. Once the applications start, you can use a web browser to open [the UI interface](http://localhost:3000), and use "localhost:8080", "localhost:8081", or "localhost:8082" to connect to any of the server nodes and search/set/delete.

## Endpoints

### Database requests

After starting the server, you'll be able to interact with the database via the HTTP client interface. The Swagger/OpenAPIv2.0 schema describes the endpoints in the HTTP interface. First, start the server (assuming default HTTP port of 8080 for examples) with `make run` or by invoking the binary directly. Then, to get a copy of the Swagger JSON schema:

```
curl -i localhost:8080/
```

You can also view and interact with endpoints via the auto-generated [Swagger API page](http://localhost:8080/swagger/index.html). This has a section for each endpoint with descriptions, parameters, and an interactive query-runner to make it easy to manually test the server.

Adding or changing endpoints will probably require regenerating the Swagger schema file and related code. Make sure that the endpoint declarative comments are updated (see the [swaggo/swag] documentation for reference info on what options are available and their parameters). Running `make` after updating the comments and code will generate the new "swagger.[json|yaml]" files, or you can run `swag init` manually.

Once any changes are solidified, you will probably also need to update the [UI subproject](./ui/README.md#updates-to-the-server-api), since it uses code autogenerated from the Swagger schema.

### CORS

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

The HTTP interface is used for client interactions with the database. It can be specified using the `LEIFDB_HTTP_PORT` environment variable with an integer value. If no value is provided, port 8080 is used.

### gPRC interface

The gRPC interface is used for interactions between members of the Raft cluster. It can be specified using the `LEIFDB_RAFT_PORT` environment variable with an integer value. If no value is provided, port 16990 is used.

### Data directory

The persistent data directory is used for storing configuration files and non-volatile server state, and can be specified using the `LEIFDB_DATA_DIR` environment variable with a path. The path may point to a non-existent location, but cannot exactly match an existing file (an existing directory is fine). If no value is provided, "\$HOME/.leifdb/<addr_hash>" is used, where "<addr_hash>" is a non-cryptographic hash of the gRPC interface for the server (such that configuration is consistent for a server as long as it is deployed with the same hostname or IP address and same port specified by `LEIFDB_DATA_DIR`)

### Snapshot threshold

_[feature in progress]_

When the log of database transactions reaches a certain size, the server will compact the logs by taking a snapshot of the database state and dropping log entries leading up to that point. Two environment variables govern this behavior: `LEIFDB_SNAPSHOT_THRESHOLD` is an integer number in bytes for how large the log file is allowed to grow before a snapshot is taken (default of 1073741824, which is equal to 1Gb), and `LEIFDB_RETAIN_N_SNAPSHOTS` is an integer for the number of snapshots to keep at a time (default of 1 and also minimum of 1). When a new snapshot is successfully created the snapshots will be counted and if there are more than the number specified then the oldest will be discarded.

### Cluster configuration

In order to interact with other members of a raft cluster, each node must know the addresses for other members. Currently, this is not determined dynamically. In order to create a multi-node deployment, there must be two environment variables set:

- `LEIFDB_MODE`: must be "multi" (default is "single")
- `LEIFDB_MEMBER_NODES`: must be a comma-separated list of addresses for other nodes, such as "10.10.0.2:16990,10.10.0.3:16990,10.10.0.4:16990"

To run a cluster on one machine, make 3 directories named "$HOME/testdata/a", "$HOME/testdata/b", and "\$HOME/testdata/c". Replace "10.10.0.x" with either "localhost" or your computer's preferred IP (can get it from `ifconfig` on Unix/Linux or `ipconfig` on Windows, or from an error message by running a server with the config file as written--better methods forthcoming). Then open three terminal windows and execute these in each:

```
env LEIFDB_DATA_DIR=~/testdata/a \
  LEIFDB_MODE=multi \
  LEIFDB_MEMBER_NODES="localhost:16990,localhost:16991,localhost:16992" \
  LEIFDB_HOST=localhost \
  LEIFDB_HTTP_PORT=8080 \
  LEIFDB_RAFT_PORT=16990 \
  ./leifdb
```

```
env LEIFDB_DATA_DIR=~/testdata/b \
  LEIFDB_MODE=multi \
  LEIFDB_MEMBER_NODES="localhost:16990,localhost:16991,localhost:16992" \
  LEIFDB_HOST=localhost \
  LEIFDB_HTTP_PORT=8081 \
  LEIFDB_RAFT_PORT=16991 \
  ./leifdb
```

```
env LEIFDB_DATA_DIR=~/testdata/c \
  LEIFDB_MODE=multi \
  LEIFDB_MEMBER_NODES="localhost:16990,localhost:16991,localhost:16992" \
  LEIFDB_HOST=localhost \
  LEIFDB_HTTP_PORT=8082 \
  LEIFDB_RAFT_PORT=16992 \
  ./leifdb
```

The output will have quite a bit of chatter (unless you raise the log level in the `init` function in "main.go"), but should reach a steady state where one of the nodes is the leader. The leader node will be logging a stream of messages like:

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

You can now issue read and write requests to any node (writes will be redirected to the leader--remember to use the `-L` flag if you are using curl). See [Database requests](#database-requests) for writing read/write requests.

### Log Level Configuration

The log level can be set using the environment variable `LEIFDB_LOG_LEVEL`. The value can be either one of `panic`, `fatal`, `error`, `warn`, `info`, `debug` or `trace`. By default, the log level is set to be `info`.

## Prior art

Aside from the Raft papers themselves ([short] and [extended]) and the [CRaft] paper, here are some related resources:

- [The Secret Lives of Data]
- [Eli Bendersky's blog post]
- A [talk on Raft] from the [Consul] team
- [Jay Kreps' article on Logs]
- [Distributed systems for fun and profit]
- [Paxos made simple] for comparison with other strategies
- The "Measurement" section of [Paxos Made Live] has a good discussion of performance benchmarking

[raft]: https://raft.github.io/
[short]: https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf
[extended]: https://raft.github.io/raft.pdf
[craft]: https://www.usenix.org/system/files/fast20-wang_zizhong.pdf
[the secret lives of data]: http://thesecretlivesofdata.com/raft/
[eli bendersky's blog post]: https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/
[talk on raft]: https://www.hashicorp.com/resources/raft-consul-consensus-protocol-explained/
[jay kreps' article on logs]: https://engineering.linkedin.com/distributed-systems/log-what-every-software-engineer-should-know-about-real-time-datas-unifying
[distributed systems for fun and profit]: http://book.mixu.net/distsys/
[paxos made live]: https://dl.acm.org/doi/10.1145/1281100.1281103
[openapiv2.0]: http://spec.openapis.org/oas/v2.0
[docker]: https://www.docker.com/
[etcd]: https://etcd.io
[kubernetes]: https://kubernetes.io/
[consul]: https://www.consul.io/
[vault]: https://www.vaultproject.io/
[zookeeper]: https://zookeeper.apache.org/
[zab]: https://www.cs.cornell.edu/courses/cs6452/2012sp/papers/zab-ieee.pdf
[paxos made simple]: http://research.microsoft.com/users/lamport/pubs/paxos-simple.pdf
[swaggo/swag]: https://github.com/swaggo/swag/
[gin-gonic/gin]: https://pkg.go.dev/github.com/gin-gonic/gin?tab=overview
[preflight requests]: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
[windows subsystem for linux]: https://docs.microsoft.com/en-us/windows/wsl/about
[report-card]: https://goreportcard.com/report/github.com/btmorr/leifdb
[report-card-badge]: https://goreportcard.com/badge/github.com/btmorr/leifdb
[license]: https://github.com/btmorr/leifdb/LICENSE
[license-badge]: https://img.shields.io/github/license/btmorr/leifdb.svg
[build]: https://dl.circleci.com/status-badge/redirect/gh/btmorr/leifdb/tree/edge
[build-badge]: https://dl.circleci.com/status-badge/img/gh/btmorr/leifdb/tree/edge.svg?style=svg
[contributing guide]: ./CONTRIBUTING.md
