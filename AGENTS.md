# AI Agent Context for Kairo

> **Kairo** - Secure CLI for managing Claude Code API providers with age (X25519)
> encryption, multi-provider support, and audit logging.

## 1. Commands

### Essential Commands

| Command                                              | Description                          |
|------------------------------------------------------|--------------------------------------|
| `make build`                                         | Build binary to `dist/kairo`         |
| `make test`                                          | Run all tests with race detector     |
| `go test -v ./cmd -run TestName`                     | Run specific test                    |
| `make lint`                                          | Run gofmt, go vet, golangci-lint     |
| `make format`                                        | Format code with gofmt               |
| `make run ARGS="switch zai"`                         | Build and run with arguments         |
| `make deps`                                          | Download and tidy dependencies       |
| `task build`                                         | Alternative build using Task         |
| `go test -race -coverprofile=coverage.txt ./...`     | CI test command                      |

### Additional Commands

- `make test-coverage` - Run tests with HTML coverage report
- `make clean` - Remove build artifacts
- `make install` - Install to `~/.local/bin`
- `make vuln-scan` - Run govulncheck for vulnerabilities
- `make release-local` - Create snapshot release with goreleaser
- `make pre-commit` - Run pre-commit hooks

## 2. Tech Stack

### Core

- **Language**: Go 1.25.6
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) (v1.10.2)
- **Encryption**: [age](https://filippo.io/age) (v1.2.1) - X25519 key encryption
- **Configuration**: YAML (gopkg.in/yaml.v3)
- **Versioning**: Masterminds/semver (v3.4.0)

### Testing & Quality

- **Testing**: Go standard testing + race detector
- **Linting**: gofmt, go vet, golangci-lint (v1.62.0)
- **Pre-commit**: Pre-commit hooks for code quality
- **Vulnerability Scanning**: govulncheck

### Build & Release

- **Build**: Make, Task (Taskfile.yml), GoReleaser
- **CI/CD**: GitHub Actions (`.github/workflows/ci.yml`)
- **Target Platforms**: Linux, macOS, Windows (amd64, arm64)

## 3. Code Style

### Imports

Group imports in this order with blank lines between:

1. Standard library packages
2. Third-party packages
3. Internal project packages (use `kairoerrors` alias for errors package)

Example:

```go
import (
    "fmt"
    "os"

    "filippo.io/age"
    "github.com/spf13/cobra"

    kairoerrors "github.com/dkmnx/kairo/internal/errors"
    "github.com/dkmnx/kairo/internal/config"
)
```

### Types & Naming

- **Files**: snake_case (e.g., `audit_helpers.go`)
- **Exported**: PascalCase (e.g., `LoadConfig`, `EncryptSecrets`)
- **Unexported**: camelCase (e.g., `loadRecipient`, `getConfigDir`)
- **YAML Tags**: snake_case with underscores (e.g., `default_provider`, `base_url`)
- **Error Types**: Custom `KairoError` with typed categories (`ConfigError`, `CryptoError`, etc.)

### Error Handling

Use the custom error wrapping system with context:

```go
// Wrap errors with type and context
return kairoerrors.WrapError(kairoerrors.ConfigError,
    "failed to load configuration", err).
    WithContext("path", configPath).
    WithContext("hint", "check file permissions")

// Create new errors
return kairoerrors.NewError(kairoerrors.CryptoError,
    "key file is empty").
    WithContext("path", keyPath)
```

### Security Patterns

- **File Permissions**: Always use `0600` for sensitive files (keys, secrets, config)
- **Key Storage**: Never log or print private key material
- **Atomic Operations**: Use `os.Rename` for atomic file replacements
- **Input Handling**: Use `internal/ui` package for secure password prompts

### Testing

- **Table-Driven Tests**: Use `[]struct` pattern with `t.Run(tt.name, func(t *testing.T) {...})`
- **Race Detection**: Always run with `-race` flag
- **Coverage**: Aim for high coverage, generate HTML reports with `make test-coverage`
- **Integration Tests**: Located in `cmd/integration_test.go`
- **Test Naming**: `TestFunctionName` or `TestFile_Method` pattern

### Documentation Standards

- **Package Comments**: Include architecture, testing notes, security considerations
- **Function Comments**: Document parameters, returns, error conditions
- **Thread Safety**: Document when functions are not thread-safe
- **Security Notes**: Add security context where relevant

Example package comment:

```go
// Package crypto provides encryption using the age library.
//
// Thread Safety: Not thread-safe (file I/O)
// Security: All key files use 0600 permissions
// Performance: Uses X25519 for fast, secure operations
```

## 4. Architecture

### Project Structure

```text
kairo/
├── cmd/              # CLI commands (Cobra framework)
│   ├── root.go       # Entry point and command routing
│   ├── setup.go      # Interactive setup wizard
│   ├── config.go     # Provider configuration
│   ├── switch.go     # Provider switching
│   └── *_test.go     # Command tests
├── internal/         # Internal business logic
│   ├── audit/        # Audit logging (config changes)
│   ├── config/       # YAML config loading/migration
│   ├── crypto/       # Age encryption/decryption
│   ├── errors/       # Custom error types and wrapping
│   ├── performance/  # Metrics collection
│   ├── providers/    # Provider registry (Anthropic, Z.AI, etc.)
│   ├── recovery/     # Recovery mechanisms
│   ├── ui/           # Terminal UI and prompts
│   ├── validate/     # Input validation
│   ├── version/      # Version information
│   └── wrapper/      # Wrapper script generation
├── pkg/              # Public reusable packages
│   └── env/          # Cross-platform config directory
├── scripts/          # Install scripts and helpers
└── docs/             # Documentation
```

### Data Flow

1. User runs `kairo <provider>` or `kairo switch <provider>`
2. Config loaded from `~/.config/kairo/config.yaml`
3. Secrets decrypted from `~/.config/kairo/secrets.age` using `age.key`
4. Provider environment variables injected
5. Claude Code CLI executed with provider configuration

### Key Components

- **Configuration**: YAML-based with automatic migration from old format
- **Encryption**: Age X25519 keys, atomic key rotation support
- **Audit**: All config changes logged to `audit.log`
- **Providers**: Built-in support for Anthropic, Z.AI, MiniMax, Kimi, DeepSeek, custom
- **Wrapper**: Shell/PowerShell scripts for environment variable injection

## 5. Development Guidelines

### Commit Messages

Follow Conventional Commits:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation
- `style:` - Formatting (gofmt)
- `refactor:` - Code restructuring
- `test:` - Test additions/changes
- `chore:` - Maintenance tasks

> **Note**: Always check the full diff before committing to ensure the message accurately reflects all changes.

### Debugging

Use the Scientific Method for debugging:

1. **Hypothesis**: Form a clear theory about the issue
2. **Test**: Create minimal reproduction or add logging
3. **Analysis**: Evaluate results, refine hypothesis

**Logging Tools**:

- Use `internal/ui` package for user-facing output (colored, formatted)
- Use standard `log` package for operational logging
- Audit log at `~/.config/kairo/audit.log` tracks all config changes

### Search

Use `mgrep` for codebase search:

```bash
# Search for patterns
mgrep "func.*Encrypt" "*.go"

# Search for specific error types
mgrep "kairoerrors\.WrapError"
```

### Documentation

Use `context7` MCP to fetch latest library documentation:

- Cobra: `/spf13/cobra`
- Age encryption: Check `filippo.io/age`
- Go standard library: Check official docs

## 6. Configuration Files

| File                           | Purpose                              |
| ------------------------------ | ------------------------------------ |
| `go.mod`                       | Go module definition                 |
| `Makefile`                     | Build automation (primary)           |
| `Taskfile.yml`                 | Alternative task runner              |
| `.goreleaser.yaml`             | Release configuration                |
| `.pre-commit-config.yaml`      | Pre-commit hooks                     |
| `.github/workflows/ci.yml`     | CI/CD pipeline                       |
| `.markdownlint-cli2.jsonc`     | Markdown linting rules               |

## 7. Security Checklist

- [ ] Sensitive files use `0600` permissions
- [ ] No secrets logged to stdout/stderr
- [ ] Private keys never exposed in error messages
- [ ] Atomic file operations for key rotation
- [ ] Input validation on all user-provided data
- [ ] Secure password prompts via `term.ReadPassword`

---

**Module**: `github.com/dkmnx/kairo`  
**Go Version**: 1.25.6  
**License**: MIT
