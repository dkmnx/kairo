# Contributing Guide

Guidelines for contributing to Kairo.

## Getting Started

### Prerequisites

- Go 1.25+
- Git
- Make (optional)

### Development Setup

```bash
# Fork repository on GitHub
# Clone your fork
git clone https://github.com/YOUR-USERNAME/kairo.git
cd kairo

# Add upstream remote
git remote add upstream https://github.com/dkmnx/kairo.git

# Create feature branch
git checkout -b feature/your-feature-name

# Install dependencies
go mod download

# Build and test
make build
make test
```

## Contribution Types

### Bug Fixes

1. Search existing issues to confirm the bug
2. Create issue with reproduction steps
3. Implement fix
4. Add/update tests
5. Submit PR with fix

### Features

1. Discuss new feature in GitHub Issues
2. Get approval before implementing
3. Implement with tests
4. Update documentation
5. Submit PR

### Documentation

1. Identify documentation gaps
2. Create/update docs
3. Verify docs build/render correctly
4. Submit PR

### Refactoring

1. Ensure tests still pass
2. Maintain backward compatibility
3. Update documentation if APIs change
4. Submit PR with clear explanation

## Code Style

### Go Conventions

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Add godoc comments for exported functions
- Keep functions focused (single responsibility)
- Return typed errors from internal packages

### Naming

- **Packages:** Short, lowercase, singular
- **Functions:** PascalCase for exported, camelCase for unexported
- **Variables:** camelCase
- **Constants:** SCREAMING_SCASE for exported, camelCase for unexported
- **Files:** lowercase with underscores (snake_case)

### Comments

```go
// Package crypto provides age encryption for secrets management.
//
// Functions:
//   - EncryptSecrets: Encrypt API keys to file
//   - DecryptSecrets: Decrypt secrets file
//   - GenerateKey: Create new X25519 key pair
package crypto

// GenerateKey generates a new X25519 encryption key and saves it to the specified path.
func GenerateKey(keyPath string) error {
```

### Error Handling

```go
// Good: Wrap errors with context
return fmt.Errorf("failed to encrypt secrets: %w", err)

// Bad: Generic errors
return err
```

## Testing

### Requirements

- All new code must have tests
- Maintain 80%+ test coverage for internal packages
- Use table-driven tests for validation rules
- Use `t.TempDir()` for test isolation

### Test Structure

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"test case 1", "input", "expected", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### Running Tests

```bash
# All tests with race detection
make test

# Specific package
go test -race ./cmd/...

# With coverage
make test-coverage
```

## Pull Request Process

### Before Submitting

1. **Lint:** `make lint`
2. **Test:** `make test`
3. **Build:** `make build`
4. **Format:** `make format`
5. **Vulnerability scan:** `make vuln-scan`
6. **Update docs:** Ensure documentation is current
7. **Clean commits:** Squash unnecessary commits

### PR Description

```markdown
## Summary

Brief description of changes

## Type of Change

- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change (fix or feature)
- [ ] Documentation update
- [ ] Refactoring

## Testing

Describe how changes were tested

## Checklist

- [ ] My code follows style guidelines
- [ ] I have performed self-review
- [ ] I have commented complex code
- [ ] I have updated documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix works
- [ ] New and existing tests pass locally
```

### Commit Messages

```text
type(scope): subject

body

footer
```

**Types:** feat, fix, docs, style, refactor, test, chore

**Example:**

```text
feat(providers): add new DeepSeek provider

Add DeepSeek AI as a new built-in provider with default
configuration for deepseek-chat model.

Closes #42
```

## Code Review

### What Reviewers Look For

- Code correctness
- Test coverage
- Documentation updates
- Security considerations
- Performance impact
- Style consistency

### Review Process

1. CI checks must pass
2. At least one approval required
3. Address all comments
4. Squash commits before merge

## Security

### Vulnerability Scanning

This project uses `govulncheck` to scan for known vulnerabilities in Go dependencies:

```bash
# Run vulnerability scan
make vuln-scan

# Install govulncheck if not present
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Vulnerability scanning is also run automatically in CI on:
- Every push to main/master
- Every pull request
- Weekly schedule (Sundays at 00:00 UTC)

### Sensitive Data

- Never commit API keys, tokens, or passwords
- Use environment variables for secrets
- Review changes for accidental secrets

### Best Practices

- Validate all inputs
- Use parameterized queries
- Follow secure coding standards
- Report security issues privately
- Run `make vuln-scan` before submitting PRs

## Communication

- **Issues:** GitHub Issues for bugs and features
- **Discussions:** GitHub Discussions for questions
- **PRs:** GitHub Pull Requests for changes

## License

By contributing, you agree to license your work under the MIT License.
