# Kairo Development

## Architecture

Go CLI that wraps Claude/Qwen Code providers with X25519 encryption via age.

**Stack:** Go 1.26+, Cobra, filippo.io/age, YAML, Go testing

```text
main.go              # Bootstrap
cmd/                 # Cobra commands (root, setup, execution, update, version)
internal/
  config/            # YAML loading, caching, migration
  crypto/            # X25519 key gen, age encrypt/decrypt
  errors/            # Typed error hierarchy with context
  providers/         # Provider registry and resolution
  secrets/           # Encrypted secret storage
  ui/                # ANSI-aware terminal output (build-tagged per OS)
  validate/          # API key, URL, model, cross-provider validation
  version/           # Build-time metadata
  wrapper/           # Shell wrapper script generation
tests/integration/   # End-to-end workflow tests
docs/                # Architecture, guides, reference
```

See `internal/README.md` for package contracts and data flow.
See `cmd/README.md` for command structure and CLIContext details.

## Conventions

- Internal packages (`internal/*`) have zero CLI dependencies. Keep them pure Go.
- Injectable function variables for testability in `cmd/` — external calls (exec, HTTP, prompts) are assigned to package-level `var` funcs, overridden in tests. See `cmd/update.go` for the pattern.
- Propagate `context.Context` through all I/O-bound call chains. Check cancellation between sequential operations (file writes, network calls).
- Error wrapping uses the typed `internal/errors` package — `kairoerrors.WrapError(kind, msg, err)` with `.WithContext(key, val)` for diagnostics.
- Build tags for platform-specific code (e.g., `//go:build !windows` in `internal/ui/`).
- Coverage threshold: 70% enforced in CI.

## Constraints

- Keep `migrateConfigFile` and similar context-aware functions decomposed below cyclop=10. The linter catches this, but the pattern is: extract substeps into named helpers (`statOldConfig`, `readAndValidateConfig`, `finalizeMigration`).
- `nlreturn` + `whitespace` linters coexist — do not place blank lines at the start of blocks (whitespace rejects) but do place blank lines before returns in the main function flow (nlreturn requires). Short error-guard returns inside `if` blocks are exempt from nlreturn's blank-line rule.
- Functions with `CheckContext` calls between sequential I/O steps naturally grow complex. Prefer extracting substeps into helpers early.
- Update checks hit the GitHub Releases API (`api.github.com`) — unauthenticated, 60 req/hr/IP limit. Do not add additional unauthenticated GitHub API calls.

## Commands

```bash
just build           # Binary to dist/
just test            # All tests with -race
just test-coverage   # Coverage report
just lint            # gofmt, go vet, golangci-lint
just pre-release     # Format, lint, pre-commit hooks, test
just release         # Create release (GITHUB_TOKEN required)
```

CI runs: `golangci-lint run ./...`, `go test -race -coverprofile=coverage.out ./...`, `go mod tidy` check.

Pre-commit hooks (`.pre-commit-config.yaml`): golangci-lint, go test, go mod tidy check.

## Patterns

- **Adding a new provider:** See `docs/guides/development-guide.md` for the full workflow.
- **Architecture decisions:** See `docs/architecture/adr/` for ADRs (X25519 choice, Cobra selection, age library).
- **Wrapper scripts:** See `docs/architecture/wrapper-scripts.md` for harness execution details.
- **CI workflows:** See `.github/workflows/ci.yml` (lint, test, cross-platform build), `release.yml`, `vulnerability-scan.yml`.
