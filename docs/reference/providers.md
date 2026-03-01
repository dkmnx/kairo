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

# Configure with setup wizard
kairo setup
```

### zai

Z.AI API - General purpose, high capability.

```bash
# Configure with setup wizard
kairo setup

kairo zai "Your query"
```

### minimax

MiniMax API - Fast responses, cost-effective.

```bash
# Configure with setup wizard
kairo setup

kairo minimax "Your query"
```

### kimi

Moonshot AI (Kimi) - Specialized tasks.

```bash
# Configure with setup wizard
kairo setup

kairo kimi "Your query"
```

### deepseek

DeepSeek AI - Cost-effective, batch processing.

```bash
# Configure with setup wizard
kairo setup

kairo deepseek "Your query"
```

### custom

User-defined provider for self-hosted or custom APIs.

```bash
# Configure with setup wizard
kairo setup
# Enter name: my-provider
# Enter base URL: https://api.example.com/v1
# Enter model: my-model
# Enter API key: ...

kairo my-provider "Your query"
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
kairo setup
kairo newprovider "Your query"
```

1. Update documentation

See [Development Guide](../guides/development-guide.md#adding-a-new-provider)
