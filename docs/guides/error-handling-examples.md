# Error Handling Examples

This guide covers common errors you may encounter when using Kairo and how to handle them effectively.

## Table of Contents

- [Configuration Errors](#configuration-errors)
- [Encryption Errors](#encryption-errors)
- [Provider Errors](#provider-errors)
- [Network Errors](#network-errors)
- [Validation Errors](#validation-errors)
- [File System Errors](#file-system-errors)

---

## Configuration Errors

### Error: "No providers configured"

**When it occurs:** First-time use or after resetting all providers

**Solution:**

```bash
# Run interactive setup wizard
kairo setup

# Or configure a specific provider
kairo config zai
```

**What happens:**

1. Checks `~/.config/kairo/config` for provider configurations
2. If file doesn't exist or is empty, prompts to run setup
3. Setup wizard guides through provider selection and API key entry

### Error: "No default provider set"

**When it occurs:** Using query mode without setting a default provider

**Solution:**

```bash
# Option 1: Set a default provider
kairo default zai

# Option 2: Switch explicitly to a provider
kairo switch zai "your query"

# Option 3: Use full command with provider
kairo switch zai
```

### Error: "configuration file not found"

**When it occurs:** Config directory doesn't exist or file is missing

**Solution:**

```bash
# Check if config directory exists
ls -la ~/.config/kairo/

# If missing, create directory and run setup
mkdir -p ~/.config/kairo
kairo setup

# If directory exists but config is missing
kairo config <provider>
```

### Error: "Error loading config: parse error"

**When it occurs:** YAML syntax error in config file

**Solution:**

```bash
# View current config (sanitized)
cat ~/.config/kairo/config

# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('~/.config/kairo/config'))"

# Reset and reconfigure if corrupted
kairo reset all
kairo setup
```

---

## Encryption Errors

### Error: "failed to decrypt: bad key"

**When it occurs:** Encryption key mismatch or corruption

**Diagnosis:**

```bash
# Check if age.key exists
ls -la ~/.config/kairo/age.key

# Check file permissions (should be 0600)
ls -la ~/.config/kairo/age.key

# Verify key is readable
cat ~/.config/kairo/age.key
```

#### Solution 1: Restore from backup (if available)

```bash
# Restore age.key from backup
cp ~/.config/kairo/age.key.backup ~/.config/kairo/age.key
chmod 600 ~/.config/kairo/age.key

# Test decryption
kairo test <provider>
```

#### Solution 2: Reset and reconfigure (lose all secrets)

```bash
# Remove all configuration
kairo reset all

# Remove corrupted key
rm ~/.config/kairo/age.key

# Run setup to generate new key and reconfigure
kairo setup
```

### Error: "failed to decrypt: file is encrypted"

**When it occurs:** Using wrong encryption key or secrets file is corrupted

**Diagnosis:**

```bash
# Check if secrets.age exists
ls -la ~/.config/kairo/secrets.age

# Verify file is encrypted (should not contain plaintext)
head -c 100 ~/.config/kairo/secrets.age
# Should show binary/encrypted data
```

**Solution:**

```bash
# If key is correct but secrets are corrupted:
kairo reset all
kairo setup
```

### Error: "failed to encrypt: write error"

**When it occurs:** Disk full or permission denied

**Diagnosis:**

```bash
# Check disk space
df -h ~/.config/kairo/

# Check directory permissions
ls -la ~/.config/kairo/

# Check write permissions
touch ~/.config/kairo/test
rm ~/.config/kairo/test
```

**Solution:**

```bash
# Fix permissions
chmod 700 ~/.config/kairo
chmod 600 ~/.config/kairo/age.key

# Free disk space if needed
# Then reconfigure
kairo config <provider>
```

---

## Provider Errors

### Error: "Provider 'xxx' not configured"

**When it occurs:** Switching to or testing a provider that hasn't been configured

**Solution:**

```bash
# Check which providers are configured
kairo list

# Configure missing provider
kairo config <provider>

# Verify configuration
kairo test <provider>
```

### Error: "invalid API key"

**When it occurs:** API key fails validation (too short, contains whitespace, or format error)

**Diagnosis:**

```bash
# Check API key requirements:
# - Minimum 8 characters
# - No leading/trailing whitespace
# - No spaces in the middle

# Reconfigure with correct key
kairo config <provider>
# Enter correct API key when prompted
```

**Solution:**

```bash
# Verify your API key format
# Z.AI: Typically "sk-..." or similar prefix
# MiniMax: Check provider documentation
# Kimi: Check provider documentation

# Reconfigure with valid key
kairo config <provider>
```

### Error: "unsupported provider"

**When it occurs:** Using incorrect provider name

**Solution:**

```bash
# List all supported providers
kairo list

# Use lowercase provider name
kairo config zai        # NOT "ZAI" or "Zai"
kairo config minimax    # NOT "MiniMax"
kairo config kimi       # NOT "Kimi" or "KIMI"

# For custom providers, use "custom"
kairo config custom
```

### Error: "invalid base URL"

**When it occurs:** Base URL is malformed, uses HTTP instead of HTTPS, or is unreachable

**Diagnosis:**

```bash
# Verify URL format
# Must start with https://
# Must be a valid URL

# Test URL accessibility
curl -I https://api.example.com/v1/chat
```

**Solution:**

```bash
# Reconfigure with correct URL
kairo config custom
# Enter: https://api.example.com/v1/chat

# Common provider base URLs:
# - Z.AI: https://api.zai.com/v1
# - MiniMax: https://api.minimax.chat/v1
# - Kimi: https://api.moonshot.cn/v1
# - DeepSeek: https://api.deepseek.com/v1
```

---

## Network Errors

### Error: "connection refused"

**When it occurs:** Cannot connect to provider API endpoint

**Diagnosis:**

```bash
# Check network connectivity
ping api.example.com

# Test HTTP connection
curl -v https://api.example.com/v1/chat

# Check firewall rules
sudo ufw status  # Linux
# or
sudo pfctl -s rules  # macOS
```

**Solution:**

```bash
# Check if provider is down
kairo status

# Test connectivity to provider
curl -I https://api.zai.com/v1

# If firewall blocking, add exception or use VPN
```

### Error: "timeout"

**When it occurs:** Request takes too long to complete

**Diagnosis:**

```bash
# Measure response time
time curl https://api.example.com/v1/chat

# Check network latency
ping api.example.com

# Check if using proxy affecting connection
env | grep -i proxy
```

**Solution:**

```bash
# Increase timeout (if using custom config)
kairo config custom
# Provider may have slow response time

# Check network connection
# Try switching to a faster provider
kairo list

# Test different providers
kairo test anthropic
kairo test zai
```

### Error: "certificate verify failed"

**When it occurs:** SSL/TLS certificate issue

**Diagnosis:**

```bash
# Test SSL connection
openssl s_client -connect api.example.com:443

# Check system CA certificates
# Linux: /etc/ssl/certs/
# macOS: /Applications/Utilities/Keychain Access.app
```

**Solution:**

```bash
# Update CA certificates (Linux)
sudo apt-get update && sudo apt-get install ca-certificates  # Debian/Ubuntu
sudo yum install ca-certificates  # RHEL/CentOS

# macOS: Update through Software Update

# For development/testing only (NOT recommended for production):
export CURL_CA_BUNDLE=/path/to/custom-ca.crt
```

---

## Validation Errors

### Error: "provider name must start with letter"

**When it occurs:** Custom provider name starts with number or special character

**Solution:**

```bash
# Valid custom provider names:
kairo config custom
# Enter name: myprovider  ✓
# Enter name: MyProvider  ✓
# Enter name: my-provider-1  ✓

# Invalid custom provider names:
# Enter name: 1provider  ✗ (starts with number)
# Enter name: _provider  ✗ (starts with underscore)
# Enter name: -provider  ✗ (starts with hyphen)
# Enter name: my.provider  ✗ (contains dot)
```

**Valid name regex:** `^[a-zA-Z][a-zA-Z0-9_-]*$`

### Error: "URL must use HTTPS"

**When it occurs:** Base URL uses HTTP instead of HTTPS

**Solution:**

```bash
# Invalid:
http://api.example.com/v1/chat  ✗

# Valid:
https://api.example.com/v1/chat  ✓

# Reconfigure with HTTPS
kairo config custom
# Enter correct URL with https://
```

### Error: "URL contains localhost or private IP"

**When it occurs:** Using localhost or private IP address (security restriction)

**Solution:**

```bash
# Blocked URLs:
http://localhost:8080/v1  ✗
http://127.0.0.1:8080/v1  ✗
http://192.168.1.1/v1  ✗
http://10.0.0.1/v1  ✗

# Use public HTTPS URLs:
https://api.example.com/v1  ✓
```

---

## File System Errors

### Error: "permission denied"

**When it occurs:** Cannot read/write config files

**Diagnosis:**

```bash
# Check file permissions
ls -la ~/.config/kairo/

# Check directory permissions
ls -la ~/.config/kairo
```

**Solution:**

```bash
# Set correct permissions
chmod 700 ~/.config/kairo
chmod 600 ~/.config/kairo/config
chmod 600 ~/.config/kairo/secrets.age
chmod 600 ~/.config/kairo/age.key

# Verify permissions
ls -la ~/.config/kairo/
```

### Error: "no space left on device"

**When it occurs:** Disk is full

**Solution:**

```bash
# Check disk space
df -h

# Clean up temporary files
rm -rf /tmp/*

# Clean up package caches (if applicable)
sudo apt-get clean  # Debian/Ubuntu

# Free space, then retry
kairo config <provider>
```

### Error: "directory not found"

**When it occurs:** Config directory doesn't exist

**Solution:**

```bash
# Create config directory
mkdir -p ~/.config/kairo

# Run setup
kairo setup
```

---

## Debug Mode

For detailed error information, use verbose mode:

```bash
# Enable verbose output
kairo --verbose status
kairo --verbose test zai
kairo --verbose config custom
```

Verbose mode shows:

- Configuration loading process
- Encryption/decryption operations
- Network requests and responses
- Validation errors with details

---

## Getting Help

If you encounter an error not covered in this guide:

1. **Enable verbose mode** to get detailed information:

   ```bash
   kairo --verbose <command>
   ```

2. **Check existing issues** on GitHub:

   <https://github.com/dkmnx/kairo/issues>

3. **Collect diagnostic information**:

   ```bash
   kairo version
   kairo status
   cat ~/.config/kairo/config | grep -v "api_key"  # Hide API keys
   ```

4. **Report issue** with:
   - Command that failed
   - Error message (use --verbose)
   - Operating system and version
   - Kairo version (`kairo version`)

---

## Related Documentation

- [Troubleshooting Guide](../troubleshooting/README.md)
- [User Guide](user-guide.md)
- [Configuration](../architecture/README.md#configuration)
