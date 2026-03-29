# Provider Reference

Built-in and custom provider configurations.

## Built-in Providers

| Provider   | Base URL                             | Model             | API Key |
| ---------- | ------------------------------------ | ----------------- | ------- |
| `zai`      | `https://api.z.ai/api/anthropic`     | `glm-5.1`         | Yes     |
| `minimax`  | `https://api.minimax.io/anthropic`   | `MiniMax-M2.7`    | Yes     |
| `deepseek` | `https://api.deepseek.com/anthropic` | `deepseek-chat`   | Yes     |
| `kimi`     | `https://api.kimi.com/coding/`       | `kimi-for-coding` | Yes     |
| `custom`   | user-defined                         | user-defined      | Yes     |

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

1. Define the provider in `internal/providers/registry.go`:

```go
var BuiltInProviders = map[string]ProviderDefinition{
    "newprovider": {
        Name:           "New Provider",
        BaseURL:        "https://api.newprovider.com/anthropic",
        Model:          "new-model",
        RequiresAPIKey: true,
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
