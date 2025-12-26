SHELL := /bin/bash

BINARY_NAME=matecommit

.PHONY: all build test test-race test-grep clean lint help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) cmd/main.go
	@echo "Build complete: $(BINARY_NAME)"

test:
	@echo "Running tests..."
	@go test -v ./...

test-race:
	@echo "Running tests with race detection..."
	@go test -v -race ./... 2>&1 | grep -E "^--- FAIL|^FAIL|^panic:|[[:space:]]+Error:" || echo "All tests passed!"

test-grep:
	@echo "Running tests (showing failures)..."
	@go test -v ./... 2>&1 | grep -E "^--- FAIL|^FAIL|^panic:|[[:space:]]+Error:" || echo "All tests passed!"

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@echo "Clean complete."

lint:
	@echo "Linting..."
	@go vet ./...

help:
	@echo "Available commands:"
	@echo "  make build       - Build the binary"
	@echo "  make test        - Run standard tests"
	@echo "  make test-race   - Run tests with race detection (filtered output)"
	@echo "  make test-grep   - Run tests and grep for failures"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make lint        - Run static analysis"
