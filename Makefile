.PHONY: test
test: app
	go test

app:
	go clean
	rm ./app || true
	gofmt -w -s .
	go fix
	go build -o app

.PHONY: run
run: app
	./app

