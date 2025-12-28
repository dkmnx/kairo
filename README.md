# Kairo

```text
 █████                 ███
░░███                 ░░░
 ░███ █████  ██████   ████  ████████   ██████
 ░███░░███  ░░░░░███ ░░███ ░░███░░███ ███░░███
 ░██████░    ███████  ░███  ░███ ░░░ ░███ ░███
 ░███░░███  ███░░███  ░███  ░███     ░███ ░███
 ████ █████░░████████ █████ █████    ░░██████
░░░░░ ░░░░░  ░░░░░░░░ ░░░░░ ░░░░░     ░░░░░░
```

[![Version](https://img.shields.io/github/v/release/dkmnx/kairo?style=flat-square)](https://github.com/dkmnx/kairo/releases)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev/dl/)
[![CI Status](https://img.shields.io/github/actions/workflow/status/dkmnx/kairo/ci.yml?branch=main&style=flat-square)](https://github.com/dkmnx/kairo/actions)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

**Kairo** is a Go port of [clauver](https://github.com/dkmnx/clauver), rewritten from Bash to provide:

- **Cross-platform single binary** - Works on Linux, macOS, and Windows
- **Enhanced security** - Type-safe Go implementation with comprehensive validation
- **Easier maintenance** - Go modules, structured testing, and standardized code organization

## Overview

| Feature           | Description                                                            |
| ----------------- | ---------------------------------------------------------------------- |
| Multi-Provider    | Switch between Native Anthropic, Z.AI, MiniMax, Kimi, DeepSeek, custom |
| Secure Encryption | All API keys encrypted with age (X25519) encryption                    |
| Key Rotation      | Periodically rotate encryption keys for enhanced security              |
| Interactive Setup | Guided configuration wizard                                            |
| Provider Testing  | Test connectivity and configuration                                    |
| Auto-Update       | Notifies when new version available                                    |

## Quick Start

```bash
# Install
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh

# Setup
kairo setup

# List providers
kairo list

# Test provider
kairo test zai

# Switch and use Claude
kairo switch zai "Help me write a function"

# Passing arguments directly
kairo -- --continue

# Rotate encryption key (security best practice)
kairo rotate

# Update to latest version
kairo update
```

## Usage Examples

### Code Generation

```bash
# Generate Python code
kairo switch zai "Write a function to calculate fibonacci numbers in Python"

# Generate Go code
kairo switch zai "Implement a REST API in Go using Gin framework"

# Generate with specific provider
kairo switch minimax "Create a React component for a todo list"
```

### Debugging

```bash
# Debug production issues
kairo switch zai "I'm getting a 404 error when accessing /api/users. Here's my code:"

# Get help with error messages
kairo switch zai "Explain this Go error: interface conversion: interface {} is nil, not *User"
```

### Multi-Provider Workflow

```bash
# Setup multiple providers
kairo config zai
kairo config minimax
kairo config deepseek

# Set default for general use
kairo default zai

# Use specific provider when needed
kairo switch minimax "Quick question"
kairo switch deepseek "Batch processing task"
```

### Direct Query Mode

Use the default provider with direct queries using `--`:

```bash
# Set default provider
kairo default zai

# Quick queries with default provider
kairo -- "What's your model?"
kairo -- "Explain quantum computing in simple terms"
kairo -- "Write a haiku about programming"

# Useful for:
# - Quick questions without switching
# - Shell aliases for common queries
# - Scripts that need AI assistance
```

**Tip:** Set up shell aliases for common queries:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias ai='kairo --'
alias explain='kairo -- "Explain'
alias debug='kairo -- "Debug'

# Usage
ai "What's the capital of France?"
explain "How does Kubernetes work?"
debug "Why is my API returning 500?"
```

### CI/CD Integration

```bash
# Use in GitHub Actions
- name: Configure Provider
  env:
    ZAI_API_KEY: ${{ secrets.ZAI_API_KEY }}
  run: |
    kairo config zai
    kairo status

- name: Run AI-powered tests
  run: |
    kairo switch zai "Generate unit tests for this code"
```

For more examples, see [Claude Code Integration Guide](docs/guides/claude-integration-examples.md) \
and [Advanced Configuration Guide](docs/guides/advanced-configuration.md).

## Commands

| Command                      | Description                      |
| ---------------------------- | -------------------------------- |
| `kairo setup`                | Interactive setup wizard         |
| `kairo config <provider>`    | Configure provider               |
| `kairo list`                 | List configured providers        |
| `kairo status`               | Test all providers               |
| `kairo test <provider>`      | Test specific provider           |
| `kairo switch <provider>`    | Switch and exec Claude           |
| `kairo default [provider]`   | Get/set default provider         |
| `kairo reset <provider/all>` | Remove provider config           |
| `kairo rotate`               | Rotate encryption key            |
| `kairo -- "query"`           | Query mode (default provider)    |
| `kairo version`              | Show version + check for updates |
| `kairo update`               | Check for and update to latest   |

## Supported Providers

| Provider            | API Key Required   |
| ------------------- | ------------------ |
| Native Anthropic    | No                 |
| Z.AI                | Yes                |
| MiniMax             | Yes                |
| Kimi (Moonshot)     | Yes                |
| DeepSeek            | Yes                |
| Custom              | Yes                |

## Modules

| Package                | Purpose                     | Documentation                               |
| ---------------------- | --------------------------- | ------------------------------------------- |
| `cmd/`                 | CLI commands (Cobra)        | [cmd/README.md](cmd/README.md)              |
| `internal/config/`     | Configuration loading       | [internal/README.md](internal/README.md)    |
| `internal/crypto/`     | Age encryption              | [internal/README.md](internal/README.md)    |
| `internal/providers/`  | Provider registry           | [internal/README.md](internal/README.md)    |
| `internal/validate/`   | Input validation            | [internal/README.md](internal/README.md)    |
| `pkg/env/`             | Environment utilities       | [pkg/README.md](pkg/README.md)              |

## Documentation

### User Guides

- **Quick Start:** [docs/guides/user-guide.md](docs/guides/user-guide.md) - Installation and usage
- **Error Handling Examples:**
  [docs/guides/error-handling-examples.md](docs/guides/error-handling-examples.md) - Common errors
- **Advanced Configuration:**
  [docs/guides/advanced-configuration.md](docs/guides/advanced-configuration.md) - Multi-provider setup
- **Claude Code Integration:**
  [docs/guides/claude-integration-examples.md](docs/guides/claude-integration-examples.md) - Practical workflows

### Developer Resources

- **Development Guide:**
  [docs/guides/development-guide.md](docs/guides/development-guide.md)
- **Architecture:**
  [docs/architecture/README.md](docs/architecture/README.md)
- **Deployment Guide:**
  [docs/guides/deployment-guide.md](docs/guides/deployment-guide.md)

### Support

- **Troubleshooting:**
  [docs/troubleshooting/README.md](docs/troubleshooting/README.md) - Common issues and solutions
- **Contributing:** [docs/contributing/README.md](docs/contributing/README.md)

## Configuration

Location: `~/.config/kairo/`

| File          | Purpose                        | Permissions |
| ------------- | ------------------------------ | ----------- |
| `config`      | Provider configurations (YAML) | 0600        |
| `secrets.age` | Encrypted API keys             | 0600        |
| `age.key`     | Encryption private key         | 0600        |

## Building

```bash
make build    # Build to dist/kairo
make test     # Run tests with race detection
make lint     # Run code quality checks
make install  # Install to ~/.local/bin
```

## Security

- Age (X25519) encryption for all API keys
- 0600 permissions on sensitive files
- Secrets decrypted in-memory only
- Keys generated on first run (backup recommended)
- Key rotation - use `kairo rotate` to periodically regenerate encryption key

## License

[MIT](LICENSE) (c) 2025 [dkmnx](https://github.com/dkmnx)

## Resources

- [GitHub Repository](https://github.com/dkmnx/kairo)
- [Report Issues](https://github.com/dkmnx/kairo/issues)
- [Changelog](CHANGELOG.md)
