# Kairo Development

## WHAT

Go CLI wrapper for Claude/Qwen Code API providers with X25519 encryption.

**Tech Stack:** Go 1.26+, Cobra, age (filippo.io/age), YAML, Go testing

**Key Directories:**

- `cmd/` - CLI commands (Cobra) - see `cmd/root.go:1`
- `internal/` - Business logic (config, crypto, providers, ui, errors, validate)
- `docs/` - Architecture, guides, reference documentation

**Entry Points:**

- `main.go:1` - Application bootstrap
- `cmd/root.go:1` - Root command and CLI setup
- `internal/README.md` - Package contracts, data flow

## HOW

```bash
just build              # Binary to dist/
just test               # All tests with race detector
just fuzz               # Fuzzing tests (5s per func)
just test-coverage      # Coverage report (threshold: 70%)
just lint               # gofmt, go vet, golangci-lint
just security           # govulncheck + lint
just pre-release        # Format, lint, pre-commit, test, goreleaser dry-run
```

Fuzz a specific func: `go test -fuzz=FuzzValidateAPIKey -fuzztime=5s ./internal/validate/`

## Code Style

Google Go Style Guide, plus these rules not enforced by linters:

- **Doc comments** on all top-level exported names (staticcheck ST1000 excluded in config)
- **No `Get` prefix** on getter methods
- **Short receiver names** (1-2 letters), consistent per type
- **Indent error flow** -- early returns, no `else` after errors
- **Initialisms** consistent case (`URL`, `ID`, `HTTP`, `API`)

## Architecture

`cmd/` routes via Cobra, delegates to `internal/` with no CLI dependencies. `internal/providers/registry.go` defines 20 built-in providers. `internal/wrapper/wrapper.go` generates temp shell scripts for secure token passing. See `docs/architecture/README.md` for flow diagrams and ADRs in `docs/architecture/adr/`.

## Testing

Table-driven, `t.TempDir()` for filesystem isolation. CI enforces 70% coverage. Fuzzing in `internal/validate/` and `cmd/`. Run `just test` for race-detected full suite.

## Boundaries

- **Internal purity**: `internal/` imports no Cobra or CLI packages
- **Secrets**: API keys in `secrets.age` only, never in `config.yaml`
- **Install scripts**: modify `scripts/install.{sh,ps1}`, run `just pre-commit` to update checksums
- **Warnings = errors**: 25+ linters in `.golangci.yml` -- treat all as errors

## Patterns

- **Add a provider**: register in `internal/providers/registry.go` + `providerOrder`, add key validation in `internal/validate/api_key.go` if needed
- **Releases**: goreleaser (`CGO_ENABLED=0`, `trimpath`, version vars from `internal/version/`)
- **Config migration**: `internal/config/migration.go` handles provider default model updates across versions
