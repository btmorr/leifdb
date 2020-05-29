.PHONY: test
test: app
	go test

app: main.go main_test.go
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
	protoc -I=./api ./api/raft.proto --go_out=plugins=grpc:.
	mkdir -p ./internal/raft
	cp ./github.com/btmorr/leifdb/internal/raft/* ./internal/raft/
	rm -rf ./github.com
