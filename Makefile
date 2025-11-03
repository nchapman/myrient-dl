.PHONY: build test lint fmt clean install help coverage

# Variables
BINARY_NAME=myrient-dl
COVERAGE_FILE=coverage.out

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	go build -o $(BINARY_NAME) .

install: ## Install the binary to GOPATH/bin
	go install

test: ## Run tests
	go test -v -race ./...

coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_FILE)
	rm -f coverage.html
	rm -rf dist/

tidy: ## Tidy and verify module dependencies
	go mod tidy
	go mod verify

all: fmt lint test build ## Run fmt, lint, test, and build

ci: lint test ## Run CI checks (lint and test)
