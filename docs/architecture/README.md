# Architecture

System architecture and design documentation for Kairo.

## Overview

Kairo is a Go CLI tool for managing Claude Code API providers with:

- **Age (X25519) encryption** for secure API key storage
- **Multi-provider support** for switching between providers
- **Audit logging** for tracking configuration changes

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
        Audit[Audit Logger]
    end

    subgraph Storage
        ConfigFile[config YAML]
        SecretsAge[secrets.age]
        AgeKey[age.key]
        AuditLog[audit.log]
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
    Commands --> Audit
    Config --> ConfigFile
    Crypto --> SecretsAge
    Crypto --> AgeKey
    Audit --> AuditLog
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
│   ├── root.go          # Root command & provider execution
│   ├── setup.go         # Interactive setup & edit
│   ├── list.go          # List providers
│   ├── delete.go        # Delete provider
│   ├── harness.go       # Harness management
│   ├── update.go        # Update CLI
│   ├── version.go       # Version info
│   ├── completion.go     # Shell completion
│   ├── audit_helpers.go # Audit logging helpers
│   └── util.go         # Utility functions
├── internal/            # Business logic
│   ├── audit/           # Audit logging
│   ├── config/          # Config loading & caching
│   ├── crypto/          # age encryption
│   ├── errors/          # Typed errors
│   ├── providers/       # Provider registry
│   ├── ui/              # UI utilities
│   ├── validate/        # Input validation
│   ├── version/         # Version information
│   └── wrapper/        # Secure wrapper scripts
├── pkg/                 # Reusable utilities
│   └── env/             # Environment helpers
├── docs/                # Documentation
│   ├── architecture/    # This directory
│   ├── contributing/    # Contribution guidelines
│   ├── guides/          # User & dev guides
│   ├── reference/       # Reference documentation
│   └── troubleshooting/ # Common issues
├── scripts/             # Install scripts
└── justfile             # Command runner
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
    H --> I[Log Audit Entry]
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
        AuditFile[audit.log]
    end

    X25519 --> Generate
    Generate --> Store
    Store --> Backup
    AgeEncrypt --> SecretsFile
    AgeDecrypt --> SecretsFile
    Perms --> ConfigFile
    Perms --> SecretsFile
    Perms --> KeyFile
    Perms --> AuditFile
```

## Audit Logging

```mermaid
flowchart TB
    subgraph Commands
        SetupCmd[setup]
        DeleteCmd[delete]
        SwitchCmd[provider execution]
    end

    subgraph Audit
        ChangeTracker[Change Tracker]
        LogFormatter[Log Formatter]
        FileWriter[File Writer]
    end

    subgraph Output
        AuditLog[audit.log]
    end

    SetupCmd --> ChangeTracker
    DeleteCmd --> ChangeTracker
    SwitchCmd --> ChangeTracker

    ChangeTracker --> LogFormatter
    LogFormatter --> FileWriter
    FileWriter --> AuditLog
```

## Configuration Schema

```yaml
# ~/.config/kairo/config.yaml
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
```

## Provider Registry

| Provider  | Base URL                   | Model           | API Key Required |
| --------- | -------------------------- | --------------- | ---------------- |
| anthropic | -                          | -               | No               |
| zai       | api.z.ai/api/anthropic     | glm-4.7         | Yes              |
| minimax   | api.minimax.io/anthropic   | MiniMax-M2.5    | Yes              |
| kimi      | api.kimi.com/coding        | kimi-for-coding | Yes              |
| deepseek  | api.deepseek.com/anthropic | deepseek-chat   | Yes              |
| custom    | user-defined               | user-defined    | Yes              |

## Error Handling

```mermaid
flowchart TB
    subgraph Error Types
        ConfigErr[ConfigError]
        CryptoErr[CryptoError]
        ValidErr[ValidationError]
        ProviderErr[ProviderError]
        FileErr[FileSystemError]
        NetErr[NetworkError]
    end

    subgraph Error Handling
        Wrap[Wrap with context]
        Context[Add context data]
        Hint[Provide hints]
    end

    ConfigErr --> Wrap
    CryptoErr --> Wrap
    ValidErr --> Wrap
    Wrap --> Context
    Context --> Hint
```

## Dependencies

### Runtime Dependencies

| Package                  | Purpose           |
| ------------------------ | ----------------- |
| `filippo.io/age`         | X25519 encryption |
| `github.com/spf13/cobra` | CLI framework     |
| `gopkg.in/yaml.v3`       | YAML parsing      |

### Development Dependencies

| Package                         | Purpose            |
| ------------------------------- | ------------------ |
| `github.com/Masterminds/semver` | Version comparison |
| `github.com/stretchr/testify`   | Testing assertions |

## Design Principles

### 1. Security First

- All API keys encrypted at rest
- 0600 permissions on sensitive files
- No plaintext secrets in logs
- HTTPS-only for provider APIs

### 2. User Experience

- Interactive setup wizard
- Clear error messages with hints
- Colored terminal output
- Shell completion support

### 3. Maintainability

- Clean package structure
- Comprehensive test coverage
- Typed error handling
- Documentation-driven design

### 4. Extensibility

- Provider registry pattern
- Configurable via environment
- Exportable audit logs
- Modular architecture

## Cross-Platform Support

```mermaid
flowchart TB
    subgraph Platforms
        Linux[Linux]
        macOS[macOS]
        Windows[Windows]
    end

    subgraph Config Directories
        Linux["~/.config/kairo/"]
        macOS["~/Library/Application Support/kairo/"]
        Windows["%APPDATA%/kairo/"]
    end

    subgraph Install Methods
        Shell["install.sh (curl | sh)"]
        PS["install.ps1 (PowerShell)"]
    end

    subgraph Token Passing
        Unix["Shell wrapper script"]
        Win["Batch script + cmd /c"]
    end

    Linux --> LinuxConfig
    macOS --> macOSConfig
    Windows --> WinConfig

    Linux --> Shell
    macOS --> Shell
    Windows --> PS

    Shell --> UnixWrapper
    PS --> WinWrapper
```

### Platform-Specific Implementations

| Feature          | Linux/macOS                 | Windows                    |
| ---------------- | --------------------------- | -------------------------- |
| Config Directory | `~/.config/kairo/`          | `%APPDATA%\kairo\`         |
| Install Script   | `install.sh` (curl \| sh)   | `install.ps1` (PowerShell) |
| Token Passing    | Shell wrapper (`#!/bin/sh`) | Batch script (`.bat`)      |
| Shell Completion | bash, zsh, fish             | PowerShell                 |

### Key Cross-Platform Code

```go
// Cross-platform config directory (pkg/env/env.go)
if runtime.GOOS == "windows" {
    return filepath.Join(home, "AppData", "Roaming", "kairo")
}
return filepath.Join(home, ".config", "kairo")

// Cross-platform token passing (cmd/switch.go)
if isWindows {
    // Generate .bat file with batch syntax
    scriptContent = "@echo off\r\n"
    // ...
} else {
    // Generate shell script with sh syntax
    scriptContent = "#!/bin/sh\n"
    // ...
}
```

### Testing on Windows

```go
// Skip Unix permission tests on Windows
if runtime.GOOS == "windows" {
    t.Skip("Windows does not support Unix-style permissions")
}
```

See also: [pkg/env](../pkg/README.md), [cmd/switch.go](../../cmd/switch.go)
