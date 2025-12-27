# Development Guide

Setup, testing, and contribution workflow for Kairo.

## Prerequisites

- Go 1.25+
- Git
- make (optional but recommended)

## Setup

### 1. Clone Repository

```bash
git clone https://github.com/dkmnx/kairo.git
cd kairo
```

### 2. Install Dependencies

```bash
go mod download
go mod verify
```

### 3. Verify Build

```bash
make build
./dist/kairo version
```

## Development Commands

| Command              | Purpose                               |
| -------------------- | ------------------------------------- |
| `make build`         | Build binary to `dist/kairo`          |
| `make test`          | Run tests with race detection         |
| `make test-coverage` | Generate coverage report              |
| `make lint`          | Run gofmt, go vet, golangci-lint      |
| `make format`        | Format code with gofmt                |
| `make pre-commit`    | Run pre-commit hooks                  |
| `make install`       | Install to `~/.local/bin`             |
| `make clean`         | Remove build artifacts                |

### Manual Commands

```bash
# Build
go build -o dist/kairo .

# Run tests
go test -race ./...

# Run specific test
go test -v ./cmd/... -run TestSetup

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Lint
gofmt -d .
go vet ./...
golangci-lint run ./...

# Format
gofmt -w .
```

## Project Structure

```text
kairo/
├── cmd/                  # CLI commands (Cobra)
│   └── README.md         # Command documentation
├── internal/             # Business logic
│   ├── config/           # Configuration management
│   ├── crypto/           # age encryption
│   ├── providers/        # Provider registry
│   ├── validate/         # Input validation
│   └── ui/               # UI utilities
├── pkg/                  # Reusable utilities
│   └── env/              # Environment helpers
├── docs/                 # Documentation
│   ├── architecture/     # Architecture diagrams
│   ├── guides/           # User & dev guides
│   ├── troubleshooting/  # Common issues
│   └── contributing/     # Contribution guidelines
├── scripts/              # Install scripts
├── Makefile              # Build targets
└── go.mod                # Module definition
```

## Adding a New Provider

### 1. Define Provider in Registry

Edit `internal/providers/registry.go`:

```go
var BuiltInProviders = map[string]ProviderDefinition{
    // Existing providers...
    "newprovider": {
        Name:           "New Provider",
        BaseURL:        "https://api.newprovider.com/anthropic",
        Model:          "new-model",
        RequiresAPIKey: true,
        EnvVars:        []string{},
    },
}
```

### 2. Add Validation (if needed)

Update `internal/validate/api_key.go` with any provider-specific validation.

### 3. Update Documentation

- Add provider to `docs/architecture/README.md`
- Add provider to `docs/guides/user-guide.md`
- Update `CHANGELOG.md`

### 4. Test the Provider

```bash
go test ./internal/providers/...
kairo config newprovider
kairo test newprovider
```

## Adding Key Rotation Support

The `crypto.RotateKey()` function handles encryption key rotation:

```go
// RotateKey generates a new key and re-encrypts all secrets
func RotateKey(configDir string) error
```

### Implementation Requirements

1. **Decrypt** existing secrets with old key
2. **Generate** new X25519 key pair
3. **Encrypt** secrets with new key
4. **Replace** old key atomically

### Testing Key Rotation

```bash
go test ./internal/crypto/... -run Rotate
go test ./cmd/... -run Rotate
```

See `internal/crypto/crypto_test.go` for test patterns.

## Testing Guidelines

### Writing Tests

- Use table-driven tests for validation rules
- Use `t.TempDir()` for test isolation
- Mock external dependencies (`exec.Command`, `os.Exit`)
- Aim for 80%+ test coverage

### Example Test

```go
func TestProviderValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "valid-key-123", false},
        {"too short", "short", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateAPIKey(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Running Tests

```bash
# All tests
go test -race ./...

# Specific package
go test -race ./cmd/...
go test -race ./internal/...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Code Style

- Follow Go conventions (Effective Go, Go Code Review Comments)
- Use `gofmt` for formatting
- Add godoc comments for exported functions
- Keep functions focused (single responsibility)
- Return typed errors from internal packages

## Pre-commit Hooks

Install pre-commit for local quality gates:

```bash
pip install pre-commit
pre-commit install
```

Available hooks:

- `gofmt` - Code formatting check
- `go vet` - Static analysis
- `go test` - Run tests with race detection
- `go mod tidy` - Verify go.mod/go.sum consistency

## Building Releases

```bash
# Create release build
make release

# Or manually with goreleaser
goreleaser release --rm-dist
```

Release workflow creates:

- Multi-platform binaries (Linux, macOS, Windows)
- Architecture-specific builds (amd64, arm64)
- Checksums and signatures
- Homebrew formula update

## Debugging

### Enable Verbose Logging

```go
// In cmd/root.go, add:
// PersistentFlags for verbose output
```

### Profile Performance

```bash
# CPU profile
go test -cpuprofile=cpu.out ./...

# Memory profile
go test -memprofile=mem.out ./...

# Analyze
go tool pprof cpu.out
```

### Common Issues

**Tests failing with "permission denied":**

```bash
# Ensure test files use t.TempDir()
# Don't hardcode paths
```

**Race conditions:**

```bash
go test -race ./...
# Fix data races with proper synchronization
```

## Dependency Management

```bash
# Add dependency
go get github.com/new/package

# Tidy dependencies
go mod tidy

# Verify checksums
go mod verify
```

## Contributing

See [Contributing Guide](../contributing/README.md) for:

- Pull request workflow
- Code review process
- Style guidelines
- Commit message format
