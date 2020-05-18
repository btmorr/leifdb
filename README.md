# Raft practice implementation in Go

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
