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
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go)](https://go.dev/dl/)
[![CI Status](https://img.shields.io/github/actions/workflow/status/dkmnx/kairo/ci.yml?branch=main&style=flat-square)](https://github.com/dkmnx/kairo/actions)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

**Go CLI wrapper for Claude Code and Qwen Code with X25519-encrypted API keys.**

## Overview

Kairo provides multi-provider API management with secure credential storage:

- **Multi-harness**: Claude Code and Qwen Code
- **Secure encryption**: age/X25519 for all API keys at rest
- **Built-in providers**: Z.AI, MiniMax, Kimi, DeepSeek, and custom providers
- **Cross-platform**: Linux, macOS, Windows

## Quick Start

### Install

- Linux/macOS: `curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh`
- Windows: `irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex`

### Prerequisites

Install one of the supported harness CLIs:

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code

# Qwen Code
npm install -g @qwen-code/qwen-code@latest
```

### Setup

```bash
kairo setup          # Interactive setup wizard
kairo list           # List configured providers
kairo zai "query"    # Use a specific provider
kairo -- "query"     # Use the default provider
```

## Commands

| Command                       | Description                                     |
| ----------------------------- | ----------------------------------------------- |
| `kairo setup`                 | Interactive setup wizard                        |
| `kairo setup --reset-secrets` | Regenerate encryption key and re-enter API keys |
| `kairo list`                  | List configured providers                       |
| `kairo default [provider]`    | Get or set the default provider                 |
| `kairo delete <provider>`     | Delete a provider                               |
| `kairo <provider> [args]`     | Execute with a specific provider                |
| `kairo -- [args]`             | Execute with the default provider               |
| `kairo harness get`           | Get the current harness                         |
| `kairo harness set <name>`    | Set the default harness                         |
| `kairo update`                | Update to the latest version                    |
| `kairo version`               | Show version information                        |

Full reference: [docs/reference/configuration.md](docs/reference/configuration.md)

## Configuration

Locations:

- Linux/macOS: `~/.config/kairo/`
- Windows: `%USERPROFILE%\AppData\Roaming\kairo\`

Files:

- `config.yaml` - provider and harness settings
- `secrets.age` - encrypted API keys
- `age.key` - encryption private key

## Security

- X25519 encryption for all API keys
- `0600` permissions on sensitive files
- In-memory decryption during use
- Temporary wrapper scripts for secure token passing to harness CLIs
- Recovery/reset flow via `kairo setup --reset-secrets`

See [Security Architecture](docs/architecture/README.md#security-architecture)

## Documentation

- [User Guide](docs/guides/user-guide.md) - Installation and usage
- [Development Guide](docs/guides/development-guide.md) - Setup and contribution
- [Architecture](docs/architecture/README.md) - System design
- [Troubleshooting](docs/troubleshooting/README.md) - Common issues

Full documentation: [docs/README.md](docs/README.md)

## Development

```bash
just build
just test
just lint
just pre-release
```

## Resources

- [GitHub](https://github.com/dkmnx/kairo)
- [Report Issues](https://github.com/dkmnx/kairo/issues)

---

**License:** [MIT](LICENSE) | **Author:** [dkmnx](https://github.com/dkmnx)
