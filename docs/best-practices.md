# Kairo Best Practices Guide

Production deployment and operational best practices for enterprise environments.

## Table of Contents

- [Security](#security)
- [Configuration Management](#configuration-management)
- [High Availability](#high-availability)
- [Monitoring and Observability](#monitoring-and-observability)
- [Disaster Recovery](#disaster-recovery)
- [Performance Optimization](#performance-optimization)
- [Compliance and Governance](#compliance-and-governance)

---

## Security

### API Key Management

**Best Practice:** Never store API keys in plain text or version control.

```bash
# ✅ Good: Use encrypted secrets
kairo config zai
# Enter API key (encrypted with age)

# ❌ Bad: Store in environment files
echo "ZAI_API_KEY=sk-xxx..." > .env  # Never do this
```

**Enterprise Implementation:**

```bash
# Use secret management systems
export ZAI_API_KEY=$(vault read -field=secret_key secret/zai/api)
kairo config zai

# Or use AWS Secrets Manager
export ZAI_API_KEY=$(aws secretsmanager get-secret-value --secret-id zai/api-key --query SecretString --output text)
kairo config zai
```

### Encryption Key Rotation

**Best Practice:** Rotate encryption keys quarterly or after security incidents.

```bash
# Manual rotation
kairo rotate

# Automated rotation (cron)
0 0 1 */3 * * /usr/local/bin/kairo rotate && /usr/local/bin/kairo status | mail -s "Kairo rotation complete" security@company.com

# Verify rotation
kairo audit --after "2025-01-01" | grep "key rotation"
```

### Principle of Least Privilege

**Best Practice:** Use separate API keys for different environments and applications.

```bash
# Development environment
kairo config dev-zai
# Enter dev-specific API key with limited quotas

# Staging environment
kairo config staging-zai
# Enter staging-specific API key

# Production environment
kairo config prod-zai
# Enter production-specific API key with highest quotas
```

### File Permissions

**Best Practice:** Ensure strict file permissions on configuration files.

```bash
# Verify correct permissions
ls -la ~/.config/kairo/
# Expected: 0700 for directory, 0600 for files

# Fix incorrect permissions
chmod 700 ~/.config/kairo/
chmod 600 ~/.config/kairo/config
chmod 600 ~/.config/kairo/secrets.age
chmod 600 ~/.config/kairo/age.key
```

---

## Configuration Management

### Environment Separation

**Best Practice:** Use separate configuration directories for different environments.

```bash
# Development environment
export XDG_CONFIG_HOME="$HOME/.config/kairo-dev"
kairo setup
kairo config zai  # Dev API key

# Staging environment
export XDG_CONFIG_HOME="$HOME/.config/kairo-staging"
kairo setup
kairo config zai  # Staging API key

# Production environment
export XDG_CONFIG_HOME="$HOME/.config/kairo-prod"
kairo setup
kairo config zai  # Production API key
```

**Enterprise Implementation with Profiles:**

```bash
#!/bin/bash
# kairo-env.sh - Environment switcher

case "$1" in
  dev)
    export XDG_CONFIG_HOME="$HOME/.config/kairo-dev"
    export KAIRO_ENV="development"
    ;;
  staging)
    export XDG_CONFIG_HOME="$HOME/.config/kairo-staging"
    export KAIRO_ENV="staging"
    ;;
  prod)
    export XDG_CONFIG_HOME="$HOME/.config/kairo-prod"
    export KAIRO_ENV="production"
    ;;
  *)
    echo "Usage: $0 {dev|staging|prod}"
    exit 1
    ;;
esac

# Verify environment
kairo status
```

### Configuration as Code

**Best Practice:** Version control provider configurations (without API keys).

```yaml
# config/providers.yaml - Version controlled
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
    env_vars:
      - ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: Minimax-M2.1
default_provider: zai
```

```bash
# Apply configuration from file
kairo apply config/providers.yaml
```

### Configuration Validation

**Best Practice:** Validate configuration before deployment.

```bash
# Pre-flight checks
kairo validate
kairo test zai
kairo test minimax
kairo audit --last 24h
```

---

## High Availability

### Multi-Provider Setup

**Best Practice:** Configure multiple providers for redundancy.

```bash
# Primary provider
kairo config zai
kairo default zai

# Backup providers
kairo config minimax
kairo config deepseek
kairo config kimi

# Create health check script
cat > /usr/local/bin/kairo-health.sh << 'EOF'
#!/bin/bash
PROVIDERS=("zai" "minimax" "deepseek" "kimi")
for provider in "${PROVIDERS[@]}"; do
  if kairo test $provider --quiet; then
    echo "$provider: HEALTHY"
  else
    echo "$provider: UNHEALTHY"
    # Send alert
    notify-send "Kairo Alert" "$provider is unhealthy"
  fi
done
EOF
chmod +x /usr/local/bin/kairo-health.sh

# Run health checks every 5 minutes
*/5 * * * * /usr/local/bin/kairo-health.sh >> /var/log/kairo-health.log 2>&1
```

### Automatic Failover

**Best Practice:** Implement automatic failover logic.

```bash
#!/bin/bash
# kairo-smart-switch.sh - Automatic failover

DEFAULT_PROVIDER="zai"
FALLBACK_PROVIDERS=("minimax" "deepseek" "kimi")

# Try default provider first
if kairo test $DEFAULT_PROVIDER --quiet; then
  kairo switch $DEFAULT_PROVIDER "$@"
  exit $?
fi

# Try fallback providers
for provider in "${FALLBACK_PROVIDERS[@]}"; do
  if kairo test $provider --quiet; then
    logger "Kairo: Failing over to $provider"
    kairo switch $provider "$@"
    exit $?
  fi
done

# All providers failed
logger "Kairo: All providers unavailable"
exit 1
```

### Geographic Distribution

**Best Practice:** Deploy providers across multiple regions.

```bash
# US East (primary)
kairo config us-east-zai
# Enter base URL: https://us-east.api.example.com/v1

# US West (backup)
kairo config us-west-zai
# Enter base URL: https://us-west.api.example.com/v1

# EU (GDPR compliance)
kairo config eu-zai
# Enter base URL: https://eu.api.example.com/v1

# Asia (lower latency)
kairo config asia-zai
# Enter base URL: https://asia.api.example.com/v1
```

---

## Monitoring and Observability

### Audit Logging

**Best Practice:** Enable comprehensive audit logging.

```bash
# View recent activity
kairo audit --last 24h

# Export audit logs
kairo audit --format json --output /var/log/kairo/audit.json

# Send to SIEM
kairo audit --format json | jq -r '.[] | @json' | logger -t kairo -p local0.info
```

### Metrics Collection

**Best Practice:** Collect metrics for provider performance.

```bash
#!/bin/bash
# kairo-metrics.sh - Collect performance metrics

while true; do
  for provider in $(kairo list --format json | jq -r '.[].name'); do
    # Measure response time
    START=$(date +%s%N)
    if kairo test $provider --quiet; then
      END=$(date +%s%N)
      DURATION=$(( (END - START) / 1000000 ))
      echo "kairo_provider_response_time{provider=\"$provider\"} $DURATION" | nc -w 1 graphite.example.com 2003
    fi
  done
  sleep 60
done
```

### Alerting

**Best Practice:** Set up proactive alerting.

```bash
#!/bin/bash
# kairo-alert.sh - Alert on critical issues

# Check for failed provider tests
FAILED=$(kairo test --all 2>&1 | grep -c "FAILED")
if [ $FAILED -gt 0 ]; then
  echo "Kairo: $FAILED providers failed tests" | mail -s "Kairo Alert" ops@company.com
fi

# Check for configuration errors
if kairo validate 2>&1 | grep -q "ERROR"; then
  echo "Kairo: Configuration validation failed" | mail -s "Kairo Alert" ops@company.com
fi

# Check for encryption key age (should be rotated quarterly)
KEY_AGE=$(($(date +%s) - $(stat -c %Y ~/.config/kairo/age.key)))
if [ $KEY_AGE -gt 7776000 ]; then  # 90 days in seconds
  echo "Kairo: Encryption key needs rotation" | mail -s "Kairo Alert" security@company.com
fi
```

---

## Disaster Recovery

### Regular Backups

**Best Practice:** Automate regular configuration backups.

```bash
#!/bin/bash
# kairo-backup.sh - Automated backup script

BACKUP_DIR="/backup/kairo"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="kairo-backup-$DATE.tar.gz"

# Create backup
tar -czf "$BACKUP_DIR/$BACKUP_FILE" ~/.config/kairo/

# Encrypt backup
gpg -c --output "$BACKUP_DIR/$BACKUP_FILE.gpg" "$BACKUP_DIR/$BACKUP_FILE"
rm "$BACKUP_DIR/$BACKUP_FILE"

# Upload to cloud storage (example: AWS S3)
aws s3 cp "$BACKUP_DIR/$BACKUP_FILE.gpg" s3://company-backups/kairo/

# Keep last 30 days of backups
find "$BACKUP_DIR" -name "kairo-backup-*.tar.gz.gpg" -mtime +30 -delete

# Log backup completion
logger "Kairo backup completed: $BACKUP_FILE"
```

### Restore Procedure

**Best Practice:** Document and test restore procedures.

```bash
#!/bin/bash
# kairo-restore.sh - Restore from backup

if [ -z "$1" ]; then
  echo "Usage: $0 <backup-file>"
  exit 1
fi

BACKUP_FILE=$1

# Download from cloud storage (if needed)
aws s3 cp "s3://company-backups/kairo/$BACKUP_FILE" /tmp/

# Decrypt backup
gpg -d -o /tmp/kairo-backup.tar.gz "/tmp/$BACKUP_FILE"

# Stop any running kairo processes
pkill -f kairo

# Backup current configuration (just in case)
mv ~/.config/kairo ~/.config/kairo.failed-restore-$(date +%Y%m%d)

# Restore from backup
tar -xzf /tmp/kairo-backup.tar.gz -C ~/

# Set correct permissions
chmod 700 ~/.config/kairo
chmod 600 ~/.config/kairo/*

# Verify restore
kairo status
kairo test --all

# Clean up
rm /tmp/kairo-backup.tar.gz "/tmp/$BACKUP_FILE"

logger "Kairo restore completed: $BACKUP_FILE"
```

### Configuration Drift Detection

**Best Practice:** Monitor for unauthorized configuration changes.

```bash
#!/bin/bash
# kairo-drift.sh - Detect configuration drift

# Store known-good configuration hash
KNOWN_GOOD="/etc/kairo/config.sha256"

# Calculate current configuration hash
CURRENT_HASH=$(sha256sum ~/.config/kairo/config | awk '{print $1}')

# Compare with known-good
if [ -f "$KNOWN_GOOD" ]; then
  STORED_HASH=$(cat "$KNOWN_GOOD")
  if [ "$CURRENT_HASH" != "$STORED_HASH" ]; then
    echo "WARNING: Configuration drift detected!"
    echo "Previous: $STORED_HASH"
    echo "Current: $CURRENT_HASH"
    logger "Kairo: Configuration drift detected"
    # Send alert
    notify-send "Kairo Alert" "Configuration drift detected"
  fi
fi

# Update known-good hash
echo "$CURRENT_HASH" > "$KNOWN_GOOD"
```

---

## Performance Optimization

### Provider Selection Strategy

**Best Practice:** Choose providers based on workload characteristics.

```bash
# Fast/simple tasks: Use cheaper/faster provider
kairo switch minimax "Summarize this text"

# Complex/creative tasks: Use more capable provider
kairo switch zai "Write a comprehensive guide"

# Code generation: Use specialized provider
kairo switch deepseek "Debug this function"

# Cost optimization: Default to cheapest
kairo default minimax
```

### Connection Pooling

**Best Practice:** Reuse connections for multiple requests.

```bash
# Use session mode for multiple related queries
kairo session start
kairo session send "First query"
kairo session send "Follow-up question"
kairo session send "Another question"
kairo session end
```

### Caching Strategy

**Best Practice:** Implement intelligent caching for repeated queries.

```bash
#!/bin/bash
# kairo-cache.sh - Simple caching wrapper

CACHE_DIR="$HOME/.cache/kairo"
CACHE_TTL=3600  # 1 hour

query="$1"
cache_key=$(echo "$query" | md5sum | awk '{print $1}')
cache_file="$CACHE_DIR/$cache_key"

# Check cache
if [ -f "$cache_file" ]; then
  cache_age=$(($(date +%s) - $(stat -c %Y "$cache_file")))
  if [ $cache_age -lt $CACHE_TTL ]; then
    cat "$cache_file"
    exit 0
  fi
fi

# Cache miss or expired - run query
kairo switch zai "$query" | tee "$cache_file"
```

---

## Compliance and Governance

### GDPR Compliance

**Best Practice:** Use EU-based providers for EU user data.

```bash
# EU provider for GDPR compliance
kairo config eu-zai
# Enter base URL: https://eu.api.z.ai/v1

# Create data residency policy
cat > /etc/kairo/gdpr-policy.yaml << 'EOF'
data_residency:
  eu_users:
    provider: eu-zai
    region: eu-central-1
  us_users:
    provider: us-zai
    region: us-east-1
EOF
```

### SOC 2 Compliance

**Best Practice:** Maintain comprehensive audit trails.

```bash
# Export all audit logs for SOC 2 compliance
kairo audit --start 2025-01-01 --end 2025-01-31 --format json > /audit/soc2/2025-01.json

# Verify log integrity
sha256sum /audit/soc2/*.json > /audit/soc2/checksums.txt

# Send to compliance team
scp /audit/soc2/*.json compliance@company.com:/audit/soc2/
```

### Rate Limiting

**Best Practice:** Implement rate limiting to control costs.

```bash
#!/bin/bash
# kairo-rate-limit.sh - Rate limiting wrapper

RATE_LIMIT=100  # requests per hour
RATE_FILE="$HOME/.config/kairo/rate_limit"

# Load current rate
if [ -f "$RATE_FILE" ]; then
  CURRENT_HOUR=$(date +%Y%m%d%H)
  FILE_HOUR=$(head -1 "$RATE_FILE" | awk '{print $1}')
  CURRENT_COUNT=$(tail -1 "$RATE_FILE" | awk '{print $1}')

  if [ "$CURRENT_HOUR" = "$FILE_HOUR" ]; then
    if [ $CURRENT_COUNT -ge $RATE_LIMIT ]; then
      echo "Rate limit exceeded. Please wait until next hour."
      exit 1
    fi
    NEW_COUNT=$((CURRENT_COUNT + 1))
  else
    NEW_COUNT=1
  fi
else
  NEW_COUNT=1
fi

# Update rate limit file
echo "$(date +%Y%m%d%H) $NEW_COUNT" > "$RATE_FILE"

# Execute command
kairo "$@"
```

### Access Control

**Best Practice:** Implement role-based access control.

```bash
#!/bin/bash
# kairo-rbac.sh - Role-based access control

USER=$(whoami)
case "$USER" in
  admin)
    # Full access
    kairo "$@"
    ;;
  developer)
    # Limited to development providers
    if [[ "$1" == "switch" && "$2" =~ ^(zai|minimax)$ ]]; then
      kairo "$@"
    else
      echo "Access denied: Developers can only use zai and minimax providers"
      exit 1
    fi
    ;;
  readonly)
    # Read-only access
    if [[ "$1" =~ ^(list|status|audit|test)$ ]]; then
      kairo "$@"
    else
      echo "Access denied: Read-only users cannot modify configuration"
      exit 1
    fi
    ;;
  *)
    echo "Access denied: Unknown role"
    exit 1
    ;;
esac
```

---

## Related Documentation

- [Troubleshooting Guide](troubleshooting/README.md)
- [Advanced Configuration](guides/advanced-configuration.md)
- [Security Architecture](architecture/README.md#security)
- [Audit Guide](guides/audit-guide.md)
