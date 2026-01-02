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

**Secure CLI for managing Claude Code API providers** with age (X25519) encryption, multi-provider support, and audit logging.

## Quick Start

```bash
# Install
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh

# Setup
kairo setup

# Configure a provider
kairo config zai

# Test provider
kairo test zai

# Switch and query
kairo switch zai "Help me write a function"

# Or use default provider
kairo -- "Quick question"
```

## Commands

| Command                       | Description                        |
| ----------------------------- | ---------------------------------- |
| `kairo setup`                 | Interactive setup wizard           |
| `kairo config <provider>`     | Configure a provider               |
| `kairo list`                  | List configured providers          |
| `kairo status`                | Test all providers                 |
| `kairo test <provider>`       | Test specific provider             |
| `kairo switch <provider>`     | Switch and exec Claude             |
| `kairo default <provider>`    | Get/set default provider           |
| `kairo reset <provider\|all>` | Remove provider config             |
| `kairo rotate`                | Rotate encryption key              |
| `kairo audit <list\|export>`  | View/export audit logs             |
| `kairo -- "query"`            | Query mode (default provider)      |
| `kairo version`               | Show version info                  |
| `kairo update`                | Check for updates                  |

## Features

| Feature                  | Description                                                              |
| ------------------------ | ------------------------------------------------------------------------ |
| **Multi-Provider**       | Native Anthropic, Z.AI, MiniMax, Kimi, DeepSeek, custom                  |
| **Secure Encryption**    | Age (X25519) encryption for all API keys                                 |
| **Key Rotation**         | Regenerate encryption keys periodically                                  |
| **Audit Logging**        | Track all configuration changes                                          |
| **Interactive Setup**    | Guided configuration wizard                                              |
| **Provider Testing**     | Test connectivity and configuration                                      |
| **Auto-Update**          | Notifications for new versions                                           |

## Documentation

**User Guides:**

- [User Guide](docs/guides/user-guide.md) - Installation and usage
- [Audit Guide](docs/guides/audit-guide.md) - Audit log usage
- [Integration Examples](docs/guides/claude-integration-examples.md) - Practical workflows

**Developer Resources:**

- [Development Guide](docs/guides/development-guide.md) - Setup and contribution
- [Architecture](docs/architecture/README.md) - System design and diagrams
- [Contributing](docs/contributing/README.md) - Contribution workflow

**Reference:**

- [Troubleshooting](docs/troubleshooting/README.md) - Common issues and solutions
- [Changelog](CHANGELOG.md) - Version history

## Modules

```text
kairo/
├── cmd/           # CLI commands (Cobra) → [cmd/README.md](cmd/README.md)
├── internal/      # Business logic
│   ├── audit/     # Audit logging
│   ├── config/    # Configuration loading
│   ├── crypto/    # Age encryption
│   ├── providers/ # Provider registry
│   ├── validate/  # Input validation
│   └── ui/        # Terminal output
└── pkg/           # Reusable utilities → [pkg/README.md](pkg/README.md)
```

## Configuration

Location: `~/.config/kairo/`

| File          | Purpose                           | Permissions |
| ------------- | --------------------------------- | ----------- |
| `config`      | Provider configurations (YAML)    | 0600        |
| `secrets.age` | Encrypted API keys                | 0600        |
| `age.key`     | Encryption private key            | 0600        |
| `audit.log`   | Configuration change history      | 0600        |

## Building

```bash
make build    # Build to dist/kairo
make test     # Run tests with race detection
make lint     # Run code quality checks
make format   # Format code with gofmt
```

## Security

- Age (X25519) encryption for all API keys
- 0600 permissions on sensitive files
- Secrets decrypted in-memory only
- Key generation on first run
- Use `kairo rotate` for periodic key rotation

## License

[MIT](LICENSE) (c) 2025 [dkmnx](https://github.com/dkmnx)

## Resources

- [GitHub](https://github.com/dkmnx/kairo)
- [Report Issues](https://github.com/dkmnx/kairo/issues)
