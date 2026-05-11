.PHONY: build test clean release-test fmt tidy help

BINARY_NAME=prune
BIN_DIR=.bin
MAIN_PATH=./cmd/prune
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

## build: build binary locally
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	go build \
		-ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)" \
		-o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary created at $(BIN_DIR)/$(BINARY_NAME)"

## test: run all tests
test:
	@echo "Running tests..."
	go test -v ./...

## release-test: test goreleaser snapshot (Linux/Windows)
release-test:
	@echo "Testing GoReleaser snapshot (Linux/Windows)..."
	goreleaser release --snapshot --clean --config .goreleaser.linux-windows.yaml

fmt:
	@echo "Formatting code..."
	go fmt ./...

tidy:
	@echo "Tidying go modules..."
	go mod tidy

## clean: remove binaries and artifacts
clean:
	@echo "Cleaning up..."
	rm -rf $(BIN_DIR)
	rm -rf dist
	@echo "Cleaned!"

help:
	@echo "Available commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'
