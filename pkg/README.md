# Utility Package (`pkg/`)

Reusable utilities with minimal dependencies.

## Structure

```text
pkg/
└── env/         # Environment and path utilities
```

**See:** [docs/architecture/README.md](../docs/architecture/README.md) for system architecture.

## env/

Cross-platform configuration directory resolution.

| Function                     | Purpose                                   |
| ---------------------------- | ----------------------------------------- |
| `GetConfigDir()`             | Get platform-appropriate config directory |
| `GetConfigDirWithOverride()` | Get config dir with environment override  |

### Supported Platforms

| OS      | Path                                          |
| ------- | --------------------------------------------- |
| Linux   | `$XDG_CONFIG_HOME/kairo` or `~/.config/kairo` |
| macOS   | `~/Library/Application Support/kairo`         |
| Windows | `%APPDATA%\kairo`                             |

### Environment Overrides

```bash
# Override config directory
export KAIRO_CONFIG_DIR=/path/to/config
```

### Usage

```go
import "github.com/dkmnx/kairo/pkg/env"

configDir, err := env.GetConfigDir()
if err != nil {
    // Handle error
}
// Returns: ~/.config/kairo (Linux)
//         ~/Library/Application Support/kairo (macOS)
//         %APPDATA%\kairo (Windows)
```

## Design Principles

1. **Minimal Dependencies** - Only standard library where possible
2. **Cross-Platform** - Works on Linux, macOS, Windows
3. **Testable** - Easy to mock for testing
4. **Reusable** - Not tied to CLI context

## Testing

```bash
# All pkg tests
go test ./pkg/...

# With race detection
go test -race ./pkg/env/...

# With coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -func=coverage.out
```

## File Permissions

All sensitive files use 0600 permissions:

| File          | Purpose                        |
| ------------- | ------------------------------ |
| `config`      | Provider configurations (YAML) |
| `secrets.age` | Encrypted API keys             |
| `age.key`     | Encryption private key         |
| `audit.log`   | Configuration change history   |

**See:** [docs/architecture/README.md#security](../docs/architecture/README.md#security) for security details.

## Adding a New Package

Create a new directory under `pkg/` with:

1. **Package-level documentation** (README.md optional for small packages)
2. **Godoc comments** for all exported functions
3. **Tests** (`*_test.go` files)
4. **Minimal dependencies** - prefer standard library

Example structure:

```text
pkg/newpackage/
├── newpackage.go      # Main implementation
└── newpackage_test.go # Tests
```
