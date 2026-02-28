# Contributing Guide

Guidelines for contributing to Kairo.

## Getting Started

1. Fork repository
2. Clone: `git clone https://github.com/YOUR-USERNAME/kairo.git`
3. Add upstream: `git remote add upstream https://github.com/dkmnx/kairo.git`
4. Create branch: `git checkout -b feature/your-feature`

## Setup

```bash
go mod download
just build
just test
```

## Before Submitting

```bash
just pre-release  # Format, lint, test
just lint
```

## Pull Request

### Format

```markdown
## Summary
Brief description

## Type
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation

## Testing
How tested

## Checklist
- [ ] Code follows style
- [ ] Tests pass
- [ ] Docs updated
```

### Commit Messages

```
type(scope): description

body
```

Types: feat, fix, docs, style, refactor, test, chore

Example:

```
feat(providers): add DeepSeek provider

Add DeepSeek AI as built-in provider.
Closes #42
```

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt`
- Add godoc comments
- Return typed errors

## Testing

```bash
go test -race ./...
go test -coverprofile=coverage.out ./...
```

Requirements:

- New code needs tests
- Use table-driven tests
- Use `t.TempDir()` for isolation

## License

MIT License - your contributions are licensed under MIT.
