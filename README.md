# Kairo

```text
 █████                 ███                    
░░███                 ░░░                     
 ░███ █████  ██████   ████  ████████   ██████ 
 ░███░░███  ░░░░░███ ░░███ ░░███░░███ ███░░███
 ░██████░    ███████  ░███  ░███ ░░░ ░███ ░███
 ░███░░███  ███░░███  ░███  ░███     ░███ ░███
 ████ █████░░████████ █████ █████    ░░██████ 
░░░░ ░░░░░  ░░░░░░░░ ░░░░░ ░░░░░      ░░░░░░  
```

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

**Kairo** is a Go CLI tool for managing Claude Code API providers. It's a Go
port of [clauver](https://github.com/dkmnx/clauver) (bash-based CLI) focused on core provider switching with
encrypted secrets management using age encryption.

## Features

- **Multi-Provider Support**: Switch between multiple Claude API providers
  including Native Anthropic, Z.AI, MiniMax, Kimi, DeepSeek, and custom
  providers
- **Secure Secrets Management**: All API keys are encrypted using age (X25519) encryption
- **Interactive Setup**: Guided configuration wizard for easy setup
- **Provider Testing**: Test connectivity and configuration for all providers
- **Default Provider**: Set and switch to a default provider for quick queries
- **Environment Variables**: Supports custom environment variables per provider

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/dkmnx/kairo.git
# Change to the project directory
cd kairo
# Build the binary (outputs to ./dist/kairo)
make build
# Install to ~/.local/bin
make install
```

## Quick Start

```bash
# Interactive setup wizard
kairo setup

# Configure a specific provider
kairo config zai

# List all configured providers
kairo list

# Test a specific provider
kairo test zai

# Switch to provider and run Claude
kairo switch zai "Help me write a function"

# Set default provider
kairo default zai

# Reset/remove a specific provider
kairo reset zai

# Reset all providers
kairo reset all

# Use default provider (query mode)
kairo "Help me debug this"
```

## Commands

| Command                | Description                                   |
|------------------------|-----------------------------------------------|
| `kairo setup`          | Interactive setup wizard                      |
| `kairo config`         | Configure provider (API key, URL, model)      |
| `kairo list`           | List all configured providers                 |
| `kairo status`         | Test connectivity for all providers           |
| `kairo switch`         | Switch and exec Claude with args              |
| `kairo default`        | Get or set default provider                   |
| `kairo test`           | Test specific provider connectivity           |
| `kairo reset`          | Reset/remove a provider configuration         |
| `kairo [query]`        | Query mode using default provider             |
| `kairo version`        | Show version                                  |

## Supported Providers

- **Native Anthropic**: Official Anthropic API
- **Z.AI**: Z.AI API (`api.z.ai`)
- **MiniMax**: MiniMax API (`api.minimax.io`)
- **Kimi**: Moonshot AI (`api.kimi.com`)
- **DeepSeek**: DeepSeek AI (`api.deepseek.com`)
- **Custom Provider**: Define your own provider endpoint

## Configuration

### Config Directory

All configuration is stored in `~/.config/kairo/`:

| File          | Purpose                                       |
|---------------|-----------------------------------------------|
| `config`      | Provider configurations (YAML, 0600)          |
| `secrets.age` | Encrypted API keys (age, 0600)                |
| `age.key`     | Encryption private key (age, 0600)            |

### Config File Format

```yaml
default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
    env_vars:
      - ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.5-air
  anthropic:
    name: Native Anthropic
    base_url: ""
    model: ""
  custom:
    name: Custom Provider
    base_url: ""
    model: ""
```

## Security

- **Encryption**: Uses age (X25519) encryption for all API keys
- **File Permissions**: Sensitive files use 0600 permissions
- **Memory Safety**: Secrets decrypted in-memory via process substitution only
- **Key Management**: Keys generated on first run, must be backed up

## Dependencies

- **filippo.io/age**: Encryption library
- **github.com/spf13/cobra**: CLI framework
- **gopkg.in/yaml.v3**: YAML parsing

## Development

```bash
# Run all tests with race detection
make test

# Run tests with coverage report
make test-coverage

# Build the binary
make build

# Install to ~/.local/bin
make install

# Run code quality checks
make lint

# Format code
make format

# Run pre-commit hooks locally
make pre-commit
```

### Makefile Targets

| Target               | Description                               |
| -------------------- | ----------------------------------------- |
| `make build`         | Build binary to `dist/kairo`              |
| `make test`          | Run tests with race detection             |
| `make test-coverage` | Generate coverage report                  |
| `make lint`          | Run gofmt, go vet, golangci-lint          |
| `make format`        | Format code with gofmt                    |
| `make pre-commit`    | Run pre-commit hooks                      |
| `make install`       | Install to `~/.local/bin`                 |
| `make uninstall`     | Remove from `~/.local/bin`                |
| `make clean`         | Remove build artifacts                    |
| `make release`       | Create release with goreleaser            |

### CI/CD Pipeline

This project uses GitHub Actions for continuous integration and deployment:

- **CI Workflow** (`.github/workflows/ci.yml`):
  - Code quality checks (gofmt, go vet)
  - Security scanning (gosec, govulncheck)
  - Multi-version testing (Go 1.21, 1.22, 1.23)
  - Multi-platform builds (Linux, macOS, Windows)
  - Dependency validation and tidy check
  - Coverage reporting

- **Release Workflow** (`.github/workflows/release.yml`):
  - Draft release creation on tag push
  - Snapshot releases for pull requests
  - goreleaser for multi-platform builds
  - Homebrew formula updates

### Pre-commit Hooks

Install pre-commit for local quality gates:

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run all hooks
pre-commit run --all-files
```

Available hooks:

- `gofmt` - Code formatting check
- `go vet` - Static analysis
- `go test` - Run tests with race detection
- `go mod tidy` - Verify go.mod/go.sum consistency

## Validation Rules

### API Key Validation

- Minimum length: 8 characters

### URL Validation

- Valid URL format (scheme://host/path)
- Scheme: https only
- Blocked hosts:
  - localhost, 127.0.0.1, ::1
  - Private IPs: 10.x.x.x, 172.16-31.x.x, 192.168.x.x
  - Link-local: 169.254.x.x
  - file:// URLs

## Architecture

```text
kairo/
├── cmd/                  # CLI command implementations
├── internal/             # Private application code
│   ├── config/           # Configuration loading/saving
│   ├── crypto/           # age encryption/decryption
│   ├── ui/               # User interface
│   ├── providers/        # Provider registry and definitions
│   └── validate/         # Input validation
├── pkg/                  # Public libraries
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
└── main.go               # Application entry point 
```

## License

[MIT](LICENSE) (c) 2025 [dkmnx](https://github.com/dkmnx)
