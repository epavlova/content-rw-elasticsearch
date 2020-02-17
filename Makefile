PROJECT_NAME=content-rw-elasticsearch
.PHONY: all test clean

all: clean test build-readonly

build:
	@echo ">>> Building Application..."
	go build -v ./cmd/${PROJECT_NAME}

build-readonly:
	@echo ">>> Building Application..."
	go build -mod=readonly -v ./cmd/${PROJECT_NAME}

test:
	@echo ">>> Running Unit Tests..."
	go test -race -v ./...

cover-test:
	@echo ">>> Running Tests with Coverage..."
	go test -race ./... -coverprofile=coverage.out -covermode=atomic

clean:
	@echo ">>> Removing binaries..."
	@rm -rf ./${PROJECT_NAME}
	@echo ">>> Cleaning modules cache..."
	go clean -modcache
