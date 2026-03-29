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

Install one of the supported harness CLIs:

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

# 3. Use a specific provider
kairo zai "Help me write a function"

# 4. Or use the default provider
kairo -- "Quick question"
```

## Commands

| Command                       | Description                                         |
| ----------------------------- | --------------------------------------------------- |
| `kairo setup`                 | Interactive setup wizard                            |
| `kairo setup --reset-secrets` | Regenerate encryption key and re-enter API keys     |
| `kairo list`                  | List configured providers                           |
| `kairo default [provider]`    | Get or set the default provider                     |
| `kairo delete <provider>`     | Delete a provider                                   |
| `kairo <provider> [args]`     | Execute with a specific provider                    |
| `kairo -- [args]`             | Execute with the default provider                   |
| `kairo harness get`           | Get current harness                                 |
| `kairo harness set <name>`    | Set default harness (`claude` or `qwen`)            |
| `kairo update`                | Update to the latest version                        |
| `kairo version`               | Show version                                        |

## Supported Providers

| Provider   | Default Model     | API Key Required |
| ---------- | ----------------- | ---------------- |
| `zai`      | `glm-5.1`         | Yes              |
| `minimax`  | `MiniMax-M2.7`    | Yes              |
| `kimi`     | `kimi-for-coding` | Yes              |
| `deepseek` | `deepseek-chat`   | Yes              |
| `custom`   | user-defined      | Yes              |

Details: [Provider Reference](../reference/providers.md)

## Configuration

### Location

| OS          | Path                                     |
| ----------- | ---------------------------------------- |
| Linux/macOS | `~/.config/kairo/`                       |
| Windows     | `%USERPROFILE%\AppData\Roaming\kairo\`   |

### Files

| File          | Purpose                        |
| ------------- | ------------------------------ |
| `config.yaml` | Provider and harness settings  |
| `secrets.age` | Encrypted API keys             |
| `age.key`     | Encryption private key         |

Details: [Configuration Reference](../reference/configuration.md)

## Security

### Encryption

- All API keys are encrypted with age/X25519
- The encryption key is generated on first setup
- API keys are decrypted only when needed

### Resetting Encrypted Secrets

Use the built-in reset flow if you lose access to `age.key` or want to regenerate the key:

```bash
kairo setup --reset-secrets
```

This deletes the current encrypted secrets and encryption key, generates a new key, and requires you to re-enter all API keys.

### Best Practices

1. Backup `age.key` together with `secrets.age`
2. Never commit secrets or your key file
3. Keep file permissions private (`0600`)
4. Use `kairo setup --reset-secrets` instead of manually deleting only `age.key`

## Troubleshooting

Common issues:

| Issue                | Solution                                            |
| -------------------- | --------------------------------------------------- |
| `command not found`  | Add `~/.local/bin` to PATH                          |
| `provider not found` | Run `kairo setup`                                   |
| `invalid API key`    | Reconfigure with `kairo setup`                      |
| `failed to decrypt`  | Restore backup or run `kairo setup --reset-secrets` |

Full guide: [Troubleshooting](../troubleshooting/README.md)

## Next Steps

- [Architecture](../architecture/README.md) - System design
- [Development Guide](development-guide.md) - Setup and contribution
