# Kairo

```text
 █████                 ███
░░███                 ░░░
 ░███ █████  ██████   ████  ████████   ██████
 ░███░░███  ░░░░░███ ░░███ ░░███░░███ ███░░███
 ░██████░    ███████  ░███  ░███ ░░░  ░███ ░███
 ░███░░███  ███░░███  ░███  ░███      ░███ ░███
 ████ █████░░████████ █████ █████     ░░██████
░░░░░ ░░░░░  ░░░░░░░░ ░░░░░ ░░░░░       ░░░░░░
```

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go)](https://go.dev/dl/)
[![CI Status](https://img.shields.io/github/actions/workflow/status/dkmnx/kairo/ci.yml?branch=main&style=flat-square)](https://github.com/dkmnx/kairo/actions)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

**Kairo** is a Go CLI tool for managing Claude Code API providers with encrypted secrets management using age (X25519) encryption.

## Overview

- **Multi-Provider Support**: Switch between Native Anthropic, Z.AI, MiniMax, Kimi, DeepSeek, and custom providers
- **Secure Encryption**: All API keys encrypted with age (X25519) encryption
- **Interactive Setup**: Guided configuration wizard
- **Provider Testing**: Test connectivity and configuration

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

# Or use default provider
kairo "Quick query"
```

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
| `kairo "query"`              | Query mode (default provider)    |
| `kairo version`              | Show version                     |

## Supported Providers

| Provider          | API Key Required |
|-------------------|------------------|
| Native Anthropic  | No               |
| Z.AI              | Yes              |
| MiniMax           | Yes              |
| Kimi (Moonshot)   | Yes              |
| DeepSeek          | Yes              |
| Custom            | Yes              |

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

- **User Guide:** [docs/guides/user-guide.md](docs/guides/user-guide.md)
- **Development Guide:** [docs/guides/development-guide.md](docs/guides/development-guide.md)
- **Architecture:** [docs/architecture/README.md](docs/architecture/README.md)
- **Troubleshooting:** [docs/troubleshooting/README.md](docs/troubleshooting/README.md)
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

## License

[MIT](LICENSE) (c) 2025 [dkmnx](https://github.com/dkmnx)

## Resources

- [GitHub Repository](https://github.com/dkmnx/kairo)
- [Report Issues](https://github.com/dkmnx/kairo/issues)
- [Changelog](CHANGELOG.md)
