version = $(shell bash ./version.sh)
run_opts ?=
binary_prefix = leifdb-

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
	go get -u github.com/swaggo/swag/cmd/swag

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

# Note this is not working completely correctly now, as the default config file
# does not work out of the box--revisit after consolidating configration and
# making it more amenable to containerization
.PHONY: container
container:
	env GOOS=linux GOARCH=amd64 go build -o build/leifdb
	cp config/default_config.toml build/config.toml
	docker build -t leifdb:0 .
