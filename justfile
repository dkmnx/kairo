# Justfile - Command runner for Kairo project
# Converted from Makefile to maintain functional equivalence
#
# Key syntax differences from Make:
# - Variables use {{VARIABLE}} for substitution
# - Shell commands use backticks: `command`
# - No .PHONY needed (Just is a command runner, not build system)
# - Recipe indentation: 2 spaces (must be consistent)
# - @ prefix suppresses command echoing (same as Make)
# - - prefix ignores errors (same as Make)

# Variables
BINARY_NAME := "kairo"
DIST_DIR := "dist"
VERSION := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +%Y-%m-%d 2>/dev/null || date +%Y-%m-%d`
LDFLAGS := "-X github.com/dkmnx/kairo/internal/version.Version=" + VERSION + " -X github.com/dkmnx/kairo/internal/version.Commit=" + COMMIT + " -X github.com/dkmnx/kairo/internal/version.Date=" + DATE
LOCAL_BIN := `echo "$HOME/.local/bin" 2>/dev/null || echo "$USERPROFILE/.local/bin"`

# Default recipe: show list of recipes
[default]
_:
    @just --list

# Build the binary
build:
    @echo "Building {{BINARY_NAME}} {{VERSION}}..."
    @mkdir -p {{DIST_DIR}}
    go build -ldflags "{{LDFLAGS}}" -o {{DIST_DIR}}/{{BINARY_NAME}} .

# Run all tests
test:
    @echo "Running tests..."
    go test -v ./...
    go test -race ./...

# Run fuzzing tests with timeout
fuzz:
    @echo "Running fuzzing tests (5s per test)..."
    @echo ""
    @echo "=== internal/validate ==="
    go test -fuzz=FuzzValidateAPIKey -fuzztime=5s ./internal/validate/
    go test -fuzz=FuzzValidateURL -fuzztime=5s ./internal/validate/
    go test -fuzz=FuzzValidateProviderModel -fuzztime=5s ./internal/validate/
    go test -fuzz=FuzzValidateCrossProviderConfig -fuzztime=5s ./internal/validate/
    @echo ""
    @echo "=== cmd ==="
    go test -fuzz=FuzzValidateCustomProviderName -fuzztime=5s ./cmd/
    @echo ""
    @echo "Fuzzing tests completed!"

# Run tests with coverage report
test-coverage:
    @echo "Running tests with coverage..."
    @mkdir -p {{DIST_DIR}}
    go test -coverprofile={{DIST_DIR}}/coverage.out ./...
    go tool cover -html={{DIST_DIR}}/coverage.out -o {{DIST_DIR}}/coverage.html
    @echo "Coverage report: {{DIST_DIR}}/coverage.html"

# Run linters
lint:
    @echo "Running linters..."
    gofmt -w .
    @if [ -n "$(gofmt -l .)" ]; then \
        echo "Formatting issues found (cannot be auto-fixed):"; \
        gofmt -l .; \
        exit 1; \
    fi
    go vet ./...
    @if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run ./...; \
    else \
        echo "golangci-lint not installed, skipping"; \
    fi

# Run pre-commit hooks
pre-commit:
    @echo "Running pre-commit hooks..."
    @if command -v pre-commit >/dev/null 2>&1; then \
        pre-commit run --all-files; \
    else \
        echo "pre-commit not installed. Install with: pip install pre-commit"; \
    fi

# Install pre-commit hooks
pre-commit-install:
    @echo "Installing pre-commit hooks..."
    @if command -v pre-commit >/dev/null 2>&1; then \
        pre-commit install; \
    else \
        echo "pre-commit not installed. Install with: pip install pre-commit"; \
    fi

# Pre-release checks: format, lint, pre-commit, test
pre-release:
    @echo "Running pre-release checks..."
    @echo ""
    just format
    @echo ""
    just lint
    @echo ""
    just pre-commit
    @echo ""
    just test
    @echo ""
    @echo "Pre-release checks passed!"

# Format code
format:
    @echo "Formatting code..."
    gofmt -w .

# Clean build artifacts
clean:
    @echo "Cleaning..."
    rm -rf {{DIST_DIR}}

# Install binary to local bin directory
install: build
    @echo "Installing {{BINARY_NAME}} to {{LOCAL_BIN}}..."
    @if [ -f scripts/install.sh ]; then \
        ./scripts/install.sh -b {{LOCAL_BIN}}; \
    else \
        mkdir -p {{LOCAL_BIN}}; \
        install -m 755 {{DIST_DIR}}/{{BINARY_NAME}} {{LOCAL_BIN}}/{{BINARY_NAME}}; \
        echo "Installed {{BINARY_NAME}} to {{LOCAL_BIN}}/"; \
    fi

# Uninstall binary
uninstall:
    @echo "Removing {{BINARY_NAME}} from ~/.local/bin..."
    rm -f {{LOCAL_BIN}}/{{BINARY_NAME}}
    @echo "Uninstalled {{BINARY_NAME}} from {{LOCAL_BIN}}/"

# Build and run with arguments
run args="": build
    @echo "Running {{BINARY_NAME}}..."
    {{DIST_DIR}}/{{BINARY_NAME}} {{args}}

# Download and tidy dependencies
deps:
    @echo "Installing dependencies..."
    go mod download
    go mod tidy
    
    @echo "Installing development tools..."
    @if command -v golangci-lint >/dev/null 2>&1; then \
        echo "golangci-lint already installed"; \
    else \
        echo "Installing golangci-lint..."; \
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
    fi
    
    @if command -v pre-commit >/dev/null 2>&1; then \
        echo "pre-commit already installed"; \
    else \
        echo "Installing pre-commit..."; \
        pip install pre-commit; \
    fi
    
    @if command -v govulncheck >/dev/null 2>&1; then \
        echo "govulncheck already installed"; \
    else \
        echo "Installing govulncheck..."; \
        go install golang.org/x/vuln/cmd/govulncheck@latest; \
    fi
    
    @if command -v goreleaser >/dev/null 2>&1; then \
        echo "goreleaser already installed"; \
    else \
        echo "Installing goreleaser..."; \
        go install github.com/goreleaser/goreleaser@v1.26.0 2>&1 || \
        echo "⚠️  goreleaser installation failed (requires Go 1.26+ for full installation)"; \
        echo "   goreleaser is only needed for releases. Skipping for now."; \
    fi
    
    @if command -v act >/dev/null 2>&1; then \
        echo "act already installed"; \
    else \
        echo "Installing act..."; \
        curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sh; \
    fi
    
    @if command -v staticcheck >/dev/null 2>&1; then \
        echo "staticcheck already installed"; \
    else \
        echo "Installing staticcheck..."; \
        go install honnef.co/go/tools/cmd/staticcheck@latest; \
    fi
    
    @echo "Installing pre-commit hooks..."
    pre-commit install

# Verify dependencies
verify-deps:
    @echo "Verifying dependencies..."
    go mod verify

# Run vulnerability scan
vuln-scan:
    @echo "Running vulnerability scan..."
    @if command -v govulncheck >/dev/null 2>&1; then \
        govulncheck ./...; \
    else \
        echo "govulncheck not installed. Install with:"; \
        echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"; \
        exit 1; \
    fi

# Create release builds with goreleaser
release:
    @echo "Running goreleaser..."
    @if command -v goreleaser >/dev/null 2>&1; then \
        if [ -z "$GITHUB_TOKEN" ]; then \
            echo "GITHUB_TOKEN not set."; \
        else \
            goreleaser release --clean; \
        fi; \
    else \
        echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
    fi

# Create local snapshot build
release-local:
    @echo "Running goreleaser (snapshot build)..."
    @if command -v goreleaser >/dev/null 2>&1; then \
        goreleaser release --clean --snapshot; \
    else \
        echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
    fi

# Build without publishing (dry-run)
release-dry-run:
    @echo "Running goreleaser (dry-run, no publish)..."
    @if command -v goreleaser >/dev/null 2>&1; then \
        goreleaser release --clean --snapshot --skip=publish; \
    else \
        echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
    fi

# Display help message
help:
    @echo "Kairo Justfile"
    @echo ""
    @echo "Output directory: {{DIST_DIR}}/"
    @echo "Install location: {{LOCAL_BIN}}/"
    @echo ""
    @echo "Recipes:"
    @echo "  default         - Build the binary (default)"
    @echo "  build           - Build the binary to {{DIST_DIR}}/"
    @echo "  test            - Run all tests"
    @echo "  fuzz            - Run fuzzing tests (10s per package)"
    @echo "  test-coverage   - Run tests with coverage report"
    @echo "  lint            - Run linters (gofmt, govet)"
    @echo "  format          - Format code with gofmt"
    @echo "  pre-commit      - Run pre-commit hooks"
    @echo "  pre-release     - Run all pre-release checks (format, lint, pre-commit, test)"
    @echo "  clean           - Remove {{DIST_DIR}}/ directory"
    @echo "  install         - Install to {{LOCAL_BIN}}/"
    @echo "  uninstall       - Remove from {{LOCAL_BIN}}/"
    @echo "  run             - Build and run with ARGS"
    @echo "  release         - Create release builds with goreleaser"
    @echo "  release-local   - Create local snapshot build"
    @echo "  release-dry-run - Build without publishing"
    @echo "  deps            - Download and tidy dependencies"
    @echo "  verify-deps     - Verify dependency checksums"
    @echo "  vuln-scan       - Run vulnerability scan with govulncheck"
    @echo "  ci-local        - Run GitHub Actions locally with act"
    @echo "  ci-local-list   - List all CI jobs"
    @echo "  help            - Show this help message"
    @echo ""
    @echo "Recipe arguments:"
    @echo "  just run 'args --help'      - Run with specific arguments"
    @echo "  just ci-local '-l'          - List CI jobs"

# List all CI jobs
ci-local-list:
    @echo "Listing GitHub Actions jobs..."
    @if command -v act >/dev/null 2>&1; then \
        act -l; \
    else \
        echo "act not installed. Install with:"; \
        echo "  curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sh"; \
        echo "  or: brew install act"; \
    fi

# Run GitHub Actions locally
ci-local ci_args="":
    @echo "Running GitHub Actions locally with act..."
    @if command -v act >/dev/null 2>&1; then \
        act {{ci_args}}; \
    else \
        echo "act not installed. Install with:"; \
        echo "  curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sh"; \
        echo "  or: brew install act"; \
    fi
