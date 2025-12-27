# User Guide

Complete guide for end-users of Kairo CLI.

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh
```

### Build from Source

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
make build
# Binary outputs to ./dist/kairo
```

### Verify Installation

```bash
kairo version
```

## Quick Start

```bash
# 1. Run interactive setup wizard
kairo setup

# 2. List configured providers
kairo list

# 3. Test a provider
kairo test zai

# 4. Switch to provider and use Claude
kairo switch zai "Help me write a function"

# 5. Or use default provider directly
kairo "Help me debug this"
```

## Commands Reference

### `kairo setup`

Interactive setup wizard for initial configuration.

```bash
kairo setup
```

Guides you through:

- Provider selection
- API key configuration
- Custom provider setup (optional)

### `kairo config <provider>`

Configure a specific provider.

```bash
# Configure Z.AI provider
kairo config zai

# Configure custom provider
kairo config custom
```

Prompts for:

- API key (masked input)
- Base URL (for custom providers)
- Model name (optional)

### `kairo list`

List all configured providers.

```bash
kairo list
```

Output:

```text
 Configured Providers:
  - zai (default)
  - anthropic
  - custom
```

### `kairo status`

Test connectivity for all configured providers.

```bash
kairo status
```

Output:

```text
Provider Status:
✓ zai     - Connected
✓ anthropic - Connected
✗ custom  - Failed: invalid API key
```

### `kairo test <provider>`

Test a specific provider's connectivity.

```bash
kairo test zai
```

### `kairo switch <provider> [query]`

Switch to a provider and optionally run a Claude query.

```bash
# Switch provider
kairo switch zai

# Switch and execute query
kairo switch zai "Explain goroutines"

# Short form
kairo switch zai
```

### `kairo default [provider]`

Get or set the default provider.

```bash
# Get current default
kairo default

# Set default provider
kairo default zai
```

### `kairo reset <provider>|all`

Remove a provider configuration.

```bash
# Remove specific provider
kairo reset zai

# Remove all providers
kairo reset all
```

### `kairo [query]`

Query mode using default provider.

```bash
kairo "Help me write a unit test"
kairo "What is the capital of France?"
```

### `kairo version`

Display version information.

```bash
kairo version
```

### `kairo completion`

Generate shell completion scripts.

```bash
# Bash
kairo completion bash

# Zsh
kairo completion zsh

# Fish
kairo completion fish
```

## Supported Providers

| Provider     | Description                | API Key Required    |
| ------------ | -------------------------- | ------------------- |
| anthropic    | Native Anthropic API       | No                  |
| zai          | Z.AI API                   | Yes                 |
| minimax      | MiniMax API                | Yes                 |
| kimi         | Moonshot AI (Kimi)         | Yes                 |
| deepseek     | DeepSeek AI                | Yes                 |
| custom       | User-defined provider      | Yes                 |

## Configuration Files

Location: `~/.config/kairo/`

| File          | Purpose                   | Permissions   |
| ------------- | ------------------------- | ------------- |
| `config`      | Provider configurations   | 0600          |
| `secrets.age` | Encrypted API keys        | 0600          |
| `age.key`     | Encryption private key    | 0600          |

## Security

### Encryption

- All API keys are encrypted using age (X25519) encryption
- Keys are generated on first run
- Secrets are decrypted in-memory only when needed

### Best Practices

1. **Backup your `age.key`** - Without it, you cannot decrypt your API keys
2. **Never commit secrets** - Config files contain no plaintext keys
3. **Use 0600 permissions** - Files are automatically set with correct permissions

## Troubleshooting

### Provider Connection Failures

```bash
# Check provider status
kairo status

# Test specific provider
kairo test <provider>

# Reconfigure provider
kairo config <provider>
```

### Common Issues

**Issue:** "Provider not found"

```bash
# List available providers
kairo list

# Configure if missing
kairo config <provider>
```

**Issue:** "Invalid API key"

```bash
# Reconfigure with correct key
kairo config <provider>
```

**Issue:** "Cannot decrypt secrets"

```bash
# Check if age.key exists
ls -la ~/.config/kairo/

# If missing, reset and reconfigure
kairo reset all
kairo setup
```

See [Troubleshooting Guide](../troubleshooting/README.md) for more solutions.
