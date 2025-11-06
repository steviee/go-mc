.PHONY: help build test test-race test-integration coverage lint fmt clean install install-hooks

# Variables
BINARY_NAME=go-mc
BUILD_DIR=build
CMD_DIR=cmd/go-mc
GOFLAGS=-v
LDFLAGS=-s -w
COVERAGE_FILE=coverage.out

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/##/  /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) -tags containers_image_openpgp -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: Build for all architectures
build-all:
	@echo "Building for all architectures..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -tags containers_image_openpgp -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build -tags containers_image_openpgp -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	@echo "✅ Binaries built for all architectures"

## test: Run unit tests
test:
	@echo "Running unit tests..."
	go test $(GOFLAGS) -timeout 5m ./...

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test $(GOFLAGS) -race -timeout 5m ./...

## test-integration: Run integration tests (requires Podman)
test-integration:
	@echo "Running integration tests..."
	go test $(GOFLAGS) -race -timeout 10m -tags=integration ./test/...

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage: $$(go tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}')"
	@echo "✅ Coverage report: coverage.html"

## lint: Run linters
lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run --timeout=5m
	@echo "✅ Linting passed"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	gofmt -w -s .
	goimports -w .
	go mod tidy
	@echo "✅ Code formatted"

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html
	@echo "✅ Cleaned"

## install: Install binary to /usr/local/bin (requires sudo)
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed: /usr/local/bin/$(BINARY_NAME)"

## install-hooks: Install pre-commit hooks
install-hooks:
	@echo "Installing pre-commit hooks..."
	@printf '#!/bin/bash\nset -e\n\necho "Running pre-commit checks..."\n\n# Format check\nif [ -n "$$(gofmt -l .)" ]; then\n\techo "❌ Code not formatted. Running gofmt..."\n\tgofmt -w -s .\n\tgit add .\nfi\n\n# Vet\necho "Running go vet..."\ngo vet ./...\n\n# Lint\necho "Running golangci-lint..."\ngolangci-lint run --timeout=5m\n\necho "✅ Pre-commit checks passed"\n' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Pre-commit hook installed"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "✅ Dependencies downloaded"
