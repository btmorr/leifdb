version = $(shell bash ./version.sh)
run_opts ?=
binary_prefix = leifdb-

.PHONY: test
test: app
	go test -tags=unit -coverprofile=coverage.out ./...
	go test -tags=mgmttest -coverprofile=mgmttest_coverage.out ./...

.PHONY: viewcoverage
viewcoverage: coverage.out
	go tool cover -html=coverage.out
	go tool cover -html=mgmttest_coverage.out

.PHONY: benchmark
benchmark:
	go test -tags=bench -bench=. ./...

.PHONY: clean
clean:
	go clean
	rm ./$(binary_prefix)* || true

.PHONY: install
install:
	go get -u github.com/swaggo/swag/cmd/swag

.PHONY: app
app: clean
	swag init
	gofmt -w -s .
	go vet
	go fix ./...
	go build -tags=unit,mgmttest -o $(binary_prefix)$(version)

.PHONY: protobuf
protobuf:
	protoc -I=./api ./api/raft.proto --go_out=plugins=grpc:.
	mkdir -p ./internal/raft
	cp ./github.com/btmorr/leifdb/internal/raft/* ./internal/raft/
	rm -rf ./github.com

.PHONY: run
run: leifdb-$(version)
	./leifdb-$(version) $(run_opts)
