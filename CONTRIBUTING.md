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

```text
type(scope): description

body
```

Types: feat, fix, docs, style, refactor, test, chore

Example:

```text
feat(providers): add DeepSeek provider

Add DeepSeek AI as built-in provider.
Closes #42
```

## Code Style

- Follow [Google Go Style Guide](https://google.github.io/styleguide/go)
- Use `gofmt` for formatting
- Use `golangci-lint` (see `.golangci.yml`)
- MixedCaps naming, no `Get` prefix on getters
- Short receiver names (1-2 letters), consistent per type
- Doc comments on all top-level exported names
- Indent error flow, early returns
- Return typed errors from `internal/` packages

## Testing

```bash
go test -race ./...
go test -coverprofile=coverage.out ./...
```

Requirements:

- New code needs tests
- Use table-driven tests
- Use `t.TempDir()` for isolation

### Coverage

- The `cmd` package must stay above the 70% coverage gate.
- New commands and setup-wizard changes must include test coverage to avoid dropping below the threshold.
- Run `go test -coverprofile=coverage.out -covermode=atomic ./...` and check `go tool cover -func=coverage.out` before submitting.
- `internal/update` (self-update path) and `cmd` (setup wizard) are the highest-value coverage targets.

## License

MIT License - your contributions are licensed under MIT.
