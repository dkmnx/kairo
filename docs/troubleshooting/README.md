# Troubleshooting Guide

Common issues and solutions for Kairo.

## Installation

### `command not found: kairo`

Binary not in `PATH`.

```bash
export PATH="$HOME/.local/bin:$PATH"
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

### `config not found`

No configuration exists yet.

```bash
kairo setup
```

### `permission denied`

Incorrect file permissions.

```bash
chmod 600 ~/.config/kairo/config.yaml
chmod 600 ~/.config/kairo/secrets.age
chmod 600 ~/.config/kairo/age.key
```

### `provider not found`

Provider is not configured.

```bash
kairo list
kairo setup
```

## Provider Issues

### `invalid API key`

API key validation failed.

Current validation rules:

- Built-in providers: minimum 32 characters
- Custom providers: minimum 20 characters

Re-run setup and enter the correct key:

```bash
kairo setup
```

### `connection refused` or timeout errors

Check the configured base URL and network access.

```bash
kairo list
kairo setup   # Edit the provider if needed
```

### `unsupported provider`

Use the configured provider name exactly as shown by `kairo list`.

```bash
kairo zai
kairo setup   # for custom providers
```

## Encryption

### `failed to decrypt` or bad key errors

Your `age.key` does not match `secrets.age`, or one of them is corrupted.

Recovery options:

1. Restore both `age.key` and `secrets.age` from backup
2. Reset encrypted secrets and re-enter API keys:

```bash
kairo setup --reset-secrets
```

## Harness Execution

### `claude: command not found`

Install Claude Code.

### `qwen: command not found`

Install Qwen Code.

### Execution Failed

```bash
kairo setup
kairo default <provider>
kairo <provider> "test query"
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
- [User Guide](../guides/user-guide.md)
- [Development Guide](../guides/development-guide.md)
