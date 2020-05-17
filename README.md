# Raft practice implementation in Go

This is an attempt to create a clustered K-V store application that implements Raft for consistency, in Go, based on the explanation of Raft from [The Secret Lives of Data](http://thesecretlivesofdata.com/raft/). If/when I get it working for the simplest case (leader accepts GET and POST requests to a specified path to read and write data respectively), then I'll think about other features, such as something other than K-V, a dashboard, etc.

## Build and run

Currently, the server is a single node that stores a single value.

To build and run the server:

```
go build
./ddb
```

To write a value:

```
curl -X POST localhost:8080/ -d '{"value": "test"}'
```

To read the current value:

```
curl localhost:8080
```

## Dev notes

### Backend

Go std library, esp:
- [JSON](https://blog.golang.org/json)
- [HTTP](https://golang.org/pkg/net/http/)
- [path manipulation](https://golang.org/pkg/path/)
- [syncronization primitives](https://golang.org/pkg/sync/)
- [timers/tickers](https://gobyexample.com/timers)

Question: do Go devs actually use `net/http` directly? (seems like yes) Is there really not a framework to make it easier to build routers such that your handlers don't have to be aware of things like HTTP verbs and whatnot?

[Go patterns](https://golang.org/doc/effective_go.html)

JSON Web Tokens [general info](https://jwt.io/introduction/), [go library](https://godoc.org/github.com/dgrijalva/jwt-go), [Auth0 info](https://auth0.com/learn/token-based-authentication-made-easy/)

### Frontend

CSS
- [Flexbox](https://css-tricks.com/snippets/css/a-guide-to-flexbox/)
- [Grid](https://css-tricks.com/snippets/css/complete-guide-grid/)
