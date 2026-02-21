# Troubleshooting Guide

Common issues and solutions for Kairo.

**Related Guides:**

- [Error Handling Examples](../guides/error-handling-examples.md) - Detailed error scenarios with solutions
- [Advanced Configuration](../guides/advanced-configuration.md) - Complex setup issues
- [User Guide](../guides/user-guide.md) - Basic usage

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
just build
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

**Solutions (in order of preference):**

1. **Restore from backup** (recommended)
   ```bash
   kairo backup restore ~/.config/kairo/backups/kairo_backup_20240101_120000.zip
   ```

2. **Restore from recovery phrase**
   ```bash
   kairo recover restore word1-word2-word3...
   ```

3. **Reset everything** (last resort - loses all configured providers)
   ```bash
   kairo reset all
   kairo setup
   ```

### Generating a Recovery Phrase

If you haven't already, generate a recovery phrase now:
```bash
kairo recover generate
```

Save this phrase securely. It can be used to restore your encryption key if lost.

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

## Advanced Troubleshooting

### Multi-Provider Conflicts

**Issue:** Commands not working as expected with multiple providers

**Diagnosis:**

```bash
# Check current default
kairo default

# Check all configured providers
kairo list

# Test each provider
kairo test zai
kairo test minimax
kairo test deepseek
```

**Solution:**

```bash
# Clear and reset specific provider
kairo reset <problematic-provider>

# Reconfigure
kairo config <provider>

# Verify
kairo test <provider>
```

### Environment Variable Conflicts

**Issue:** Provider using wrong API key from environment

**Diagnosis:**

```bash
# Check environment variables
env | grep API_KEY

# Check which variables Kairo uses
# ZAI_API_KEY, ANTHROPIC_API_KEY, MINIMAX_API_KEY, etc.
```

**Solution:**

```bash
# Unset conflicting variables
unset ZAI_API_KEY
unset MINIMAX_API_KEY

# Reconfigure explicitly
kairo config zai
# Enter correct key
```

### Configuration Corruption After Update

**Issue:** Version upgrade causes configuration errors

**Diagnosis:**

```bash
# Check version
kairo version

# Validate configuration
cat ~/.config/kairo/config

# Check for deprecated fields
```

**Solution:**

```bash
# Backup current config
cp ~/.config/kairo/config ~/.config/kairo/config.backup

# Run setup to regenerate config
kairo setup

# Manually merge if needed
# Compare config.backup with new config
```

### Permission Issues in Containerized Environments

**Issue:** Cannot write config files in Docker/Kubernetes

**Diagnosis:**

```bash
# Check user in container
whoami

# Check config directory ownership
ls -la ~/.config/kairo/

# Check if directory exists
ls -la ~/.config/
```

**Solution:**

```bash
# Docker: Run with proper user
docker run -u $(id -u):$(id -g) -v ~/.config/kairo:/root/.config/kairo kairo

# Kubernetes: Configure proper security context
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

### Encryption Key Backup Failure

**Issue:** Cannot backup or restore age.key

**Diagnosis:**

```bash
# Check if key file exists
ls -la ~/.config/kairo/age.key

# Try to read key
cat ~/.config/kairo/age.key

# Check backup location
ls -la ~/.config/kairo/*.backup
```

**Solution:**

```bash
# Create backup
cp ~/.config/kairo/age.key ~/.config/kairo/age.key.backup
chmod 600 ~/.config/kairo/age.key.backup

# Verify backup
diff ~/.config/kairo/age.key ~/.config/kairo/age.key.backup

# Test restore
cp ~/.config/kairo/age.key.backup ~/.config/kairo/age.key.test
chmod 600 ~/.config/kairo/age.key.test
```

### Provider Connection Intermittent Failures

**Issue:** Provider works sometimes, fails other times

**Diagnosis:**

```bash
# Test multiple times
for i in {1..10}; do
  kairo test <provider>
  sleep 1
done

# Check network stability
ping -c 10 api.example.com

# Check DNS resolution
nslookup api.example.com
```

**Solution:**

```bash
# Use alternative DNS (temporary)
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf

# Or set specific DNS for provider
# Edit /etc/hosts (requires admin)
# 1.2.3.4 api.example.com

# Try different provider
kairo list
```

### Performance Degradation After Many Providers

**Issue:** Slow startup and switching with many providers

**Diagnosis:**

```bash
# Count providers
kairo list | grep -c "Configured"

# Check configuration size
wc -l ~/.config/kairo/config

# Check secrets file size
ls -lh ~/.config/kairo/secrets.age
```

**Solution:**

```bash
# Remove unused providers
kairo reset <unused-provider>

# Or archive old providers
# Export config
cat ~/.config/kairo/config > config_backup.yaml

# Remove old providers
kairo reset old-provider-1 old-provider-2

# Reconfigure if needed
kairo config <provider>
```

## Log Collection

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
