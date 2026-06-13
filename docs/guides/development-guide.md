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
just fuzz           # Run fuzzing tests (5s per func)
just security       # Run govulncheck + lint
just format         # Format code with gofmt
just deps           # Install dependencies and dev tools
just verify-deps    # Verify Go module dependencies
just pre-release    # Format, lint, pre-commit, test
just install        # Install to ~/.local/bin/
just clean          # Remove build artifacts
just run            # Build and run with arguments
```

Run `just --list` for all available recipes.

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
│   ├── constants/      # Shared constants (paths, defaults)
│   ├── crypto/         # age/X25519 key management and encryption
│   ├── errors/         # Typed errors
│   ├── execution/      # Harness execution dispatch
│   ├── fsutil/         # Atomic file write utility
│   ├── harness/        # Harness dispatch (Claude, Qwen, Pi, Crush)
│   ├── providers/      # Built-in provider registry
│   ├── secrets/        # Secrets loading and saving
│   ├── ui/             # Terminal output and prompts
│   ├── update/         # Self-update logic
│   ├── validate/       # Validation helpers
│   ├── version/        # Build metadata
│   └── wrapper/        # Secure wrapper scripts for token passing
├── docs/               # Project documentation
├── scripts/            # Install and utility scripts
├── main.go             # Application entry point
└── justfile            # Development commands
```

## Adding a Provider

### Custom providers (config only)

Define providers in `config.yaml` under `custom_providers` — no code changes needed:

```yaml
custom_providers:
  my-llm:
    name: My LLM
    base_url: https://api.example.com/anthropic
    model: custom-model
    requires_api_key: true
    api_key_env_var: MY_LLM_API_KEY
    min_key_length: 32
    key_prefix: sk-
```

Custom providers override built-in providers with the same key.

### Built-in providers (code)

1. Add the provider definition in `internal/providers/registry.go`:

```go
var builtInProviders = map[string]ProviderDefinition{
    // ... existing entries ...
    "newprovider": {
        Name:           "New Provider",
        BaseURL:        "https://api.newprovider.com/anthropic",
        Model:          "new-model",
        RequiresAPIKey: true,
        APIKeyEnvVar:   "NEWPROVIDER_API_KEY",
        KeyFormat:      KeyFormatMin32,
    },
}
```

1. Add the provider key to `providerOrder` in the same file so it appears in setup menus.

2. Run targeted tests:

```bash
go test ./internal/providers/... ./internal/validate/...
```

1. Update user and reference docs.

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
if command -v pre-commit >/dev/null 2>&1; then
  echo "pre-commit already installed"
  pre-commit install
elif command -v uv >/dev/null 2>&1; then
  uv tool install pre-commit
  export PATH="$HOME/.local/bin:$PATH"
  pre-commit install
elif command -v pip >/dev/null 2>&1; then
  pip install --user pre-commit
  export PATH="$HOME/.local/bin:$PATH"
  pre-commit install
else
  echo "ERROR: Install uv or pip first" >&2
  exit 1
fi

# Note: if pre-commit was just installed, you may need to restart your shell
# or run: export PATH="$HOME/.local/bin:$PATH"
```

## Contributing

1. Fork repository
2. Create a branch: `git checkout -b feature/name`
3. Make changes and test them
4. Run `just pre-release`
5. Submit a PR

See [Contributing Guide](../CONTRIBUTING.md) for PR format and commit guidance.
