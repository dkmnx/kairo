# Utility Package (`pkg/`)

Reusable utilities with minimal dependencies.

## `env/`

Cross-platform configuration directory resolution.

| Function         | Purpose                                   |
| ---------------- | ----------------------------------------- |
| `GetConfigDir()` | Get platform-appropriate config directory |

### Supported Platforms

| OS      | Path                                          |
| ------- | --------------------------------------------- |
| Linux   | `$XDG_CONFIG_HOME/kairo` or `~/.config/kairo` |
| macOS   | `~/Library/Application Support/kairo`         |
| Windows | `%APPDATA%\kairo`                             |

## Setup

```bash
# No additional setup required
cd /path/to/kairo
go build ./pkg/...
```

## Testing

```bash
go test ./pkg/...
```

## Usage

```go
import "github.com/dkmnx/kairo/pkg/env"

configDir, err := env.GetConfigDir()
if err != nil {
    // Handle error
}
// Use configDir for config operations
```

## File Permissions

All sensitive files use 0600 permissions:

- `config` - Provider configurations (YAML)
- `secrets.age` - Encrypted API keys
- `age.key` - Encryption private key
