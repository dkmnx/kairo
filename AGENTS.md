# Kairo Development

## WHAT

**Kairo** is a Go CLI wrapper for Claude/Qwen Code API providers with X25519 encryption and audit logging.

**Tech Stack:** Go 1.25+, Cobra, age (filippo.io/age), YAML, Go testing

**Key Directories:**
- `cmd/` - Cobra CLI commands and entry points (main.go:1, cmd/root.go:1)
- `internal/` - Business logic (audit, config, crypto, providers, ui, errors, validate)
- `pkg/` - Reusable utilities (env for cross-platform config dir)
- `docs/` - User guides, architecture, best practices
- `tests/` - Integration tests

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
just fuzz           # Fuzzing tests (5s per test)
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
just verify-deps    # Verify dependency checksums
```

**Releases:**
```bash
just release          # Create release (requires GITHUB_TOKEN)
just release-local    # Local snapshot build
just release-dry-run  # Build without publishing
```

**CI/CD:**
```bash
just ci-local       # Run GitHub Actions locally with act
just ci-local-list  # List CI jobs
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `setup` | Interactive setup wizard to configure providers |
| `config <provider>` | Configure a provider with API key, base URL, and model |
| `list` | List all configured providers |
| `switch <provider>` | Switch default provider |
| `reset <provider|all>` | Remove provider configuration |
| `rotate` | Rotate encryption key |
| `backup` | Create backup of config, keys, and secrets |
| `restore <backup>` | Restore from backup file |
| `audit` | View audit log entries |
| `completion <shell>` | Generate shell completion scripts |

## Project Structure

```
kairo/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go            # Root command, entry point
│   ├── setup.go           # Interactive setup wizard
│   ├── config.go          # Provider configuration
│   ├── switch.go          # Provider switching/execution
│   ├── reset.go           # Provider reset
│   ├── audit.go           # Audit logging
│   ├── backup.go          # Backup/restore
│   ├── rotate.go          # Key rotation
│   ├── list.go            # List providers
│   ├── completion.go      # Shell completion
│   └── *_test.go          # Test files
├── internal/               # Business logic (no CLI deps)
│   ├── audit/             # Audit logging
│   ├── config/            # YAML loading/saving
│   ├── crypto/            # Age encryption
│   ├── providers/         # Provider registry
│   ├── validate/          # Input validation
│   ├── ui/                # Terminal UI utilities
│   ├── errors/            # Typed errors
│   ├── recovery/          # Recovery phrase generation
│   └── version/           # Version info
├── pkg/                    # Reusable utilities
│   └── env/               # Cross-platform config dir
├── tests/                  # Integration tests
│   └── integration/       # Full workflow tests
├── docs/                   # Documentation
│   ├── architecture/      # System design
│   ├── guides/            # User guides
│   └── best-practices.md  # Development conventions
├── scripts/                # Installation scripts
├── dist/                   # Build output
├── justfile                # Command runner
├── go.mod/go.sum           # Dependencies
├── .github/workflows/      # CI/CD
└── AGENTS.md               # This file
```

## Docs

Read these if relevant to your task:
- `internal/README.md` - Package contracts, architecture, data flow
- `docs/architecture/README.md` - System design and wrapper architecture
- `docs/best-practices.md` - Error handling patterns, testing conventions
- `docs/guides/development-guide.md` - Adding commands, testing, CI workflows
