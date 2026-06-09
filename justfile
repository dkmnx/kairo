# Justfile - Command runner for Kairo project
# Converted from Makefile to maintain functional equivalence
#
# Key syntax differences from Make:
# - Variables use {{VARIABLE}} for substitution
# - Shell commands use $() syntax
# - No .PHONY needed (Just is a command runner, not build system)
# - Recipe indentation: 2 spaces (must be consistent)
# - @ prefix suppresses command echoing (same as Make)
# - - prefix ignores errors (same as Make)

# Use PowerShell on Windows, sh on Unix
set windows-shell := ["powershell", "-NoProfile", "-Command"]
set shell := ["sh", "-c"]

# Detect Go binary
GO := "go"

# Variables
BINARY_NAME := if os() == "windows" { "kairo.exe" } else { "kairo" }
DIST_DIR := "dist"
VERSION := `git describe --tags --always --dirty`
COMMIT := `git rev-parse --short HEAD`
DATE := `date -u +%Y-%m-%d`
LDFLAGS := "-X github.com/dkmnx/kairo/internal/version.Version=" + VERSION + " -X github.com/dkmnx/kairo/internal/version.Commit=" + COMMIT + " -X github.com/dkmnx/kairo/internal/version.Date=" + DATE

# Race detector flag: disabled on Windows (requires cgo/C compiler)
RACE_FLAG := if os() == "windows" { "" } else { "-race" }

# Default recipe: show list of recipes
[default]
_:
    @just --list

# Build the binary
build:
    @echo "Building {{BINARY_NAME}} {{VERSION}}..."
    {{GO}} build -ldflags "{{LDFLAGS}}" -o {{DIST_DIR}}/{{BINARY_NAME}} .

# Run all tests (with race detector on platforms that support it)
test:
    @echo "Running tests..."
    {{GO}} test -v {{RACE_FLAG}} ./...

# Run fuzzing tests with timeout
fuzz:
    @echo "Running fuzzing tests (5s per test)..."
    @echo ""
    @echo "=== internal/validate ==="
    {{GO}} test -fuzz=FuzzValidateAPIKey -fuzztime=5s ./internal/validate/
    {{GO}} test -fuzz=FuzzValidateURL -fuzztime=5s ./internal/validate/
    {{GO}} test -fuzz=FuzzValidateProviderModel -fuzztime=5s ./internal/validate/
    {{GO}} test -fuzz=FuzzValidateCrossProviderConfig -fuzztime=5s ./internal/validate/
    @echo ""
    @echo "=== cmd ==="
    {{GO}} test -fuzz=FuzzValidateCustomProviderName -fuzztime=5s ./cmd/
    @echo ""
    @echo "Fuzzing tests completed!"

# Run tests with coverage report
test-coverage:
    @echo "Running tests with coverage..."
    {{GO}} test -coverprofile={{DIST_DIR}}/coverage.out {{RACE_FLAG}} ./...
    {{GO}} tool cover -html={{DIST_DIR}}/coverage.out -o {{DIST_DIR}}/coverage.html
    @echo "Coverage report: {{DIST_DIR}}/coverage.html"

# Run linters
lint:
    @echo "Running linters..."
    {{GO}} vet ./...
    golangci-lint run ./...

# Format code
format:
    @echo "Formatting code..."
    gofmt -w .

# Run pre-commit hooks
pre-commit:
    @echo "Running pre-commit hooks..."
    pre-commit run --all-files

# Install pre-commit hooks
pre-commit-install:
    @echo "Installing pre-commit hooks..."
    pre-commit install

# Pre-release checks: format, lint, pre-commit, test, goreleaser dry-run
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
    @echo "Running goreleaser dry-run..."
    goreleaser release --clean --snapshot
    @echo ""
    @echo "Pre-release checks passed!"

# Clean build artifacts
[unix]
clean:
    @echo "Cleaning..."
    rm -rf {{DIST_DIR}}

[windows]
clean:
    @echo "Cleaning..."
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue {{DIST_DIR}}

# Install binary to GOBIN or GOPATH/bin
install: build
    @echo "Installing {{BINARY_NAME}}..."
    {{GO}} install {{DIST_DIR}}/{{BINARY_NAME}}

# Uninstall binary
[unix]
uninstall:
    @echo "Removing {{BINARY_NAME}}..."
    rm -f "$(go env GOPATH)/bin/{{BINARY_NAME}}"
    @echo "Uninstalled {{BINARY_NAME}}"

[windows]
uninstall:
    @echo "Removing {{BINARY_NAME}}..."
    Remove-Item -Force -ErrorAction SilentlyContinue "$(go env GOPATH)\bin\{{BINARY_NAME}}"
    @echo "Uninstalled {{BINARY_NAME}}"

# Build and run with arguments
run args="": build
    @echo "Running {{BINARY_NAME}}..."
    {{DIST_DIR}}/{{BINARY_NAME}} {{args}}

# Download and tidy dependencies + install dev tools
deps:
    @echo "Installing dependencies..."
    {{GO}} mod download
    {{GO}} mod tidy
    @echo ""
    @echo "Installing development tools..."
    {{GO}} install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
    {{GO}} install golang.org/x/vuln/cmd/govulncheck@latest
    {{GO}} install honnef.co/go/tools/cmd/staticcheck@latest
    {{GO}} install github.com/goreleaser/goreleaser/v2@latest
    @echo ""
    @echo "Installing pre-commit..."
    command -v pre-commit >/dev/null 2>&1 && echo "pre-commit already installed" || (uv pip install pre-commit && pre-commit install)
    @echo ""
    @echo "Dependencies installed!"

# Verify dependencies
verify-deps:
    @echo "Verifying dependencies..."
    {{GO}} mod verify

# Run vulnerability scan
vuln-scan:
    @echo "Running vulnerability scan..."
    govulncheck ./...

# Run comprehensive security checks (vuln scan + lint)
security: vuln-scan lint
    @echo "Security checks passed!"

# Create release builds with goreleaser
release:
    @echo "Running goreleaser..."
    goreleaser release --clean

# Create local snapshot build
release-local:
    @echo "Running goreleaser (snapshot build)..."
    goreleaser release --clean --snapshot

# Build without publishing (dry-run)
release-dry-run:
    @echo "Running goreleaser (dry-run, no publish)..."
    goreleaser release --clean --snapshot --skip=publish

# Generate provider table for README
generate:
    @echo "Generating provider table..."
    {{GO}} run ./cmd/gen/
    @echo "Provider table generated! Update README.md with the output."

# Display help message
help:
    @echo "Kairo Justfile"
    @echo ""
    @echo "Output directory: {{DIST_DIR}}/"
    @echo ""
    @echo "Recipes:"
    @echo "  build           - Build the binary to {{DIST_DIR}}/"
    @echo "  test            - Run all tests"
    @echo "  fuzz            - Run fuzzing tests (5s per package)"
    @echo "  test-coverage   - Run tests with coverage report"
    @echo "  lint            - Run linters (go vet, golangci-lint)"
    @echo "  format          - Format code with gofmt"
    @echo "  pre-commit      - Run pre-commit hooks"
    @echo "  pre-release     - Run all pre-release checks (format, lint, pre-commit, test)"
    @echo "  clean           - Remove {{DIST_DIR}}/ directory"
    @echo "  install         - Install to GOBIN"
    @echo "  uninstall       - Remove from GOBIN"
    @echo "  run             - Build and run with ARGS"
    @echo "  release         - Create release builds with goreleaser"
    @echo "  release-local   - Create local snapshot build"
    @echo "  release-dry-run - Build without publishing"
    @echo "  deps            - Download and tidy dependencies"
    @echo "  verify-deps     - Verify dependency checksums"
    @echo "  vuln-scan       - Run vulnerability scan with govulncheck"
    @echo "  security        - Run comprehensive security checks (vuln-scan + lint)"
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
    act -l

# Run GitHub Actions locally
ci-local ci_args="":
    @echo "Running GitHub Actions locally with act..."
    act {{ci_args}}
