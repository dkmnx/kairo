# Configuration Reference

Configuration file formats and options for Kairo.

## Config Directory

| OS          | Location                                 |
| ----------- | ---------------------------------------- |
| Linux/macOS | `~/.config/kairo/`                       |
| Windows     | `%USERPROFILE%\AppData\Roaming\kairo\`   |

Kairo can also read configuration from a custom directory via the `--config` CLI flag.

## Files

| File          | Purpose                        | Permissions |
| ------------- | ------------------------------ | ----------- |
| `config.yaml` | Provider and harness settings  | `0600`      |
| `secrets.age` | Encrypted API keys             | `0600`      |
| `age.key`     | Encryption private key         | `0600`      |

## `config.yaml`

Provider, model, and harness configuration in YAML format.

### Schema

```yaml
default_provider: string
default_harness: claude | qwen
default_models:
  <provider-name>: string
providers:
  <provider-name>:
    name: string
    base_url: string
    model: string
    env_vars:
      - KEY=value
```

Notes:

- `default_harness` is optional. If omitted, Kairo uses `claude`.
- `default_models` is optional migration metadata maintained for built-in providers.
- API keys are not stored in `config.yaml`.

### Example

```yaml
default_provider: zai
default_harness: claude
default_models:
  zai: glm-5.1
  minimax: MiniMax-M2.7
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-5.1
    env_vars:
      - ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash
  minimax:
    name: MiniMax
    base_url: https://api.minimax.io/anthropic
    model: MiniMax-M2.7
    env_vars:
      - ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=120
      - ANTHROPIC_SMALL_FAST_MAX_TOKENS=24576
```

## `secrets.age`

Encrypted API keys using age/X25519.

Before encryption, Kairo stores secrets as newline-delimited `KEY=value` entries, for example:

```text
ZAI_API_KEY=...
MINIMAX_API_KEY=...
```

The file on disk is the age-encrypted form of that content.

## `age.key`

X25519 private key in age format.

Generated on first setup. The file contains the private identity line followed by the public recipient line.

## Environment Variables

| Variable           | Purpose                    | Default          |
| ------------------ | -------------------------- | ---------------- |
| `KAIRO_UPDATE_URL` | Override update check URL  | GitHub Releases  |

## Built-in Providers

| Provider   | API Key Required | Default Base URL                     | Default Model         |
| ---------- | ---------------- | ------------------------------------ | --------------------- |
| `zai`      | Yes              | `https://api.z.ai/api/anthropic`     | `glm-5.1`             |
| `minimax`  | Yes              | `https://api.minimax.io/anthropic`   | `MiniMax-M2.7`        |
| `kimi`     | Yes              | `https://api.kimi.com/coding/`       | `kimi-for-coding`     |
| `deepseek` | Yes              | `https://api.deepseek.com/anthropic` | `deepseek-v4-pro[1m]` |
| `custom`   | Yes              | user-defined                         | user-defined          |

## Custom Provider

Required fields:

- `base_url`: HTTPS endpoint
- `model`: Required for custom providers

Optional fields:

- `env_vars`: Array of `KEY=value` strings

Validation rules:

- API key: minimum 20 characters
- Base URL: must use HTTPS and cannot target localhost/private IP ranges
- Model: maximum 100 characters
- Endpoint compatibility: should be Anthropic-compatible

See [Provider Reference](providers.md) for details.
