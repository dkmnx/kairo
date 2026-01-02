.PHONY: all build test clean lint format run install uninstall goreleaser ci-local

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
	@gofmt -w .
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Formatting issues found (cannot be auto-fixed):"; \
		gofmt -l .; \
		exit 1; \
	fi
	@go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

pre-commit:
	@echo "Running pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "pre-commit not installed. Install with: pip install pre-commit"; \
	fi

pre-commit-install:
	@echo "Installing pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
	else \
		echo "pre-commit not installed. Install with: pip install pre-commit"; \
	fi

format:
	@echo "Formatting code..."
	gofmt -w .

clean:
	@echo "Cleaning..."
	rm -rf $(DIST_DIR)

install:
	@echo "Installing $(BINARY_NAME) to $(LOCAL_BIN)..."
	@if [ -f scripts/install.sh ]; then \
		./scripts/install.sh -b $(LOCAL_BIN); \
	else \
		install -d $(LOCAL_BIN); \
		install -m 755 $(DIST_DIR)/$(BINARY_NAME) $(LOCAL_BIN)/$(BINARY_NAME); \
		echo "Installed $(BINARY_NAME) to $(LOCAL_BIN)/"; \
	fi

uninstall:
	@echo "Removing $(BINARY_NAME) from ~/.local/bin..."
	rm -f $(LOCAL_BIN)/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from $(LOCAL_BIN)/"

run: build
	@echo "Running $(BINARY_NAME)..."
	$(DIST_DIR)/$(BINARY_NAME) $(ARGS)

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

verify-deps:
	@echo "Verifying dependencies..."
	go mod verify

release:
	@echo "Running goreleaser..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		if [ -z "$$GITHUB_TOKEN" ]; then \
			echo "GITHUB_TOKEN not set. "; \
		else \
			goreleaser release --clean; \
		fi; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

release-local:
	@echo "Running goreleaser (snapshot build)..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --clean --snapshot; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

release-dry-run:
	@echo "Running goreleaser (dry-run, no publish)..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --clean --snapshot --skip=publish; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

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
	@echo "  release       - Create release builds with goreleaser"
	@echo "  release-local - Create local snapshot build"
	@echo "  release-dry-run - Build without publishing"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  verify-deps   - Verify dependency checksums"
	@echo "  ci-local      - Run GitHub Actions locally with act"
	@echo "  ci-local-list - List all CI jobs"
	@echo "  help          - Show this help message"

ci-local-list:
	@echo "Listing GitHub Actions jobs..."
	@if command -v act >/dev/null 2>&1; then \
		act -l; \
	else \
		echo "act not installed. Install with:"; \
		echo "  curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sh"; \
		echo "  or: brew install act"; \
	fi

ci-local:
	@echo "Running GitHub Actions locally with act..."
	@if command -v act >/dev/null 2>&1; then \
		act $(CI_ARGS); \
	else \
		echo "act not installed. Install with:"; \
		echo "  curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sh"; \
		echo "  or: brew install act"; \
	fi
