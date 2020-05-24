.PHONY: test
test: app
	go test

app:
	gofmt -w *.go
	go fix
	go clean
	go build -o app

.PHONY: run
run: app
	./app

