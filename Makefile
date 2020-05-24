.PHONY: test
test: app
	go test

app: clean
	gofmt -w *.go
	go fix
	go build -o app

.PHONY: clean
clean:
	go clean
	rm ./app || true

.PHONY: run
run: app
	./app

