# Audit Guide

Track and monitor all configuration changes with Kairo's audit logging feature.

## Overview

The audit log records all configuration changes including:

- Provider configuration (api_key, base_url, model changes)
- Default provider changes
- Provider resets
- Encryption key rotations
- Provider setup operations
- Provider switches

**File Location:** `~/.config/kairo/audit.log`

## Commands

### List Audit Entries

View human-readable audit history:

```bash
kairo audit list
```

**Output:**

```text
  [2026-01-02 12:00:00] Configured custom (update) -
    api_key: sk-an********mnop,
    base_url: https://old.com -> https://new.com
  [2026-01-02 11:30:00] Switched to zai
  [2026-01-02 10:15:00] Rotated encryption key
  [2026-01-02 09:00:00] Set default provider to zai
```

### Export Audit Log

Export to CSV or JSON format:

**CSV Export:**

```bash
kairo audit export -o audit.csv
```

**CSV Output:**

```csv
timestamp,event,provider,action,changes
2026-01-02T12:00:00Z,config,custom,update,
  "api_key: sk-an********mnop,
   base_url: https://old.com -> https://new.com,
   model: old-model -> new-model"
2026-01-02T11:30:00Z,switch,zai,,
2026-01-02T10:15:00Z,rotate,all,,
```

**JSON Export:**

```bash
kairo audit export -o audit.json -f json
```

**JSON Output:**

```json
[
  {
    "timestamp": "2026-01-02T12:00:00Z",
    "event": "config",
    "provider": "custom",
    "action": "update",
    "changes": [
      {"field": "api_key", "new": "sk-an********mnop"},
      {"field": "base_url", "old": "https://old.com",
       "new": "https://new.com"},
      {"field": "model", "old": "old-model", "new": "new-model"}
    ]
  }
]
```

## Audit Entry Format

### Fields

| Field     | Description                                           |
|-----------|-------------------------------------------------------|
| timestamp | ISO 8601 formatted time                               |
| event     | Event type (config, switch, default, reset, rotate)   |
| provider  | Provider name                                         |
| action    | Action type (add, update, delete)                     |
| changes   | Field-level changes                                   |

### Event Types

| Event   | Description                    | Triggers                   |
|---------|--------------------------------|----------------------------|
| config  | Provider configuration changed | kairo config \<provider\>  |
| switch  | Provider switched              | kairo switch \<provider\>  |
| default | Default provider changed       | kairo default \<provider\> |
| reset   | Provider removed               | kairo reset \<provider\>   |
| rotate  | Encryption key rotated         | kairo rotate               |
| setup   | Provider setup completed       | kairo setup                |

### Change Tracking

API keys are masked in audit logs for security:

| Original Key                    | Logged As           |
|---------------------------------|---------------------|
| sk-ant-api03-abcdefghijklmnop   | sk-an********mnop   |
| short                           | *** (too short)     |

## Use Cases

### Security Auditing

Monitor who made configuration changes:

```bash
# View recent changes
kairo audit list | head -20

# Export for analysis
kairo audit export -o audit-$(date +%Y%m%d).csv
```

### Compliance

Track configuration changes for compliance requirements:

```bash
# Filter by provider
kairo audit list | grep zai

# Filter by event type
kairo audit list | grep config
```

### Troubleshooting

Identify when configuration was changed:

```bash
# View all config changes for a provider
kairo audit list | grep "custom.*config"

# Export and analyze
kairo audit export -o audit.json
jq '.[] | select(.provider == "zai")' audit.json
```

## Security Considerations

- Audit log uses 0600 permissions
- API keys are always masked (first 5 + last 4 characters)
- No plaintext secrets in audit entries
- Audit log is append-only

## Configuration

The audit log location is managed by the config directory:

```bash
# View audit log location
ls -la ~/.config/kairo/audit.log

# Check audit log size
wc -l ~/.config/kairo/audit.log
```

## Related Commands

| Command                    | Description                       |
|----------------------------|-----------------------------------|
| kairo config \<provider\>  | Configure provider (logs changes) |
| kairo default \<provider\> | Set default provider              |
| kairo reset \<provider\>   | Remove provider                   |
| kairo rotate               | Rotate encryption key             |
| kairo switch \<provider\>  | Switch provider                   |

## See Also

- [Architecture - Audit Logging](../architecture/README.md#audit-logging)
- [User Guide](user-guide.md)
- [Development Guide](development-guide.md)
