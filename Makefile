# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy go.mod
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go mod verify

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs -timeout 30s ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build: build all examples to verify they compile
.PHONY: build
build:
	go build ./examples/...

# ==================================================================================== #
# RELEASE
# Following the conventions from Alex Edwards' "Let's Go Further":
# always audit and test before tagging, always use annotated tags,
# always push the tag explicitly so it appears on GitHub Releases.
# ==================================================================================== #

## release/tag version=vX.Y.Z: create and push an annotated semantic version tag
## Example: make release/tag version=v0.1.0
.PHONY: release/tag
release/tag: confirm tidy audit test
	git tag -a ${version} -m "Release ${version}"
	git push origin ${version}
	@echo "Tagged and pushed ${version}"

## release/check: list the most recent release tags
.PHONY: release/check
release/check:
	git tag -l "v*" --sort=-version:refname | head -20
