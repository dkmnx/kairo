# User Guide

Installation, setup, and basic usage for Kairo CLI.

## Installation

### Quick Install

```bash
# Linux/macOS
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh

# Windows
irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex
```

### Build from Source

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
just build
mkdir -p ~/.local/bin && cp dist/kairo ~/.local/bin/
```

### Prerequisites

Kairo requires Claude Code or Qwen Code CLI:

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code

# Qwen Code
npm install -g @qwen-code/qwen-code@latest
```

Verify:

```bash
claude --version
# or
qwen --version
```

## Quick Start

```bash
# 1. Run setup wizard
kairo setup

# 2. List providers
kairo list

# 3. Use specific provider
kairo zai "Help me write a function"

# 4. Or use default provider
kairo -- "Quick question"
```

## Commands

| Command                    | Description                                   |
| -------------------------- | --------------------------------------------- |
| `kairo setup`              | Interactive setup wizard (add/edit providers) |
| `kairo list`               | List configured providers                     |
| `kairo delete <provider>`  | Delete a provider                             |
| `kairo <provider> [args]`  | Execute with specific provider                |
| `kairo -- [args]`          | Query with default provider                   |
| `kairo harness get`        | Get current harness                           |
| `kairo harness set <name>` | Set default harness (claude or qwen)          |
| `kairo update`             | Update to latest version                      |
| `kairo version`            | Show version                                  |
| `kairo completion <shell>` | Generate shell completion                     |

## Supported Providers

| Provider  | API Key Required |
| --------- | ---------------- |
| anthropic | No               |
| zai       | Yes              |
| minimax   | Yes              |
| kimi      | Yes              |
| deepseek  | Yes              |
| custom    | Yes              |

Details: [Reference: Providers](../reference/providers.md)

## Configuration

### Location

| OS      | Path                                   |
| ------- | -------------------------------------- |
| Linux   | `~/.config/kairo/`                     |
| macOS   | `~/Library/Application Support/kairo/` |
| Windows | `%APPDATA%\kairo\`                     |

### Files

| File          | Purpose                 |
| ------------- | ----------------------- |
| `config.yaml` | Provider configurations |
| `secrets.age` | Encrypted API keys      |
| `age.key`     | Encryption key          |

Details: [Reference: Configuration](../reference/configuration.md)

## Security

### Encryption

- All API keys encrypted with age (X25519)
- Keys generated on first run
- Decryption only in-memory

### Key Rotation

To rotate your encryption key:

1. Backup your current configuration: `cp -r ~/.config/kairo ~/.config/kairo.backup`
2. Remove age.key file: `rm ~/.config/kairo/age.key`
3. Re-run setup: `kairo setup`

This will regenerate the encryption key and re-encrypt your API keys.

Rotate periodically (monthly) or after security incidents.

### Best Practices

1. Backup `age.key` - Required to decrypt API keys
2. Never commit secrets - Config contains no plaintext
3. Use 0600 permissions - Automatic
4. Rotate keys regularly

## Troubleshooting

Common issues:

| Issue                | Solution                       |
| -------------------- | ------------------------------ |
| "command not found"  | Add `~/.local/bin` to PATH     |
| "provider not found" | Run `kairo setup`              |
| "invalid API key"    | Reconfigure with `kairo setup` |
| "failed to decrypt"  | Restore from backup or reset   |

Full guide: [Troubleshooting](troubleshooting/README.md)

## Next Steps

- [Architecture](../architecture/README.md) - System design
- [Development Guide](development-guide.md) - Setup and contribution
