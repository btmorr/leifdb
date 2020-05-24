.PHONY: test
test: app
	go test

app:
	gofmt -w *.go
	go fix
	go clean
	golint
	go build -o app

.PHONY: run
run: app
	./app

