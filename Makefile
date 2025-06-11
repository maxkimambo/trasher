# Makefile for trasher

# Variables
BINARY_NAME=trasher
BIN_DIR=bin
PKG=./...
VERSION?=dev

# Build flags
LDFLAGS=-ldflags "-X github.com/maxkimambo/trasher/cmd.version=$(VERSION)"

# Default target
.DEFAULT_GOAL := build

# Create bin directory if it doesn't exist
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Build the application
.PHONY: build
build: $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) main.go

# Run tests
.PHONY: test
test:
	go test -v $(PKG)

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out $(PKG)
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	rm -f test_*.dat test_*.checksum.txt

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Format code
.PHONY: fmt
fmt:
	go fmt $(PKG)

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Run the application (requires build)
.PHONY: run
run: build
	./$(BIN_DIR)/$(BINARY_NAME)

# Development build (with debug info)
.PHONY: dev
dev: $(BIN_DIR)
	go build -gcflags="all=-N -l" -o $(BIN_DIR)/$(BINARY_NAME) main.go

# Release build for multiple platforms
.PHONY: release
release: clean
	mkdir -p $(BIN_DIR)/release
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/release/$(BINARY_NAME)-windows-amd64.exe main.go

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application (default)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts and test files"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  run           - Build and run the application"
	@echo "  dev           - Build with debug info"
	@echo "  release       - Build for multiple platforms"
	@echo "  help          - Show this help message"