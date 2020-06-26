version = "0.1.0"

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
	npx autorest --typescript --input-file=../docs/swagger.json --output-folder=. --license-header=NONE --package-name=leifdb --package-version=$(version)
	scripts/autorest.post.sh ./src/models/index.ts ./src/models/mappers.ts ./src/models/parameters.ts ./src/leifDbClientAPI.ts ./src/leifDbClientAPIContext.ts
