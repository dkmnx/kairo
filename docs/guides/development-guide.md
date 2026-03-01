# Development Guide

Setup, testing, and contribution workflow for Kairo.

## Prerequisites

- Go 1.25+
- Git

## Setup

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
go mod download
```

## Build & Test

```bash
just build        # Build binary to dist/
just test         # Run tests with race detector
just lint         # Run gofmt, go vet, golangci-lint
just format       # Format code with gofmt
just pre-release  # Format, lint, test
just install      # Install to ~/.local/bin/
```

Manual commands:

```bash
go build -o dist/kairo .
go test -race ./...
gofmt -w .
go vet ./...
```

## Project Structure

```text
kairo/
├── cmd/           # CLI commands (Cobra)
├── internal/      # Business logic
│   ├── config/    # YAML loading
│   ├── crypto/    # age encryption
│   ├── errors/    # Typed errors
│   ├── providers/ # Provider registry
│   ├── ui/        # Terminal UI
│   └── validate/  # Input validation
├── pkg/           # Reusable utilities
│   └── env/       # Cross-platform config dir
└── docs/          # Documentation
```

## Adding a Provider

1. Define in `internal/providers/registry.go`:

```go
var BuiltInProviders = map[string]ProviderDefinition{
    "newprovider": {
        Name:        "New Provider",
        BaseURL:     "https://api.newprovider.com/anthropic",
        Model:       "new-model",
        RequiresAPIKey: true,
    },
}
```

1. Test:

```bash
go test ./internal/providers/...
kairo setup  # Configure new provider
```

## Testing

```bash
# All tests
go test -race ./...

# Specific package
go test -race ./cmd/...
go test -race ./internal/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

Test patterns:

- Table-driven tests for validation
- `t.TempDir()` for isolation
- Mock external dependencies

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Add godoc comments for exported functions
- Return typed errors from internal packages

## Pre-commit

```bash
pip install pre-commit
pre-commit install
```

## Contributing

1. Fork repository
2. Create feature branch: `git checkout -b feature/name`
3. Make changes and test
4. Run `just pre-release`
5. Submit PR

See [Contributing Guide](../contributing/README.md) for:

- Commit message format
- PR description template
- Code review process
