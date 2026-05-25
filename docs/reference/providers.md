# Provider Reference

Built-in and custom provider configurations.

## Built-in Providers

| Provider                 | API Key Env Var        | Default Model         | API Key |
| :----------------------- | :--------------------- | :-------------------- | :------ |
| `zai`                    | `ZAI_API_KEY`          | `glm-5.1`             | Yes     |
| `minimax`                | `MINIMAX_API_KEY`      | `MiniMax-M2.7`        | Yes     |
| `kimi`                   | `KIMI_API_KEY`         | `kimi-for-coding`     | Yes     |
| `deepseek`               | `DEEPSEEK_API_KEY`     | `deepseek-v4-pro[1m]` | Yes     |
| `anthropic`              | `ANTHROPIC_API_KEY`    | (provider-managed)    | Yes     |
| `openai`                 | `OPENAI_API_KEY`       | (provider-managed)    | Yes     |
| `google`                 | `GEMINI_API_KEY`       | (provider-managed)    | Yes     |
| `mistral`                | `MISTRAL_API_KEY`      | (provider-managed)    | Yes     |
| `groq`                   | `GROQ_API_KEY`         | (provider-managed)    | Yes     |
| `cerebras`               | `CEREBRAS_API_KEY`     | (provider-managed)    | Yes     |
| `cloudflare-workers-ai`  | `CLOUDFLARE_API_KEY`   | (provider-managed)    | Yes     |
| `xai`                    | `XAI_API_KEY`          | (provider-managed)    | Yes     |
| `openrouter`             | `OPENROUTER_API_KEY`   | (provider-managed)    | Yes     |
| `vercel-ai-gateway`      | `AI_GATEWAY_API_KEY`   | (provider-managed)    | Yes     |
| `opencode`               | `OPENCODE_API_KEY`     | (provider-managed)    | Yes     |
| `huggingface`            | `HF_TOKEN`             | (provider-managed)    | Yes     |
| `fireworks`              | `FIREWORKS_API_KEY`    | (provider-managed)    | Yes     |
| `azure-openai-responses` | `AZURE_OPENAI_API_KEY` | (provider-managed)    | Yes     |
| `minimax-cn`             | `MINIMAX_CN_API_KEY`   | (provider-managed)    | Yes     |
| `custom`                 | user-defined           | user-defined          | Yes     |
Providers without default base URLs and models (marked "provider-managed") are passed through to the harness CLI directly. The harness manages its own endpoint and model selection for these providers.

## Provider Details

### `zai`

Z.AI API.

```bash
kairo setup
kairo zai "Your query"
```

### `minimax`

MiniMax API.

```bash
kairo setup
kairo minimax "Your query"
```

### `kimi`

Moonshot AI (Kimi).

```bash
kairo setup
kairo kimi "Your query"
```

### `deepseek`

DeepSeek AI.

```bash
kairo setup
kairo deepseek "Your query"
```

### `custom`

User-defined provider for compatible Anthropic-style endpoints.

```bash
kairo setup
# Enter name: my-provider
# Enter base URL: https://api.example.com/anthropic
# Enter model: my-model
# Enter API key: ...

kairo my-provider "Your query"
```

## Custom Provider Requirements

- **Base URL**: Must use HTTPS and cannot target localhost/private IP ranges
- **API key**: Minimum 20 characters
- **Model**: Required, maximum 100 characters
- **Compatibility**: Anthropic-compatible API endpoint

## API Key Validation

- Built-in providers (`zai`, `minimax`, `kimi`, `deepseek`) require keys with a minimum length of 32 characters
- Custom and unknown providers require keys with a minimum length of 20 characters

## Adding a New Provider

### Custom Provider via config.yaml

Define providers in `~/.config/kairo/config.yaml` under `custom_providers`:

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
```

Then run `kairo setup` — the custom provider appears in the dropdown. Custom providers override built-in providers with the same key, letting you patch defaults (e.g., model name) without recompiling.

### Built-in Provider via code

1. Define the provider in `internal/providers/registry.go`:

```go
var builtInProviders = map[string]ProviderDefinition{
    // ... existing entries ...
    "newprovider": {
        Name:           "New Provider",
        BaseURL:        "https://api.newprovider.com/anthropic",
        Model:          "new-model",
        RequiresAPIKey: true,
        APIKeyEnvVar:   "NEWPROVIDER_API_KEY",
        KeyFormat:      KeyFormatMin32,
    },
}
```

1. Add the provider to `providerOrder` in the same file.

1. Test the provider:

```bash
go test ./internal/providers/... ./internal/validate/...
kairo setup
kairo newprovider "Your query"
```

1. Update documentation.

See [Development Guide](../guides/development-guide.md#adding-a-provider)
