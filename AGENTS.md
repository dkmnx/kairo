# Kairo Development

## WHAT

Go CLI wrapper for Claude/Qwen Code API providers with X25519 encryption and audit logging.

**Tech Stack:** Go 1.25+, Cobra, age (filippo.io/age), YAML, Go testing

**Key Directories:**

- `cmd/` - CLI commands (Cobra) - see `cmd/root.go:1`
- `internal/` - Business logic (audit, config, crypto, providers, ui, errors, validate)
- `pkg/` - Reusable utilities (cross-platform config dir)
- `docs/` - Architecture, guides, best practices

**Entry Points:**

- `main.go:1` - Application bootstrap
- `cmd/root.go:1` - Root command and CLI setup
- `internal/README.md` - Package contracts, data flow

## HOW

```bash
just build          # Binary to dist/
just test           # All tests with race detector
just test-coverage  # Coverage report
just lint           # gofmt, go vet, golangci-lint
just pre-release    # Format, lint, pre-commit hooks, test
just release        # Create release (requires GITHUB_TOKEN)
```

## Docs

Read these if relevant to your task:

- `docs/architecture/README.md` - System design
- `docs/best-practices.md` - Error handling, testing conventions
- `docs/guides/development-guide.md` - Adding commands, CI workflows

## Notes

- Internal packages have no CLI dependencies - keep them pure
- Use `just --list` to see all available commands
- Run `just pre-release` before committing
