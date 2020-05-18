# Raft practice implementation in Go

This is an attempt to create a clustered K-V store application that implements [Raft](https://raft.github.io/) for consistency, in Go, based on the explanation of Raft from [The Secret Lives of Data](http://thesecretlivesofdata.com/raft/) and the [short paper](https://www.usenix.org/system/files/conference/atc14/atc14-paper-ongaro.pdf). If/when I get it working for the simplest case (leader accepts GET and POST requests to a specified path to read and write data respectively), then I'll think about other features, such as full OpenAPI support, something other than K-V, a dashboard, etc.

## Build and run

Currently, the server is a single node that stores a single value.

To build and run the server:

```
go build
./go-raft
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

## Manual raft requests

### Vote

Request a vote (start an election for the new term):

```
curl -i -X POST localhost:8080/ -d '{"term": 1}'
```

A 200 response means that the other node voted "yay". A 4xx response means that the request was rejected (409 for past term, others for invalid requests).

### Health

Health check:

```
curl localhost:8080/health
```

### Append

[not yet implemented]

### Commit

[not yet implemented]
