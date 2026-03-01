# Troubleshooting Guide

Common issues and solutions for Kairo.

## Installation

### "command not found: kairo"

Binary not in PATH.

```bash
# Add to PATH
export PATH="$HOME/.local/bin:$PATH"

# Verify
ls -la ~/.local/bin/kairo
```

### Installation Script Fails

Build from source:

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
just build
mkdir -p ~/.local/bin && cp dist/kairo ~/.local/bin/
```

## Configuration

### "config not found"

No configuration exists.

```bash
kairo setup
```

### "permission denied"

Incorrect file permissions.

```bash
chmod 600 ~/.config/kairo/config
chmod 600 ~/.config/kairo/secrets.age
chmod 600 ~/.config/kairo/age.key
```

### "provider not found"

Provider not configured.

```bash
kairo list
kairo setup
```

## Provider Issues

### "invalid API key"

API key validation failed.

```bash
# Requirements: min 8 chars, no whitespace
kairo setup
# Select provider and enter correct API key
```

### "connection refused" / "timeout"

Network or endpoint issue.

```bash
kairo setup
# Select provider to test connectivity
```

### "unsupported provider"

Use lowercase provider names.

```bash
kairo zai     # lowercase provider names
kairo setup   # for custom providers
```

## Encryption

### "failed to decrypt: bad key"

Key mismatch or corruption.

**Options:**

1. Restore from backup
2. Reset (loses all): Re-run `kairo setup` to reconfigure providers

## Claude Execution

### "claude: command not found"

Install Claude Code: <https://claude.com/downloads>

### Execution Failed

```bash
# Reconfigure provider
kairo setup

# Set a default provider
kairo default <provider>

# Test by running with a provider
kairo <provider> "test query"
```

## Shell Completion

```bash
# Bash
kairo completion bash >> ~/.bashrc

# Zsh
kairo completion zsh > ~/.zsh/completion/_kairo

# Fish
kairo completion fish > ~/.config/fish/completions/kairo.fish
```

## Advanced Troubleshooting

### Verbose Mode

```bash
kairo -v list
kairo -v setup
```

### Collect Diagnostics

```bash
kairo version
kairo list
cat ~/.config/kairo/config.yaml
```

## Getting Help

- [GitHub Issues](https://github.com/dkmnx/kairo/issues)
- [User Guide](guides/user-guide.md)
- [Development Guide](guides/development-guide.md)
