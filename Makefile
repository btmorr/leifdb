.PHONY: test
test: app
	go test -coverprofile=coverage.out ./...

viewcoverage: coverage.out
	go tool cover -html=coverage.out

.PHONY: clean
clean:
	go clean
	rm ./app || true

app: clean
	gofmt -w -s .
	go fix
	go build -o app

.PHONY: protobuf
protobuf:
	protoc -I=./api ./api/raft.proto --go_out=plugins=grpc:.
	mkdir -p ./internal/raft
	cp ./github.com/btmorr/leifdb/internal/raft/* ./internal/raft/
	rm -rf ./github.com
