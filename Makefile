# Use 'make help' to see available commands

BINARY_NAME=aps
BIN_DIR=bin
VERSION=$(shell cat VERSION.txt 2>/dev/null | sed 's/\.$$//' || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X hop.top/aps/internal/version.Version=$(VERSION) -X hop.top/aps/internal/version.Commit=$(COMMIT) -X hop.top/aps/internal/version.Date=$(DATE) -X hop.top/aps/internal/version.BuiltBy=makefile"
CGO_ENABLED=1
export CGO_ENABLED

.PHONY: all build test lint lint-docs run clean release release-snapshot ci help setup \
	test-stories \
	docker-build-test docker-test-up docker-test-down docker-test-shell \
	docker-test-install docker-test-e2e-user docker-test-cleanup docker-quick-start \
	setup-wsm

all: build test lint

build: ## Build the binary locally
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -buildvcs=false $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/aps
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

test-stories: ## Run tests linked in docs/stories/
	@bash scripts/test-stories.sh $(ARGS)

test-workflows: ## Run GitHub Actions locally with act
	@echo "Running workflows locally with act..."
	@act push --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest

lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run ./...

lint-docs: ## Validate documentation links and referenced test files
	@bash scripts/check-links.sh

run: ## Run the CLI locally
	@go run ./cmd/aps $(ARGS)

setup: ## Install development tools via mise
	@echo "Setting up development environment..."
	@mise install
	@go mod tidy

setup-wsm: ## Set up WSM alongside APS (run from parent directory)
	@echo "Setting up WSM + APS integration..."
	@if [ -f "../wsm/scripts/setup-aps-wsm.sh" ]; then \
		bash ../wsm/scripts/setup-aps-wsm.sh; \
	else \
		echo "Error: wsm repository not found at ../wsm"; \
		echo "Please clone wsm alongside aps in the same parent directory:"; \
		echo "  cd .."; \
		echo "  git clone https://github.com/IdeaCraftersLabs/wsm.git"; \
		exit 1; \
	fi

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

## Docker Testing Targets

docker-build-test: ## Build test Docker image
	@echo "Building test Docker image..."
	@docker build -t aps-test-env -f Dockerfile.test .
	@echo "Test image built successfully"

docker-test-up: ## Start test container in background
	@echo "Starting test container..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Test container started. Use 'make docker-test-shell' to enter."

docker-test-down: ## Stop and remove test containers
	@echo "Stopping test containers..."
	@docker compose -f docker-compose.test.yml down
	@echo "Test containers stopped and removed"

docker-test-shell: ## Start interactive shell in test container
	@echo "Starting interactive shell in test container..."
	@docker run -it --rm \
		-v $(PWD):/host-src:ro \
		-v $(PWD)/tests/fixtures:/test-fixtures:ro \
		aps-test-env \
		/bin/bash

docker-test-install: ## Install built binary in test container
	@echo "Installing binary in test container..."
	@if [ ! -f $(BIN_DIR)/$(BINARY_NAME) ]; then \
		echo "Binary not found. Building..."; \
		$(MAKE) build; \
	fi
	@docker compose -f docker-compose.test.yml run --rm test-env \
		sh -c "cp /host-src/$(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME) && chmod +x /usr/local/bin/$(BINARY_NAME)"
	@echo "Binary installed successfully in test container"

docker-test-e2e-user: ## Run user journey E2E tests in Docker
	@echo "=== Running User Journey E2E Tests ==="
	@echo "Building binary..."
	@$(MAKE) build
	@echo "Building test environment..."
	@$(MAKE) docker-build-test
	@echo "Installing binary in test environment..."
	@$(MAKE) docker-test-install
	@echo "Running user journey tests..."
	@docker compose -f docker-compose.test.yml run --rm test-env \
		/test-fixtures/scripts/run-all-tests.sh
	@echo "=== User Journey Tests Complete ==="

docker-test-cleanup: ## Clean up Docker test artifacts
	@echo "Cleaning up Docker test artifacts..."
	@docker compose -f docker-compose.test.yml down -v
	@docker rmi aps-test-env 2>/dev/null || true
	@echo "Cleanup complete"

docker-quick-start: ## Quick start Docker testing environment
	@echo "=== Docker Quick Start ==="
	@if command -v docker >/dev/null 2>&1; then \
		bash tests/fixtures/scripts/quick-start.sh; \
	else \
		echo "ERROR: Docker is not installed or not in PATH"; \
		exit 1; \
	fi

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
