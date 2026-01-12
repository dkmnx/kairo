# AGENTS.md

## Project Overview

Kairo is a Go CLI tool for managing Claude Code API providers with encrypted secrets
management using age (X25519) encryption. It provides a secure, interactive interface
for switching between multiple API providers.

## Guardrails

- **No unapproved actions**: Require user confirmation for destructive operations
  (config reset, provider deletion)
- **Protect sensitives**: Never log API keys, encryption keys, or secrets. Use 0600
  permissions for sensitive files
- **Admit limits/redirect**: When encountering unsupported features, guide users to
  appropriate commands or alternatives
- **Suggest enhancements**: For features beyond current scope, propose them with clear
  value propositions
- **Decline harm**: Refuse to implement features that compromise security (e.g.,
  plaintext key storage, disabled encryption)

## Best Practices

- **YAGNI**: Implement features based on actual user requirements, not speculation
- **SOLID**: Single responsibility per package, open/closed for provider registry,
  dependency injection
- **DRY**: Extract duplicated logic into shared functions (e.g., config.ParseSecrets())
- **KISS**: Prefer simple, readable Go code over complex abstractions

## 1-3-1 Framework

For issues and new features:

1. **1 core problem/requirement**: Clearly state what needs to be solved
2. **3 solutions**: Brainstorm alternatives with pros/cons (performance, maintainability,
   security)
3. **1 best recommendation**: Select and justify the optimal approach

## Code Style

- **Imports**: Standard library first, then third-party (blank line between groups)
- **Formatting**: Run `gofmt -w .` before committing (enforced by pre-commit)
- **Naming**: PascalCase for exported, camelCase for unexported, ALL_CAPS for constants
- **YAML tags**: Use snake_case (e.g., `yaml:"default_provider"`)
- **Error handling**: Use `kairoerrors.WrapError()` for wrapping with context, custom
  KairoError types
- **File operations**: Always specify 0600 permissions for sensitive files using
  `os.WriteFile(path, data, 0600)`
- **Tests**: Use `t.TempDir()` for isolation, `t.Cleanup()` for cleanup, table-driven
  tests

### Error Handling Pattern

```go
import kairoerrors "github.com/dkmnx/kairo/internal/errors"

// Wrap with context
return kairoerrors.WrapError(kairoerrors.ConfigError,
    "failed to read configuration", err).
    WithContext("path", configPath).
    WithContext("hint", "check file permissions")

// Create new error
return kairoerrors.NewError(kairoerrors.ValidationError, "invalid provider name")
```

Available error types: ConfigError, CryptoError, ValidationError, ProviderError,
FileSystemError, NetworkError

## Security

- **No secrets**: Never commit API keys, age keys, or encrypted secrets
- **Threat modeling**: Consider XSS, path traversal, file permission escalation
- **Validation**: Enforce HTTPS-only URLs, block localhost/private IPs, API key min 8
  chars
- **File permissions**: Sensitive files must use 0600 permissions

## Documentation

- Add godoc comments for all exported functions
- Clean code with comments only for complex logic
- Update CHANGELOG.md for all user-facing changes

## Test-Driven Development (MANDATORY)

ALL implementation must follow strict TDD:

### Cycle

1. **RED**: Write failing test first (never write production code without test)
2. **GREEN**: Write minimal code to pass (no more, no less)
3. **REFACTOR**: Improve while tests stay green

### Structure

- **Arrange-Act-Assert**: Set up test state, perform action, verify outcomes
- **Coverage**: 100% for critical paths, 90%+ overall
- **Order**: unit -> integration -> e2e, never skip levels

### Rules

- **Forbidden**: Writing code before tests, skipping tests, "I'll test later"
- **Test doubles**: Use mocks/stubs/spies only at boundaries (external APIs, file system,
  os.Exit)
- **Each test**: Independent, fast, readable, maintainable
- **Naming**: `testShould_ExpectedBehavior_When_StateUnderTest`

## Debugging

### Scientific Method

1. Observe the error/symptom
2. Form a hypothesis about the root cause
3. Test hypothesis with minimal reproduction
4. Analyze results and iterate

### Minimal Repro

- Isolate failing code in standalone test
- Remove dependencies on external services when possible
- Use test doubles for non-deterministic components

### 3 Strategic Prints

1. Print input values at function entry
2. Print intermediate state after transformations
3. Print output values before return

### Binary Search (if needed)

For complex failures with multiple interacting components:

- Split test suite in half (first half vs second half)
- Run isolated package tests vs integration tests
- Comment out half of code changes to identify culprit

## CRITICAL: No Shortcuts Rule (Strictly Enforced)

All code submitted will be rigorously reviewed by a separate, independent AI agent
with zero tolerance for incomplete work.

**Explicitly forbidden (immediate rejection + full rewrite):**

- Placeholders, stubs, "TODO", or commented-out pseudocode
- Dummy/simplified implementations or mock data as shortcuts
- Hardcoded values instead of proper configuration/abstraction
- Incomplete functions, classes, or control flows
- Fake APIs, simulated responses, or fallback behaviors
- Any assumption that "this will be fixed later"

**Every single line must be:** production-ready, fully implemented, conceptually
tested, and defensible. Partial submissions waste time - the reviewing agent will
reject them, and you will redo everything from scratch. Submit only complete, correct,
final code.

## Build & Release

- `make build`: Build to dist/kairo with version injection
- `make test`: Run all tests (verbose + race detection)
- `go test -v ./package -run TestName`: Run single test
- `make test-coverage`: Generate HTML coverage report in dist/coverage.html
- `make lint`: Run gofmt, go vet, golangci-lint
- `make format`: Format all Go files with gofmt
- `make pre-commit`: Run pre-commit hooks manually
- `make verify-deps`: Verify dependency checksums (security)
- `make ci-local`: Run GitHub Actions locally with act

Pre-commit hooks run automatically on commit: gofmt, govet, go-test (race), go-mod-tidy

## Project-Specific Guidelines

### Architecture

- `cmd/`: CLI commands using Cobra framework. Keep logic minimal, delegate to internal
  packages
- `internal/`: Business logic, validation, encryption. No CLI dependencies
- `pkg/`: Reusable utilities with minimal dependencies

### Configuration

- File format: YAML with snake_case keys (default_provider, base_url)
- Storage location: `~/.config/kairo/` (use `pkg/env.GetConfigDir()`)
- Permissions: 0600 for config, secrets.age, age.key files

### Provider Names

- Storage keys: Lowercase in config YAML (zai, minimax, custom)
- Environment variables: Uppercase for API keys (ZAI_API_KEY, MINIMAX_API_KEY)
- Custom providers: Must start with letter, alphanumeric with underscores and hyphens
  (regex: `^[a-zA-Z][a-zA-Z0-9_-]*$`)

### Security Checklist

- [ ] Age encryption (X25519) for all API keys
- [ ] 0600 permissions on sensitive files
- [ ] HTTPS-only URL validation
- [ ] Blocked localhost/private IPs
- [ ] No secrets in logs or error messages
- [ ] API key validation (min 8 characters)
- [ ] Custom provider name validation (regex pattern)
