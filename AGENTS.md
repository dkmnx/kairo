# AI Assistant Directives

## Commands

- **Build**: `make build` - Build to dist/kairo with version injection
- **Test**: `make test` - Run all tests (verbose + race detection)
- **Test (Single)**: `go test ./package -run TestName` - Run a specific test
- **Lint**: `make lint` - Run gofmt, go vet, golangci-lint
- **Run**: `make run` - Build and run with ARGS
- **Coverage**: `make test-coverage` - Generate HTML coverage report at dist/coverage.html
- **Deps**: `make deps` - Download and tidy dependencies

## Tech Stack

- **Language**: Go 1.25
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Encryption**: age X25519 (filippo.io/age)
- **Config Format**: YAML (gopkg.in/yaml.v3)
- **Versioning**: Semver (github.com/Masterminds/semver/v3)
- **Terminals**: golang.org/x/term

## Code Style

- **Imports**: Standard library first, then third-party; blank line between groups
- **Types**: Strong typing with Go structs
- **Naming**: Exported = PascalCase; unexported = camelCase; constants = ALL_CAPS; files = snake_case.go
- **Err Handling**: Use custom KairoError types from `internal/errors` with `WrapError()` and `.WithContext()` chains
- **File Permissions**: Sensitive files use 0600 explicitly
- **YAML Tags**: Use snake_case (e.g., `yaml:"default_provider"`)

## Architecture

```
kairo/
├── cmd/              # CLI commands using Cobra
├── internal/         # Business logic (no CLI deps)
│   ├── audit/        # Audit logging
│   ├── config/       # YAML loading/saving, migration
│   ├── crypto/       # Age encryption (X25519)
│   ├── errors/       # Custom error types
│   ├── providers/    # Provider registry
│   ├── validate/     # Input validation
│   ├── version/      # Build info injection
│   └── wrapper/      # Claude Code wrapper execution
└── pkg/              # Reusable utilities
    └── env/          # Cross-platform config dir detection
```

## Debugging

Use the Scientific Method: Observe → Form Hypothesis → Test (isolated test) → Analyze → Iterate. Add strategic prints at function entry, after transformations, and before return.

## Search

Use `mgrep` for codebase search with specific patterns over broad queries.

## Documentation

Use `context7` MCP server for up-to-date library documentation. Main docs in `docs/` directory.
