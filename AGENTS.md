# Kairo Development

## WHAT

**Kairo** is a Go CLI wrapper for Claude/Qwen Code API providers with X25519 encryption and audit logging.

**Tech Stack:** Go 1.25+, Cobra, age (filippo.io/age), YAML, Go testing

**Key Directories:**
- `cmd/` - Cobra CLI commands and entry points (main.go:1, cmd/root.go:1)
- `internal/` - Business logic (audit, config, crypto, providers, ui, errors, validate)
- `pkg/` - Reusable utilities (env for cross-platform config dir)
- `docs/` - User guides, architecture, best practices

**Entry Points:**
- `main.go:1` - Application bootstrap
- `cmd/root.go:1` - Root command and CLI setup

## HOW

**Build:**
```bash
just build          # Binary to dist/
just install        # Install to ~/.local/bin/
```

**Test:**
```bash
just test           # All tests with race detector
just test-coverage  # Coverage report
```

**Lint/Format:**
```bash
just pre-release    # Format, lint, pre-commit hooks, test
just format         # gofmt -w .
just lint           # gofmt, go vet, golangci-lint
```

**Dependencies:**
```bash
just deps           # Install dependencies and tools
just vuln-scan      # govulncheck
```

**CI/CD:**
```bash
just ci-local       # Run GitHub Actions locally with act
```

## Docs

Read these if relevant to your task:
- `docs/architecture/README.md` - System design and wrapper architecture
- `docs/best-practices.md` - Error handling patterns, testing conventions
- `docs/guides/development-guide.md` - Adding commands, testing, CI workflows
- `internal/README.md` - Internal package contracts and separation of concerns
