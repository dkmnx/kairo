# AGENTS.md

## Project Overview

Kairo is a Go CLI tool for managing Claude Code API providers with encrypted secrets management using age
(X25519) encryption. It provides a secure, interactive interface for switching between multiple API
providers.

## Guardrails

- **No unapproved actions**: Require user confirmation for destructive operations (config reset, provider deletion)
- **Protect sensitives**: Never log API keys, encryption keys, or secrets. Use 0600 permissions for sensitive files
- **Admit limits/redirect**: When encountering unsupported features, guide users to appropriate commands or alternatives
- **Suggest enhancements**: For features beyond current scope, propose them with clear value propositions
- **Decline harm**: Refuse to implement features that compromise security (e.g., plaintext key storage, disabled encryption)

## Best Practices

### Code Quality

- **YAGNI (You Aren't Gonna Need It)**: Implement features based on actual user requirements, not speculation
- **SOLID**: Single responsibility per package, open/closed for provider registry, dependency injection for testability
- **DRY (Don't Repeat Yourself)**: Extract duplicated logic into shared functions (e.g., config.ParseSecrets())
- **KISS (Keep It Simple, Stupid)**: Prefer simple, readable Go code over complex abstractions

### 1-3-1 Framework

For issues and new features:

- **1 core problem/requirement**: Clearly state what needs to be solved
- **3 solutions**: Brainstorm alternatives with pros/cons (performance, maintainability, security)
- **1 best recommendation**: Select and justify the optimal approach

### Documentation

- Add godoc comments for all exported functions
- Clean code with comments only for complex logic (validation rules, encryption operations)
- Update CHANGELOG.md for all user-facing changes

### Security

- **No secrets**: Never commit API keys, age keys, or encrypted secrets
- **Threat modeling**: Consider XSS, path traversal, file permission escalation when handling user input
- Validation: Enforce HTTPS-only URLs, block localhost/private IPs, validate API key length (min 8 chars)
- File permissions: Sensitive files must use 0600 permissions

### Testing Requirements

- **80%+ test coverage**: Target for all internal packages (config, crypto, validate, providers)
- Comprehensive docs: README for usage, godoc for API reference
- Pinned dependencies: Use specific versions in go.mod, run go mod verify
- Fix all warnings: Use ReAct analysis (logs, code, causes) to resolve build warnings

## Debugging

### Scientific Method

1. Observe the error/symptom
2. Form a hypothesis about the root cause
3. Test hypothesis with minimal reproduction
4. Analyze results and iterate

### Minimal Repro

- Isolate failing code in standalone test
- Remove dependencies on external services when possible
- Use test doubles (mocks, stubs) for non-deterministic components

### 3 Strategic Prints

1. Print input values at function entry
2. Print intermediate state after transformations
3. Print output values before return

### Binary Search (if needed)

For complex failures with multiple interacting components, systematically narrow down:

- Split test suite in half (first half vs second half)
- Run isolated package tests vs integration tests
- Comment out half of code changes to identify culprit

## CRITICAL: No Shortcuts Rule (Strictly Enforced)

All code submitted will be rigorously reviewed by a separate, independent AI agent with zero tolerance for incomplete work.

**Explicitly forbidden (immediate rejection + full rewrite):**

- Placeholders, stubs, "TODO", or commented-out pseudocode
- Dummy/simplified implementations or mock data as shortcuts
- Hardcoded values instead of proper configuration/abstraction
- Incomplete functions, classes, or control flows
- Fake APIs, simulated responses, or fallback behaviors
- Any assumption that "this will be fixed later"

**Every single line must be:**

- Production-ready
- Fully implemented
- Conceptually tested
- Defensible

Partial submissions waste time — reviewing agent will reject them, and you will redo everything from scratch.

**Submit only complete, correct, final code.**

## Project-Specific Guidelines

### Architecture

- `cmd/` package: CLI commands using Cobra framework. Keep logic minimal, delegate to internal packages
- `internal/` packages: Business logic, validation, encryption. No CLI dependencies
- `pkg/` packages: Reusable utilities with minimal dependencies

### Configuration

- File format: YAML with snake_case keys (default_provider, base_url)
- Storage location: `~/.config/kairo/` (use pkg/env.GetConfigDir())
- Permissions: 0600 for config, secrets.age, age.key files

### Provider Names

- Storage keys: Lowercase in config YAML (zai, minimax, custom)
- Environment variables: Uppercase for API keys (ZAI_API_KEY, MINIMAX_API_KEY)
- Custom providers: Must start with letter, alphanumeric/underscore/hyphen only (regex: `^[a-zA-Z][a-zA-Z0-9_-]*$`)

### Error Handling

- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Return typed errors from internal packages when useful for caller checks
- CLI commands: Print errors via cmd.Printf() or ui.PrintError(), return to let Cobra handle exit code

### Testing

- Use `t.TempDir()` for test isolation (creates temp dirs, auto-cleanup)
- Table-driven tests for validation rules and provider registry
- Mock external dependencies (exec.Command, os.Exit) for CLI command tests
- Run `go test -race ./...` to catch data races

### Build & Release

- `make build`: Build to dist/kairo with version injection
- `make test`: Run all tests with verbose output and race detection
- `make lint`: Run gofmt, go vet, golangci-lint
- `make release`: Create cross-platform builds with goreleaser

### Security Checklist

- [ ] Age encryption (X25519) for all API keys
- [ ] 0600 permissions on sensitive files
- [ ] HTTPS-only URL validation
- [ ] Blocked localhost/private IPs
- [ ] No secrets in logs or error messages
- [ ] API key validation (min 8 characters)
- [ ] Custom provider name validation (regex pattern)
