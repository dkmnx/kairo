# Provider Reference

Built-in and custom provider configurations.

## Built-in Providers

| Provider  | Base URL                   | Model           | API Key |
| --------- | -------------------------- | --------------- | ------- |
| anthropic | -                          | -               | No      |
| zai       | api.z.ai/api/anthropic     | glm-4.7         | Yes     |
| minimax   | api.minimax.io/anthropic   | MiniMax-M2.5    | Yes     |
| kimi      | api.kimi.com/coding        | kimi-for-coding | Yes     |
| deepseek  | api.deepseek.com/anthropic | deepseek-chat   | Yes     |
| custom    | user-defined               | user-defined    | Yes     |

## Provider Details

### anthropic

Native Anthropic API with ANTHROPIC_API_KEY from environment.

```bash
# Set API key
export ANTHROPIC_API_KEY=sk-ant-...

# Configure
kairo config anthropic
```

### zai

Z.AI API - General purpose, high capability.

```bash
kairo config zai
# Enter API key: sk-ant-...

kairo zai "Your query"
```

### minimax

MiniMax API - Fast responses, cost-effective.

```bash
kairo config minimax
# Enter API key...

kairo minimax "Your query"
```

### kimi

Moonshot AI (Kimi) - Specialized tasks.

```bash
kairo config kimi
# Enter API key...

kairo kimi "Your query"
```

### deepseek

DeepSeek AI - Cost-effective, batch processing.

```bash
kairo config deepseek
# Enter API key...

kairo deepseek "Your query"
```

### custom

User-defined provider for self-hosted or custom APIs.

```bash
kairo config custom
# Enter name: my-provider
# Enter base URL: https://api.example.com/v1
# Enter model: my-model
# Enter API key: ...

kairo switch my-provider "Your query"
```

## Custom Provider Requirements

- **Base URL**: Must be HTTPS (localhost/private IPs blocked)
- **API key**: Minimum 8 characters
- **Model**: Optional, max 100 characters
- **Compatibility**: OpenAI chat completion format

## Adding a New Provider

1. Define in `internal/providers/registry.go`:

```go
var BuiltInProviders = map[string]ProviderDefinition{
    "newprovider": {
        Name:        "New Provider",
        BaseURL:     "https://api.newprovider.com/anthropic",
        Model:       "new-model",
        RequiresAPIKey: true,
    },
}
```

1. Test the provider:

```bash
go test ./internal/providers/...
kairo config newprovider
kairo test newprovider
```

1. Update documentation

See [Development Guide](../guides/development-guide.md#adding-a-new-provider)
