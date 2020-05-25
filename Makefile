.PHONY: test
test: app
	go test

app:
	protoc -I=./api --go_out=. ./api/*.proto
	go clean
	rm ./app || true
	gofmt -w -s .
	go fix
	go build -o app

.PHONY: run
run: app
	./app

.PHONY: protobuf
protobuf:
	protoc -I=./api --go_out=. ./api/*.proto
	cp ./github.com/btmorr/leifdb/internal/persistence/* ./internal/persistence/
	rm -rf ./github.com