# Makefile for paws

# Get version from git tag, fallback to commit hash
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS := -s -w \
	-X github.com/lzecca78/paws/src/cmd.version=$(VERSION)

# Binary name
BINARY := paws

.PHONY: all build clean test install run version help

## Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## Build for all platforms (local goreleaser snapshot)
build-all:
	goreleaser build --snapshot --clean

## Run tests
test:
	go test ./... -v

## Run tests with coverage
test-cover:
	go test ./... -cover

## Run tests with coverage report
test-cover-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf dist/

## Install the binary to $GOPATH/bin
install:
	go install -ldflags "$(LDFLAGS)" .

## Run the application
run: build
	./$(BINARY)

## Show version that would be embedded
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

## Lint the code
lint:
	golangci-lint run ./...

## Format the code
fmt:
	go fmt ./...

## Tidy dependencies
tidy:
	go mod tidy

## Release (requires goreleaser and GITHUB_TOKEN)
release:
	goreleaser release --clean

## Release snapshot (for testing)
release-snapshot:
	goreleaser release --snapshot --clean

## Show help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"

# Default target
all: build
