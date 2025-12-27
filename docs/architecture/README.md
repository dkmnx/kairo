# Architecture

## Overview

Kairo is a Go CLI tool for managing Claude Code API providers with encrypted secrets management using age (X25519) encryption.

## System Architecture

```mermaid
flowchart TB
    subgraph User
        CLI[CLI Commands]
        Shell[Shell/Terminal]
    end

    subgraph Application
        Cobra[Cobra Framework]
        Commands[Command Handlers]
        UI[UI Utilities]
    end

    subgraph Business Logic
        Config[Config Manager]
        Crypto[Encryption Service]
        Providers[Provider Registry]
        Validate[Input Validation]
    end

    subgraph Storage
        ConfigFile[config YAML]
        SecretsAge[secrets.age]
        AgeKey[age.key]
    end

    subgraph External
        Claude[Claude Code]
        APIs[Provider APIs]
    end

    Shell --> CLI
    CLI --> Cobra
    Cobra --> Commands
    Commands --> UI
    Commands --> Config
    Commands --> Crypto
    Commands --> Validate
    Config --> ConfigFile
    Crypto --> SecretsAge
    Crypto --> AgeKey
    Config --> Providers
    Commands --> APIs
    Commands --> Claude
```

## Component Interaction

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Config
    participant Crypto
    participant Storage

    User->>CLI: kairo setup
    CLI->>Crypto: EnsureKeyExists()
    Crypto->>Storage: Check age.key
    alt Key doesn't exist
        Crypto->>Storage: Generate X25519 key
    end
    CLI->>Config: LoadConfig()
    Config->>Storage: Read config
    CLI->>User: Prompt for API key
    User->>CLI: Enter key
    CLI->>Crypto: EncryptSecrets(key)
    Crypto->>Storage: Write secrets.age
```

## Directory Structure

```text
kairo/
├── cmd/                 # CLI commands (Cobra)
│   ├── root.go          # Root command
│   ├── setup.go         # Interactive setup
│   ├── config.go        # Provider config
│   ├── list.go          # List providers
│   ├── status.go        # Test all providers
│   ├── test.go          # Test provider
│   ├── switch.go        # Switch & exec
│   ├── default.go       # Default provider
│   ├── reset.go         # Reset config
│   └── version.go       # Version info
├── internal/            # Business logic
│   ├── config/          # Config loading
│   ├── crypto/          # age encryption
│   ├── providers/       # Provider registry
│   ├── validate/        # Input validation
│   └── ui/              # UI utilities
├── pkg/                 # Reusable utilities
│   └── env/             # Environment helpers
├── docs/                # Documentation
├── scripts/             # Install scripts
└── Makefile             # Build targets
```

## Data Flow: Provider Configuration

```mermaid
flowchart LR
    A[User Input] --> B[Validate API Key]
    B --> C[Validate URL]
    C --> D[Encrypt Secrets]
    D --> E[Write secrets.age]
    F[Config Update] --> G[Write config YAML]
    G --> H[Set Permissions 0600]
```

## Security Architecture

```mermaid
flowchart TB
    subgraph Encryption Layer
        X25519[X25519 Key Pair]
        AgeEncrypt[age Encrypt]
        AgeDecrypt[age Decrypt]
    end

    subgraph Key Management
        Generate[Key Generation]
        Store[Secure Storage]
        Backup[User Backup Required]
    end

    subgraph File Permissions
        Perms[0600 Owner Read/Write]
        ConfigFile[config]
        SecretsFile[secrets.age]
        KeyFile[age.key]
    end

    X25519 --> Generate
    Generate --> Store
    Store --> Backup
    AgeEncrypt --> SecretsFile
    AgeDecrypt --> SecretsFile
    Perms --> ConfigFile
    Perms --> SecretsFile
    Perms --> KeyFile
```

## Configuration Schema

```yaml
# ~/.config/kairo/config
default_provider: zai
providers:
  zai:
    name: Z.AI
    base_url: https://api.z.ai/api/anthropic
    model: glm-4.7
    env_vars:
      - ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.5-air
  anthropic:
    name: Native Anthropic
    base_url: ""
    model: ""
```

## Provider Registry

| Provider     | Base URL                     | Model               | API Key Required    |
| ------------ | ---------------------------- | ------------------- | ------------------- |
| anthropic    | -                            | -                   | No                  |
| zai          | api.z.ai/api/anthropic       | glm-4.7             | Yes                 |
| minimax      | api.minimax.io/anthropic     | Minimax-M2.1        | Yes                 |
| kimi         | api.kimi.com/coding          | kimi-for-coding     | Yes                 |
| deepseek     | api.deepseek.com/anthropic   | deepseek-chat       | Yes                 |
| custom       | user-defined                 | user-defined        | Yes                 |
