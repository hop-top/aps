.PHONY: build clean test

BINARY_NAME=aps
BIN_DIR=bin

# Current OS/Arch
OS=$(shell go env GOOS)
ARCH=$(shell go env GOARCH)

build: build-local build-linux build-windows

build-local:
	@echo "Building local binary..."
	go build -o $(BIN_DIR)/$(OS)_$(ARCH)/$(BINARY_NAME) ./cmd/aps
	@ln -sf $(OS)_$(ARCH)/$(BINARY_NAME) $(BIN_DIR)/$(BINARY_NAME)

build-linux:
	@echo "Building Linux amd64..."
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/linux_amd64/$(BINARY_NAME) ./cmd/aps

build-windows:
	@echo "Building Windows amd64..."
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/windows_amd64/$(BINARY_NAME).exe ./cmd/aps

clean:
	@echo "Cleaning bin directory..."
	rm -rf $(BIN_DIR)/*

test:
	go test -v ./tests/e2e
