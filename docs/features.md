# Features

## Backup & Recovery

### Creating Backups

```bash
# Create a backup (stores in ~/.config/kairo/backups/)
kairo backup

# Backup is timestamped: kairo_backup_20240115_093000.zip
```

### Restoring from Backup

```bash
# List available backups
ls ~/.config/kairo/backups/

# Restore from a specific backup
kairo restore ~/.config/kairo/backups/kairo_backup_20240115_093000.zip
```

### Recovery Phrases

Generate a recovery phrase to restore your encryption key without a backup:

```bash
# Generate a recovery phrase for your current key
kairo recover generate

# Save the phrase securely!

# Restore from phrase (if key is lost)
kairo recover restore word1-word2-word3-word4...
```

### What Gets Backed Up

- `age.key` - Your encryption private key
- `secrets.age` - Your encrypted API keys
- `config.yaml` - Your provider configurations

### What Recovery Phrases Protect

Recovery phrases encode your encryption key using base64 encoding.
Store them securely - anyone with access can restore your key.
