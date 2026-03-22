# Development Guide

Setup, testing, and contribution workflow for Kairo.

## Prerequisites

- Go 1.26+
- Git
- [just](https://github.com/casey/just)

## Setup

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
go mod download
```

## Build & Test

```bash
just build          # Build binary to dist/
just test           # Run tests, then race-enabled tests
just test-coverage  # Generate coverage report in dist/
just lint           # Run gofmt checks, go vet, golangci-lint
just format         # Format code with gofmt
just pre-release    # Format, lint, pre-commit, test
just install        # Install to ~/.local/bin/
```

Manual commands:

```bash
go build -o dist/kairo .
go test -v ./...
go test -race ./...
go test -coverprofile=dist/coverage.out ./...
go tool cover -func=dist/coverage.out
gofmt -w .
go vet ./...
```

## Project Structure

```text
kairo/
├── cmd/                # Cobra command layer
├── internal/           # Business logic
│   ├── config/         # Config loading, migration, caching, paths
│   ├── crypto/         # age/X25519 key management and encryption
│   ├── errors/         # Typed errors
│   ├── providers/      # Built-in provider registry
│   ├── ui/             # Terminal output and prompts
│   ├── validate/       # Validation helpers
│   ├── version/        # Build metadata
│   └── wrapper/        # Secure wrapper scripts for token passing
├── docs/               # Project documentation
├── scripts/            # Install and utility scripts
├── main.go             # Application entry point
└── justfile            # Development commands
```

## Adding a Provider

1. Add the provider definition in `internal/providers/registry.go`:

```go
var BuiltInProviders = map[string]ProviderDefinition{
    "newprovider": {
        Name:           "New Provider",
        BaseURL:        "https://api.newprovider.com/anthropic",
        Model:          "new-model",
        RequiresAPIKey: true,
    },
}
```

1. Add the provider key to `providerOrder` in the same file so it appears in setup menus.

2. If needed, add provider-specific API key validation in `internal/validate/api_key.go`.

3. Update user and reference docs.

4. Run targeted tests:

```bash
go test ./internal/providers/... ./internal/validate/...
```

## Testing

```bash
# All tests
go test -v ./...
go test -race ./...

# Specific package
go test ./cmd/...
go test ./internal/providers/...

# With coverage
go test -coverprofile=dist/coverage.out ./...
go tool cover -func=dist/coverage.out
```

Common test patterns used in this project:

- Table-driven tests
- `t.TempDir()` for filesystem isolation
- Mocked command execution for CLI integration points
- Race detector coverage for concurrency-sensitive code

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Add godoc comments for exported functions
- Return typed errors from internal packages
- Keep internal packages free of Cobra/CLI dependencies

## Pre-commit

```bash
pip install pre-commit
pre-commit install
```

## Contributing

1. Fork repository
2. Create a branch: `git checkout -b feature/name`
3. Make changes and test them
4. Run `just pre-release`
5. Submit a PR

See [Contributing Guide](../contributing/README.md) for PR format and commit guidance.
