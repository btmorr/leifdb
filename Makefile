version = $(shell bash ./version.sh)

# Note: be careful with values for `binary_prefix`, because of how it is used
# in the "clean" task--it will delete any files with this prefix
binary_prefix = leifdb-
tag ?= 0

.PHONY: test
test: app
	go test -v -tags=unit -coverprofile=coverage.out ./...
	go test -v -tags=mgmttest -coverprofile=mgmttest_coverage.out ./...

.PHONY: viewcoverage
viewcoverage: coverage.out
	go tool cover -html=coverage.out
	go tool cover -html=mgmttest_coverage.out

.PHONY: benchmark
benchmark:
	go test -v -tags=bench -bench=. ./...

.PHONY: clean
clean:
	go clean
	rm -rf ./build || true
	rm ./$(binary_prefix)* || true

.PHONY: install
install:
	go install github.com/swaggo/swag/cmd/swag
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

.PHONY: app
app: clean
	swag init
	gofmt -w -s .
	go vet
	go build -tags=unit,mgmttest -o $(binary_prefix)$(version)

.PHONY: protobuf
protobuf:
	protoc -I=./api ./api/raft.proto --go_out=. --go-grpc_out=.
	mkdir -p ./internal/raft
	cp ./github.com/btmorr/leifdb/internal/raft/* ./internal/raft/
	rm -rf ./github.com

.PHONY: linuxbin
linuxbin:
	env GOOS=linux GOARCH=amd64 go build -o build/leifdb

.PHONY: testcluster
testcluster: linuxbin
	make -C ui build
	docker-compose up --build
