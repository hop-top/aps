# Use 'make help' to see available commands

BINARY_NAME=aps
BIN_DIR=bin
VERSION=$(shell cat VERSION.txt 2>/dev/null | sed 's/\.$$//' || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X oss-aps-cli/internal/version.Version=$(VERSION) -X oss-aps-cli/internal/version.Commit=$(COMMIT) -X oss-aps-cli/internal/version.Date=$(DATE) -X oss-aps-cli/internal/version.BuiltBy=makefile"

.PHONY: all build test lint run clean release release-snapshot ci help setup

all: build test lint

build: ## Build the binary locally
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/aps
	@echo "Binary built at $(BIN_DIR)/$(BINARY_NAME)"

test: test-go test-workflows ## Run all tests (Go and Workflows)

test-go: ## Run standard Go tests
	@echo "Running Go tests..."
	@go test -v ./...

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	@go test -v ./tests/unit/...

test-e2e: ## Run E2E tests
	@echo "Running E2E tests..."
	@go test -v ./tests/e2e/...

test-workflows: ## Run GitHub Actions locally with act
	@echo "Running workflows locally with act..."
	@act push --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

run: ## Run the CLI locally
	@go run ./cmd/aps $(ARGS)

setup: ## Install development tools via mise
	@echo "Setting up development environment..."
	@mise install
	@go mod tidy

clean: ## Remove build artifacts
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	@rm -rf dist/
	@rm -f coverage.out coverage-e2e.out coverage-merged.out coverage-report.txt coverage.html

release-snapshot: ## Build release snapshots locally
	@echo "Building release snapshots..."
	@goreleaser release --snapshot --clean

release: ## Run a full release (requires git tag)
	@echo "Running full release..."
	@goreleaser release --clean

ci: ## Trigger remote CI workflows on GitHub
	@echo "Triggering remote CI workflows..."
	@gh workflow run ci.yml
	@gh workflow run platform-adapter-tests.yml
	@gh workflow run security.yml

local-ci: test-workflows ## Alias for test-workflows

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
