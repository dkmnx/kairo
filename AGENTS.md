# Development Rules for Kairo

## Project Overview

**Kairo** is a secure CLI tool for managing Claude Code API providers.
It uses age (X25519) encryption, supports multiple providers, and includes
audit logging. Written in Go.

## Tech Stack

- **Language:** Go 1.25+
- **Build Tool:** Go modules (`go.mod`)
- **Command Runner:** Just (`justfile`)
- **CLI Framework:** Cobra (`github.com/spf13/cobra`)
- **Encryption:** Age/X25519 (`filippo.io/age`)
- **Configuration:** YAML (`gopkg.in/yaml.v3`)
- **Terminal UI:** `golang.org/x/term`
- **Testing:** Go testing framework with race detector

## Commands

### Build

```bash
# Build binary to dist/
just build
go build -o dist/kairo .

# Install to ~/.local/bin/
just install
```

### Test

```bash
# Run all tests
just test
go test -v ./...
go test -race ./...

# Run with coverage
just test-coverage
go test -coverprofile=coverage.out ./...
```

### Pre-release

```bash
# Run all pre-release checks before releasing
just pre-release
```

This command runs the following checks in sequence:
1. **format** - Format code with gofmt
2. **lint** - Run linters (gofmt, go vet, golangci-lint)
3. **pre-commit** - Run all pre-commit hooks
4. **test** - Run all tests with race detector

### Lint

```bash
# Format code
just format
gofmt -w .

# Run linters
just lint
gofmt -w .
gofmt -l .
go vet ./...
golangci-lint run ./...
```

### Dependency Management

```bash
# Install dependencies and tools
just deps
go mod download
go mod tidy

# Verify dependencies
just verify-deps
go mod verify

# Vulnerability scan
just vuln-scan
govulncheck ./...
```

### Releases

```bash
# Create release (requires GITHUB_TOKEN)
just release
goreleaser release --clean

# Local snapshot
just release-local
goreleaser release --clean --snapshot
```

### Pre-commit Hooks

```bash
# Run pre-commit hooks
just pre-commit
pre-commit run --all-files

# Install hooks
just pre-commit-install
pre-commit install
```

### CI/CD

```bash
# Run GitHub Actions locally
just ci-local
act

# List CI jobs
just ci-local-list
act -l
```

## Project Structure

```text
kairo/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command, entry point
│   ├── setup.go           # Interactive setup wizard
│   ├── config.go          # Provider configuration
│   ├── switch.go          # Provider switching/execution
│   ├── audit.go           # Audit logging
│   ├── *_test.go          # Test files
│   └── README.md          # Command documentation
├── internal/               # Business logic (no CLI deps)
│   ├── audit/             # Audit logging
│   ├── config/            # YAML loading/saving
│   ├── crypto/            # Age encryption
│   ├── providers/         # Provider registry
│   ├── validate/          # Input validation
│   ├── ui/               # Terminal UI utilities
│   ├── errors/           # Typed errors
│   └── README.md         # Internal packages docs
├── pkg/                   # Reusable utilities
│   └── env/              # Cross-platform config dir
├── docs/                  # Documentation
├── scripts/              # Installation scripts
├── dist/                 # Build output
├── justfile              # Command runner
├── go.mod/go.sum         # Dependencies
├── .github/workflows/    # CI/CD
└── .pre-commit-config.yaml  # Pre-commit hooks
```

## Code Style

- **Line Length:** 120 characters (MD013)
- **Indentation:** Tabs (Go standard)
- **Naming:** Go conventions (PascalCase for exported, camelCase for unexported)
- **Error Handling:** Typed errors from `internal/errors` package
- **Formatting:** `gofmt -w .` (run before committing)
- **Vetting:** `go vet ./...` (run before committing)

### Error Handling Pattern

```go
import kairoerrors "github.com/dkmnx/kairo/internal/errors"

// Wrap with context
return kairoerrors.WrapError(kairoerrors.ConfigError,
    "failed to load configuration", err).
    WithContext("path", configPath)

// Create new error
return kairoerrors.NewError(kairoerrors.ValidationError,
    "invalid provider name")
```

**Error Types:**

- `ConfigError` - Configuration loading/saving
- `CryptoError` - Encryption/decryption
- `ValidationError` - Input validation
- `ProviderError` - Provider operations
- `FileSystemError` - File operations
- `NetworkError` - Network operations

### Command Handler Pattern

- Minimal business logic in command handlers
- Delegate to internal packages for core functionality
- Consistent error handling with user-friendly messages
- All user input read securely using `ui` package

## Git Rules

- **ONLY commit files YOU changed**
- NEVER use `git add -A` or `git add .`
- Use `git add <specific-files>` for your changes
- Run `git status` before committing to verify
- Commit message format: conventional commits

## Testing

- Run `go test -race ./...` before committing
- Write tests for new commands (`*_test.go` files)
- Integration tests verify end-to-end workflows
- Mock external process execution via `execCommand` variable

## Adding a New Provider

1. **Define in `internal/providers/registry.go`:**

```go
var BuiltInProviders = map[string]ProviderDefinition{
    "newprovider": {
        Name:           "New Provider",
        BaseURL:        "https://api.newprovider.com/anthropic",
        Model:          "new-model",
        RequiresAPIKey: true,
        EnvVars:        []string{},
    },
}
```

1. **Add validation if needed:** Update `internal/validate/` for provider-specific rules

2. **Test:**

```bash
go test ./internal/providers/...
kairo config newprovider
kairo test newprovider
```

## Key Files

- `cmd/root.go` - Entry point, argument parsing
- `internal/config/config.go` - YAML loading/saving
- `internal/crypto/age.go` - Encryption operations
- `internal/providers/registry.go` - Provider definitions
- `justfile` - All build/test/lint commands
