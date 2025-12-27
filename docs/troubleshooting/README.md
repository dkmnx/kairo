# Troubleshooting Guide

Common issues and solutions for Kairo.

## Installation Issues

### "command not found: kairo"

**Cause:** Binary not in PATH

**Solution:**

```bash
# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH="$HOME/.local/bin:$PATH"

# Verify installation
ls -la ~/.local/bin/kairo
```

### Installation Script Fails

**Solution:** Build from source

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
make build
mkdir -p ~/.local/bin
cp dist/kairo ~/.local/bin/
```

## Configuration Issues

### "config not found"

**Cause:** No configuration file exists

**Solution:**

```bash
kairo setup
```

### "permission denied" on config files

**Cause:** Incorrect file permissions

**Solution:**

```bash
chmod 600 ~/.config/kairo/config
chmod 600 ~/.config/kairo/secrets.age
chmod 600 ~/.config/kairo/age.key
```

### "provider not found"

**Cause:** Provider not configured

**Solution:**

```bash
# List available providers
kairo list

# Configure provider
kairo config <provider>
```

## Provider Issues

### "invalid API key"

**Cause:** API key validation failed

**Solution:**

```bash
# Reconfigure with correct key
kairo config <provider>

# Verify key meets requirements:
# - Minimum 8 characters
# - No whitespace
```

### "connection refused" / "timeout"

**Cause:** Network or provider endpoint issue

**Solution:**

```bash
# Test provider connectivity
kairo test <provider>

# Check provider status
kairo status

# Verify base URL (for custom providers)
kairo config custom
# Enter correct HTTPS URL
```

### "unsupported provider"

**Cause:** Provider name not recognized

**Solution:**

```bash
# Use lowercase provider name
kairo config zai  # NOT "ZAI"

# Or use "custom" for undefined providers
kairo config custom
```

## Encryption Issues

### "failed to decrypt: bad key"

**Cause:** Wrong or corrupted encryption key

**Solution:**

```bash
# Check if age.key exists
ls -la ~/.config/kairo/

# If missing, reset and reconfigure
kairo reset all
kairo setup
```

**Warning:** Resetting will lose all configured API keys.

### "failed to generate key"

**Cause:** Permission or disk space issue

**Solution:**

```bash
# Check disk space
df -h ~/.config/kairo/

# Check directory permissions
ls -la ~/.config/kairo/
```

## Claude Execution Issues

### "claude: command not found"

**Cause:** Claude Code not installed

**Solution:** Install Claude Code from <https://claude.com/downloads>

### "Claude execution failed"

**Cause:** Provider configuration or authentication issue

**Solution:**

```bash
# Test provider first
kairo test <provider>

# Switch with verbose output
kairo switch <provider>

# Check provider is default if using query mode
kairo default
```

## Shell Completion

### Completion not working

**Solution:** Generate and source completion script

**Bash:**

```bash
kairo completion bash >> ~/.bashrc
source ~/.bashrc
```

**Zsh:**

```bash
kairo completion zsh > ~/.zsh/completion/_kairo
# Add to ~/.zshrc: fpath+=~/.zsh/completion
autoload -U compinit
compinit
```

**Fish:**

```bash
kairo completion fish > ~/.config/fish/completions/kairo.fish
```

## Performance Issues

### Slow provider switching

**Cause:** Network latency or timeout

**Solution:**

```bash
# Test specific provider
time kairo test <provider>

# Check network connectivity
curl -I <provider-base-url>
```

### High memory usage

**Cause:** Memory leak or large configuration

**Solution:**

```bash
# Profile memory usage
go tool pprof -http=:8080 http://localhost:6060 heap

# Reset configuration if corrupted
kairo reset all
kairo setup
```

## Log Collection

For bug reports, collect the following:

```bash
# Version info
kairo version

# Configuration (without secrets)
cat ~/.config/kairo/config

# Provider status
kairo status

# List providers
kairo list
```

## Getting Help

- GitHub Issues: <https://github.com/dkmnx/kairo/issues>
- Check [User Guide](./user-guide.md)
- Check [Development Guide](./development-guide.md)
