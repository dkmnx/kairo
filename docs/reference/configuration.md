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
default_harness: claude | qwen | pi | crush
default_models:
  <provider-name>: string
providers:
  <provider-name>:
    name: string
    base_url: string
    model: string
    env_vars:
      - KEY=value
custom_providers:
  <provider-name>:
    name: string
    base_url: string
    model: string
    requires_api_key: true
    api_key_env_var: string
    min_key_length: number
    key_prefix: string
    key_pattern: string
    env_vars:
      - KEY=value
```

Notes:

- `default_harness` is optional. If omitted, Kairo uses `claude`. Valid values: `claude`, `qwen`, `pi`, `crush`.
- `default_models` is optional migration metadata maintained for built-in providers.
- `custom_providers` is optional. Custom provider definitions are validated at startup and merged into the provider registry. Custom entries with the same key as a built-in provider override the built-in definition.

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

## Custom Providers

Define provider definitions directly in `config.yaml` without recompiling Kairo. Custom providers override built-in providers with the same key.

```yaml
custom_providers:
  my-llm:
    name: My LLM
    base_url: https://api.example.com/anthropic
    model: custom-model
    requires_api_key: true
    api_key_env_var: MY_LLM_API_KEY
    min_key_length: 32
    key_prefix: sk-
    env_vars:
      - EXTRA_VAR=value
```

Fields:

| Field              | Required | Default | Description                                       |
| ------------------ | -------- | ------- | ------------------------------------------------- |
| `name`             | Yes      | —       | Display name shown in setup and list commands     |
| `base_url`         | No       | `""`    | Anthropic-compatible endpoint (HTTPS only)        |
| `model`            | No       | `""`    | Default model (user can override during setup)    |
| `requires_api_key` | No       | `true`  | Whether an API key is required                    |
| `api_key_env_var`  | No       | `""`    | Environment variable name for the API key         |
| `min_key_length`   | No       | `20`    | Minimum API key length                            |
| `key_prefix`       | No       | `""`    | Required API key prefix (e.g. `sk-`)              |
| `key_pattern`      | No       | `""`    | Regex pattern the API key must match              |
| `env_vars`         | No       | `[]`    | Extra environment variables passed to the harness |
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

| Variable             | Purpose                           | Default          |
| -------------------- | --------------------------------- | ---------------- |
| `KAIRO_CONFIG_DIR`   | Override config directory path    | Platform default |
| `KAIRO_UPDATE_URL`   | Override update check URL         | GitHub Releases  |
## Built-in Providers

| Provider   | API Key Required | Default Base URL                     | Default Model         |
| ---------- | ---------------- | ------------------------------------ | --------------------- |
| `zai`      | Yes              | `https://api.z.ai/api/anthropic`     | `glm-5.1`             |
| `minimax`  | Yes              | `https://api.minimax.io/anthropic`   | `MiniMax-M2.7`        |
| `kimi`     | Yes              | `https://api.kimi.com/coding/`       | `kimi-for-coding`     |
| `deepseek` | Yes              | `https://api.deepseek.com/anthropic` | `deepseek-v4-pro[1m]` |
| `custom`   | Yes              | user-defined                         | user-defined          |
## Custom Provider

Required fields when using `kairo setup`:

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
