# Advanced Configuration Guide

Advanced configuration scenarios and best practices for power users.

## Table of Contents

- [Multi-Provider Setup](#multi-provider-setup)
- [Custom Provider Configuration](#custom-provider-configuration)
- [Environment Variable Integration](#environment-variable-integration)
- [Configuration Management](#configuration-management)
- [Security Best Practices](#security-best-practices)
- [Performance Optimization](#performance-optimization)

---

## Multi-Provider Setup

### Scenario: Multiple Providers for Different Use Cases

Configure multiple providers to switch between them based on use case:

```bash
# Setup primary provider (e.g., Z.AI for general tasks)
kairo config zai
# Enter API key: sk-xxx...
# Enter base URL (optional): https://api.zai.com/v1

# Setup secondary provider (e.g., MiniMax for specialized tasks)
kairo config minimax
# Enter API key: xxx...
# Enter base URL: https://api.minimax.chat/v1

# Setup tertiary provider (e.g., DeepSeek for cost optimization)
kairo config deepseek
# Enter API key: sk-xxx...
# Enter base URL: https://api.deepseek.com/v1

# List all configured providers
kairo list
```

**Usage:**

```bash
# Switch between providers
kairo switch zai "Explain quantum computing"
kairo switch minimax "Write Python code"
kairo switch deepseek "Translate to Spanish"

# Set default provider
kairo default zai

# Use default for quick queries
kairo "Summarize this article"
kairo "Debug this function"

# Override default when needed
kairo switch minimax "Create a marketing plan"
```

### Scenario: Development vs Production Providers

Use different API keys for development and production:

```bash
# Development provider (using test API key)
kairo config dev-prod
# Enter name: dev-prod
# Enter API key: sk-test-xxx...

# Production provider (using production API key)
kairo config prod-prod
# Enter name: prod-prod
# Enter API key: sk-prod-xxx...

# Set default to development
kairo default dev-prod

# Use development by default
kairo "Test query"

# Switch to production when ready
kairo switch prod-prod "Production query"
```

### Scenario: Regional Providers

Configure providers for different regions:

```bash
# US-based provider
kairo config us-prod
# Enter base URL: https://us.api.example.com/v1

# EU-based provider (for GDPR compliance)
kairo config eu-prod
# Enter base URL: https://eu.api.example.com/v1

# Asia-based provider (for lower latency)
kairo config asia-prod
# Enter base URL: https://asia.api.example.com/v1

# Switch based on region requirements
kairo switch eu-prod "Process EU user data"
kairo switch asia-prod "Optimize for Asian market"
```

---

## Custom Provider Configuration

### Scenario: Self-Hosted API

Configure Kairo to work with a self-hosted LLM API:

```bash
# Configure custom provider
kairo config custom

# Enter provider name: my-llm
# Enter API key: your-self-hosted-key
# Enter base URL: https://your-domain.com/v1/chat

# Test configuration
kairo test my-llm

# Use the provider
kairo switch my-llm "Your query"
```

**Requirements:**

- API must be compatible with OpenAI chat endpoint format
- URL must use HTTPS (localhost blocked for security)
- API key minimum 8 characters

### Scenario: Proxy Provider

Configure Kairo to work through a proxy API:

```bash
# Configure proxy provider
kairo config custom

# Enter provider name: my-proxy
# Enter API key: proxy-api-key
# Enter base URL: https://proxy-api.example.com/v1
```

### Scenario: Multi-Model Provider

Configure different models within the same provider:

```bash
# Model 1: General purpose
kairo config custom
# Enter name: gpt-4-general
# Enter API key: sk-xxx...
# Enter base URL: https://api.openai.com/v1
# Enter model: gpt-4

# Model 2: Fast response
kairo config custom
# Enter name: gpt-35-turbo
# Enter API key: sk-xxx...
# Enter base URL: https://api.openai.com/v1
# Enter model: gpt-3.5-turbo

# Switch between models
kairo switch gpt-4-general "Complex task"
kairo switch gpt-35-turbo "Quick response"
```

---

## Environment Variable Integration

### Scenario: CI/CD Pipeline Integration

Use environment variables for automated deployments:

```bash
# In CI/CD configuration (e.g., GitHub Actions)
export ZAI_API_KEY="sk-xxx..."
export ANTHROPIC_API_KEY="sk-ant-xxx..."
export MINIMAX_API_KEY="minimax-key..."
export KIMI_API_KEY="kimi-key..."

# Configure providers (keys from environment)
kairo config zai        # Uses $ZAI_API_KEY
kairo config anthropic   # No key needed
kairo config minimax     # Uses $MINIMAX_API_KEY

# Test in CI/CD
kairo status

# Use in automated scripts
kairo switch zai "Generate documentation"
```

**CI/CD Example (GitHub Actions):**

```yaml
name: Test with Kairo
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install Kairo
        run: |
          curl -sSL https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.sh | sh
      - name: Configure Providers
        env:
          ZAI_API_KEY: ${{ secrets.ZAI_API_KEY }}
          MINIMAX_API_KEY: ${{ secrets.MINIMAX_API_KEY }}
        run: |
          kairo config zai
          kairo config minimax
      - name: Test Providers
        run: |
          kairo status
          kairo test zai
      - name: Run Query
        run: |
          kairo switch zai "Hello from CI/CD"
```

### Scenario: Docker Integration

Use Kairo in Docker containers:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o kairo main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/.config/kairo
COPY --from=builder /app/kairo /usr/local/bin/

# Environment variables for API keys
ENV ZAI_API_KEY=""
ENV MINIMAX_API_KEY=""

# Initialize configuration
RUN kairo config zai || true
RUN kairo config minimax || true

CMD ["kairo", "status"]
```

**Usage:**

```bash
docker build -t kairo .
docker run -e ZAI_API_KEY="sk-xxx..." kairo status
```

### Scenario: Kubernetes Integration

Configure Kairo as a Kubernetes sidecar:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app-with-kairo
spec:
  containers:
  - name: app
    image: my-app:latest
  - name: kairo
    image: kairo:latest
    env:
    - name: ZAI_API_KEY
      valueFrom:
        secretKeyRef:
          name: kairo-secrets
          key: zai-api-key
    command: ["/bin/sh", "-c"]
    args:
    - |
      kairo config zai
      kairo status
      sleep infinity
```

---

## Configuration Management

### Scenario: Backup and Restore

Regularly backup your Kairo configuration:

```bash
# Backup configuration directory
tar -czf kairo-backup-$(date +%Y%m%d).tar.gz ~/.config/kairo/

# Encrypt backup (recommended)
gpg -c kairo-backup-$(date +%Y%m%d).tar.gz
# Remove unencrypted backup
rm kairo-backup-$(date +%Y%m%d).tar.gz

# Restore from backup
tar -xzf kairo-backup-20250128.tar.gz -C ~/

# Verify
kairo list
kairo status
```

### Scenario: Configuration Migration

Migrate configuration between machines:

```bash
# On source machine
tar -czf kairo-config.tar.gz ~/.config/kairo/
scp kairo-config.tar.gz user@target-machine:~

# On target machine
tar -xzf kairo-config.tar.gz -C ~/
chmod 700 ~/.config/kairo
chmod 600 ~/.config/kairo/*

# Test configuration
kairo status
```

### Scenario: Team Configuration Sharing

Share provider configuration (not API keys) across team:

```bash
# Export config without secrets
cat ~/.config/kairo/config | grep -v "api_key" > team-config.yaml

# Team member imports config
cp team-config.yaml ~/.config/kairo/config

# Team member adds their own API keys
kairo config zai  # Enters their own key
kairo config minimax  # Enters their own key
```

---

## Security Best Practices

### Scenario: Regular Key Rotation

Implement automated key rotation:

```bash
# Rotate encryption key
kairo rotate

# Verify all providers still work
kairo status

# Test each provider
kairo test zai
kairo test minimax
```

**Automate with cron:**

```bash
# Add to crontab (monthly rotation)
crontab -e

# Add line:
0 0 1 * * /usr/local/bin/kairo rotate && /usr/local/bin/kairo status | mail -s "Kairo rotation complete" user@example.com
```

### Scenario: Multi-Factor Authentication

Use API keys with MFA (where supported):

```bash
# Configure provider with MFA-enabled API key
kairo config zai
# Enter API key with MFA token: sk-xxx-otp...

# Provider handles MFA verification
kairo test zai
```

### Scenario: Audit Configuration

Regularly audit configuration:

```bash
# List all providers
kairo list

# Check configuration (without API keys)
cat ~/.config/kairo/config | grep -v "api_key"

# Verify file permissions
ls -la ~/.config/kairo/

# Test all providers
kairo status

# Check for unused providers
# Manually review list and remove unused
kairo reset <unused-provider>
```

---

## Performance Optimization

### Scenario: Provider Selection Based on Latency

Test and select fastest provider:

```bash
# Test all providers with timing
time kairo test zai
time kairo test minimax
time kairo test deepseek

# Set fastest as default
kairo default zai

# Use others when specific features needed
kairo switch minimax "Specialized task"
```

### Scenario: Load Balancing Across Providers

Create scripts for load balancing:

```bash
#!/bin/bash
# load-balanced-kairo.sh

PROVIDERS=("zai" "minimax" "deepseek")
INDEX=$((RANDOM % ${#PROVIDERS[@]}))
SELECTED=${PROVIDERS[$INDEX]}

kairo switch $SELECTED "$@"
```

**Usage:**

```bash
chmod +x load-balanced-kairo.sh
./load-balanced-kairo.sh "Your query"
```

### Scenario: Caching Responses

Use provider-specific caching:

```bash
# For repetitive queries, use cheaper/faster provider
kairo default minimax  # Lower cost

# For complex tasks, use better provider
kairo switch zai "Complex task"
```

---

## Troubleshooting Advanced Scenarios

### Issue: Provider Works in CLI But Fails in CI/CD

**Diagnosis:**

```bash
# Check environment variables in CI/CD
env | grep API_KEY

# Verify provider configuration
kairo --verbose status
```

**Solution:**

```bash
# Ensure environment variables are set correctly
export ZAI_API_KEY="sk-xxx..."
kairo config zai
```

### Issue: Multiple Configurations Conflict

**Solution:**

```bash
# Use separate config directories for different environments
export XDG_CONFIG_HOME="$HOME/.config/kairo-dev"
kairo setup

export XDG_CONFIG_HOME="$HOME/.config/kairo-prod"
kairo setup
```

### Issue: Configuration Not Synced Across Machines

**Solution:**

```bash
# Use version control for config (excluding secrets)
echo "~/.config/kairo/config" >> .gitignore
echo "~/.config/kairo/secrets.age" >> .gitignore
echo "~/.config/kairo/age.key" >> .gitignore

# Add provider configurations to git
git add ~/.config/kairo/config
git commit -m "Update provider configurations"

# Use separate API keys per machine
```

---

## Related Documentation

- [User Guide](user-guide.md)
- [Error Handling Examples](error-handling-examples.md)
- [Configuration Architecture](../architecture/README.md#configuration)
- [Security Guide](../architecture/README.md#security)

---

## Complex Multi-Provider Scenarios

### Scenario: Provider Pooling for Load Distribution

Configure multiple providers and distribute load across them:

```bash
# Setup provider pool
kairo config provider-1  # Using zai endpoint
kairo config provider-2  # Using minimax endpoint
kairo config provider-3  # Using deepseek endpoint
kairo config provider-4  # Using kimi endpoint

# Create load distribution script
cat > ~/bin/kairo-pool.sh << 'SCRIPT'
#!/bin/bash
PROVIDERS=("provider-1" "provider-2" "provider-3" "provider-4")
INDEX=$((RANDOM % ${#PROVIDERS[@]}))
kairo switch ${PROVIDERS[$INDEX]} "$@"
SCRIPT
chmod +x ~/bin/kairo-pool.sh

# Use the pool
kairo-pool.sh "Your query"
```

### Scenario: Provider Selection Based on Query Type

Route different types of queries to specialized providers:

```bash
# Setup specialized providers
kairo config code-gen      # Optimized for code generation
kairo config creative      # Optimized for creative writing
kairo config analytical    # Optimized for analysis tasks
kairo config translation   # Optimized for translation

# Create smart router
cat > ~/bin/kairo-smart.sh << 'SCRIPT'
#!/bin/bash
QUERY="$1"

if echo "$QUERY" | grep -qiE "(code|function|debug|programming|api)"; then
  kairo switch code-gen "$QUERY"
elif echo "$QUERY" | grep -qiE "(write|create|story|poem|creative)"; then
  kairo switch creative "$QUERY"
elif echo "$QUERY" | grep -qiE "(analyze|data|statistics|report)"; then
  kairo switch analytical "$QUERY"
elif echo "$QUERY" | grep -qiE "(translate|language|english|spanish)"; then
  kairo switch translation "$QUERY"
else
  kairo switch code-gen "$QUERY"  # Default
fi
SCRIPT
chmod +x ~/bin/kairo-smart.sh

# Use smart router
kairo-smart.sh "Debug this Python function"
kairo-smart.sh "Write a poem about spring"
kairo-smart.sh "Translate to Spanish"
```

### Scenario: Cost-Optimized Multi-Tier Strategy

Use different providers based on query complexity and cost:

```bash
# Tier 1: Fast and cheap (simple queries)
kairo config tier1-cheap
# Use provider with lowest cost per token

# Tier 2: Balanced (moderate complexity)
kairo config tier2-balanced
# Use provider with good performance/cost ratio

# Tier 3: Premium (complex queries)
kairo config tier3-premium
# Use best provider regardless of cost

# Create cost-optimized router
cat > ~/bin/kairo-cost-opt.sh << 'SCRIPT'
#!/bin/bash
QUERY="$1"
TOKEN_COUNT=$(echo "$QUERY" | wc -c)

if [ $TOKEN_COUNT -lt 100 ]; then
  # Simple query - use cheapest
  kairo switch tier1-cheap "$QUERY"
elif [ $TOKEN_COUNT -lt 500 ]; then
  # Medium query - use balanced
  kairo switch tier2-balanced "$QUERY"
else
  # Complex query - use best
  kairo switch tier3-premium "$QUERY"
fi
SCRIPT
chmod +x ~/bin/kairo-cost-opt.sh
```

### Scenario: Geographic Multi-Region Deployment

Deploy providers across multiple regions for latency optimization:

```bash
# Configure regional providers
kairo config us-east
# Enter base URL: https://us-east.api.example.com/v1

kairo config us-west
# Enter base URL: https://us-west.api.example.com/v1

kairo config eu-central
# Enter base URL: https://eu-central.api.example.com/v1

kairo config asia-pacific
# Enter base URL: https://asia-pacific.api.example.com/v1

# Detect user location and use nearest provider
cat > ~/bin/kairo-geo.sh << 'SCRIPT'
#!/bin/bash
# Simple geo-detection based on timezone
TIMEZONE=$(timedatectl | grep "Time zone" | awk '{print $3}')

case "$TIMEZONE" in
  America/*)
    kairo switch us-east "$@"
    ;;
  Europe/*)
    kairo switch eu-central "$@"
    ;;
  Asia/*)
    kairo switch asia-pacific "$@"
    ;;
  *)
    kairo switch us-east "$@"  # Default
    ;;
esac
SCRIPT
chmod +x ~/bin/kairo-geo.sh
```

### Scenario: High-Availability Provider Cluster

Configure provider cluster with automatic health monitoring:

```bash
# Setup cluster
kairo config primary-1
kairo config primary-2
kairo config backup-1
kairo config backup-2

# Health check script
cat > ~/bin/kairo-ha.sh << 'SCRIPT'
#!/bin/bash
PROVIDERS=("primary-1" "primary-2" "backup-1" "backup-2")

for provider in "${PROVIDERS[@]}"; do
  if kairo test $provider --quiet --timeout 5; then
    logger "Kairo: Using healthy provider $provider"
    kairo switch $provider "$@"
    exit $?
  fi
done

logger "Kairo: All providers failed"
echo "Error: All providers are unavailable" >&2
exit 1
SCRIPT
chmod +x ~/bin/kairo-ha.sh

# Continuous health monitoring
cat > ~/bin/kairo-monitor.sh << 'SCRIPT'
#!/bin/bash
while true; do
  for provider in primary-1 primary-2 backup-1 backup-2; do
    if ! kairo test $provider --quiet --timeout 5; then
      logger "Kairo Alert: Provider $provider is unhealthy"
      notify-send "Kairo Alert" "Provider $provider is unhealthy"
    fi
  done
  sleep 60
done &
SCRIPT
chmod +x ~/bin/kairo-monitor.sh
```

### Scenario: A/B Testing Providers

Compare provider performance for the same queries:

```bash
# Setup test providers
kairo config test-provider-a
kairo config test-provider-b

# A/B test script
cat > ~/bin/kairo-ab-test.sh << 'SCRIPT'
#!/bin/bash
QUERY="$1"
RESULTS_DIR="$HOME/kairo-ab-results"
mkdir -p "$RESULTS_DIR"

for provider in test-provider-a test-provider-b; do
  OUTPUT="$RESULTS_DIR/${provider}_$(date +%s).txt"
  START=$(date +%s%N)
  kairo switch $provider "$QUERY" | tee "$OUTPUT"
  END=$(date +%s%N)
  DURATION=$(( (END - START) / 1000000 ))
  echo "Response time: ${DURATION}ms" >> "$OUTPUT"
  echo "Provider: $provider" >> "$OUTPUT"
done

echo "A/B test complete. Results saved to $RESULTS_DIR"
SCRIPT
chmod +x ~/bin/kairo-ab-test.sh

# Run A/B test
kairo-ab-test.sh "Explain quantum computing"
```

### Scenario: Environment-Specific Provider Configurations

Use different providers for different environments:

```bash
# Development environment
kairo config dev-provider
# Use cheaper/faster provider for development

# Staging environment
kairo config staging-provider
# Use production-like provider for staging

# Production environment
kairo config prod-provider
# Use most reliable provider for production

# Environment-aware wrapper
cat > ~/bin/kairo-env.sh << 'SCRIPT'
#!/bin/bash
ENV="${KAIRO_ENV:-development}"

case "$ENV" in
  development)
    kairo switch dev-provider "$@"
    ;;
  staging)
    kairo switch staging-provider "$@"
    ;;
  production)
    kairo switch prod-provider "$@"
    ;;
  *)
    echo "Unknown environment: $ENV" >&2
    exit 1
    ;;
esac
SCRIPT
chmod +x ~/bin/kairo-env.sh

# Usage
export KAIRO_ENV=production
kairo-env.sh "Production query"
```

### Scenario: Provider Configuration Validation

Validate multi-provider configurations before deployment:

```bash
#!/bin/bash
# kairo-validate-multi.sh - Validate multi-provider setup

ERRORS=0

# Check each provider
for provider in $(kairo list --format json | jq -r '.[].name'); do
  echo "Validating $provider..."
  
  # Test provider connectivity
  if ! kairo test $provider --quiet; then
    echo "ERROR: $provider connectivity test failed"
    ERRORS=$((ERRORS + 1))
  fi
  
  # Check provider has API key configured
  if ! kairo status --provider $provider | grep -q "API key: configured"; then
    echo "ERROR: $provider missing API key"
    ERRORS=$((ERRORS + 1))
  fi
  
  # Check provider configuration
  if kairo validate --provider $provider 2>&1 | grep -q "ERROR"; then
    echo "ERROR: $provider configuration invalid"
    ERRORS=$((ERRORS + 1))
  fi
done

# Check for environment variable collisions
if kairo validate --cross-provider 2>&1 | grep -q "collision"; then
  echo "ERROR: Environment variable collision detected"
  ERRORS=$((ERRORS + 1))
fi

# Check default provider is set
if [ -z "$(kairo default)" ]; then
  echo "WARNING: No default provider set"
fi

if [ $ERRORS -eq 0 ]; then
  echo "Multi-provider configuration is valid"
  exit 0
else
  echo "Found $ERRORS error(s) in multi-provider configuration"
  exit 1
fi
```

---

## Additional Multi-Provider Resources

- [Best Practices Guide](../best-practices.md) - Enterprise deployment patterns
- [User Guide](user-guide.md) - Basic usage
- [Error Handling Examples](error-handling-examples.md) - Error scenarios
