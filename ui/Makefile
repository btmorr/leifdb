version = $(shell bash ../version.sh)

.PHONY: install
install:
	yarn

.PHONY: build
build:
	yarn build

.PHONY: test
test:
	yarn test

.PHONY: autorest
autorest:
	npx autorest --typescript --input-file=../docs/swagger.json --output-folder=. --license-header=eslint-disable --package-name=leifdb --package-version=$(version)
