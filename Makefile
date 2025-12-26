.PHONY: all build test clean lint format run install uninstall

BINARY_NAME := kairo
DIST_DIR := dist
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%d)
LDFLAGS := -X github.com/dkmnx/kairo/internal/version.Version=$(VERSION) -X github.com/dkmnx/kairo/internal/version.Commit=$(COMMIT) -X github.com/dkmnx/kairo/internal/version.Date=$(DATE)
LOCAL_BIN := $(shell echo $$HOME)/.local/bin

all: build

build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) .

test:
	@echo "Running tests..."
	go test -v ./...
	go test -race ./...

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(DIST_DIR)
	go test -coverprofile=$(DIST_DIR)/coverage.out ./...
	go tool cover -html=$(DIST_DIR)/coverage.out -o $(DIST_DIR)/coverage.html
	@echo "Coverage report: $(DIST_DIR)/coverage.html"

lint:
	@echo "Running linters..."
	@gofmt -l .
	@go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

format:
	@echo "Formatting code..."
	gofmt -w .

clean:
	@echo "Cleaning..."
	rm -rf $(DIST_DIR)

install: build
	@echo "Installing $(BINARY_NAME) to ~/.local/bin..."
	install -d $(LOCAL_BIN)
	install -m 755 $(DIST_DIR)/$(BINARY_NAME) $(LOCAL_BIN)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(LOCAL_BIN)/"
	@echo "Add $(LOCAL_BIN) to your PATH if not already present:"

uninstall:
	@echo "Removing $(BINARY_NAME) from ~/.local/bin..."
	rm -f $(LOCAL_BIN)/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from $(LOCAL_BIN)/"

run: build
	@echo "Running $(BINARY_NAME)..."
	$(DIST_DIR)/$(BINARY_NAME) $(ARGS)

build-linux:
	@echo "Building $(BINARY_NAME) for linux..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .

build-darwin:
	@echo "Building $(BINARY_NAME) for darwin..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .

build-windows:
	@echo "Building $(BINARY_NAME) for windows..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .

release: build-linux build-darwin build-windows
	@echo "Release builds created in $(DIST_DIR)/:"
	@ls -la $(DIST_DIR)/$(BINARY_NAME)-* 2>/dev/null || echo "No release builds found"

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

verify-deps:
	@echo "Verifying dependencies..."
	go mod verify

help:
	@echo "Kairo Makefile"
	@echo ""
	@echo "Output directory: $(DIST_DIR)/"
	@echo "Install location: $(LOCAL_BIN)/"
	@echo ""
	@echo "Targets:"
	@echo "  all           - Build the binary (default)"
	@echo "  build         - Build the binary to $(DIST_DIR)/"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linters (gofmt, govet)"
	@echo "  format        - Format code with gofmt"
	@echo "  clean         - Remove $(DIST_DIR)/ directory"
	@echo "  install       - Install to $(LOCAL_BIN)/"
	@echo "  uninstall     - Remove from $(LOCAL_BIN)/"
	@echo "  run           - Build and run with ARGS"
	@echo "  build-linux   - Build for Linux amd64"
	@echo "  build-darwin  - Build for Darwin amd64"
	@echo "  build-windows - Build for Windows amd64"
	@echo "  release       - Build all release binaries (linux, darwin, windows)"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  verify-deps   - Verify dependency checksums"
	@echo "  help          - Show this help message"
