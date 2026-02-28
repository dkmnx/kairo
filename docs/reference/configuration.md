# Configuration Reference

Configuration file formats and options for Kairo.

## Config Directory

| OS      | Location                               |
| ------- | -------------------------------------- |
| Linux   | `~/.config/kairo/`                     |
| macOS   | `~/Library/Application Support/kairo/` |
| Windows | `%APPDATA%\kairo\`                     |

## Files

| File          | Purpose                 | Permissions |
| ------------- | ----------------------- | ----------- |
| `config.yaml` | Provider configurations | 0600        |
| `secrets.age` | Encrypted API keys      | 0600        |
| `age.key`     | Encryption private key  | 0600        |

## config.yaml

Provider configurations in YAML format.

### Schema

```yaml
default_provider: string
default_harness: claude | qwen
providers:
  <provider-name>:
    name: string
    base_url: string
    model: string
    env_vars:
      - string
    api_key: encrypted
```

### Example

```yaml
default_provider: zai
default_harness: claude
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
    env_vars:
      - ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash
  anthropic:
    name: Native Anthropic
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.5
```

## secrets.age

Encrypted API keys using age (X25519).

Structure: age-encrypted JSON with provider names as keys.

## age.key

X25519 private key in age format.

Generated on first run using `age-keygen`.

## Environment Variables

| Variable                  | Purpose                   | Default          |
| ------------------------- | ------------------------- | ---------------- |
| `KAIRO_CONFIG_DIR`        | Override config directory | Platform default |
| `KAIRO_UPDATE_URL`        | Custom update check URL   | GitHub Releases  |
| `KAIRO_METRICS_ENABLED`   | Enable metrics            | false            |
| `KAIRO_SKIP_UPDATE_CHECK` | Disable update check      | -                |

## Provider Configuration

### Built-in Providers

| Provider  | API Key Required | Default Base URL           |
| --------- | ---------------- | -------------------------- |
| anthropic | No               | -                          |
| zai       | Yes              | api.z.ai/api/anthropic     |
| minimax   | Yes              | api.minimax.io/anthropic   |
| kimi      | Yes              | api.kimi.com/coding        |
| deepseek  | Yes              | api.deepseek.com/anthropic |
| custom    | Yes              | user-defined               |

### Custom Provider

Required fields:

- `base_url`: HTTPS endpoint
- `model`: Model name

Optional fields:

- `env_vars`: Array of `KEY=value` strings
- `api_key`: Encrypted API key

See [Provider Reference](providers.md) for details.
