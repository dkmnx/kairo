# ADR 0004: Dependency Injection Discipline

## Context

The Kairo CLI has side-effecting dependencies (process execution, file I/O, HTTP calls, encryption, update checks) that must be testable without real resources. Early code used two different strategies:

1. **Structural DI** (preferred): Dependencies are interfaces stored on a context struct (`cmd/deps.go`), injected at startup and swappable in tests via `mockProcess`/`mockWrapper`/`mockUpdate`/`mockCrypto`.
2. **Global mutable seam** (retired): Package-level `var WarnFunc` in `internal/secrets` that tests mutated via `func SetWarnFunc(...) func()`, then restored with `defer`.

The global seam approach has drawbacks:

- Test isolation depends on manual state restoration (the doc comment explicitly warned about leaking between tests).
- The mutation pattern is invisible to compilation (unlike interface injection where a missing method causes a build error).
- It creates a dual-warning path: `WarnFunc` printed to stderr during parse AND the returned `Result.Warnings` slice — and only `cmd/setup_config.go` consumed the latter; `cmd/delete.go` relied on the stderr side-effect.

## Decision

All side-effecting dependencies **must** be injected via structural interfaces wired through the `Deps` struct in `cmd/deps.go`. Global-mutable test seams (package-level vars that are swapped in tests) are **disallowed**.

### Rationale

| Criterion           | Structural DI (Deps)                     | Global mutable seam                |
| ------------------- | ---------------------------------------- | ---------------------------------- |
| Compile-time safety | Missing methods cause build failure      | No compile-time safety             |
| Test isolation      | Separate mock per test instance          | Shared mutable state, must restore |
| Overhead            | Requires interface declaration           | Minimal code                       |
| Discoverability     | All deps visible in `Deps` struct        | Scattered across packages          |
| Refactoring safety  | Adding/removing methods found at compile | Adding/removing vars may be missed |

### Exceptions

- Init-time `var` blocks for hardcoded constants (e.g., CIDR prefixes, regexp patterns) are not side-effecting and are exempt.
- Package-level `MustXxx` functions (e.g., `mustParseCIDR`) that panic on programmer error are exempt — they are not test seams.

## Consequences

### Positive

- All test doubles follow a single, discoverable pattern (`cmd/test_helpers.go` exposes `mockProcess`, `mockWrapper`, `mockUpdate`, `mockCrypto`).
- The `testDeps()` constructor provides sensible defaults and an override callback for per-test customization.
- New contributors only need to learn one DI pattern.

### Negative

- Adding a new dependency requires updating both the interface definition and the mock type (one-time cost).
- The `Deps` struct can grow large; balance by grouping related operations into single interfaces (e.g., `ProcessRunner` bundles `LookPath` + `ExecCommandContext` + `ExitProcess`).

## Implementation Details

### Adding a new injectable dependency

1. Define the interface in `cmd/interfaces.go`.
2. Add a production implementation (may be a thin wrapper around an `os`/`exec`/library call).
3. Add a mock type with function fields in `cmd/test_helpers.go` (follow the existing `mockProcess`/`mockWrapper`/`mockUpdate` pattern).
4. Add the interface to the `Deps` struct and to `testDeps()`.
5. Wire it in `NewDeps()`.

### Removal of retired seam

`internal/secrets/secrets.go` once had `var WarnFunc`/`func SetWarnFunc(...) func()` and a side-effect loop inside `ParseWithStats`. These were removed in June 2026. The `Result.Warnings` slice is the sole carrier of parse warnings. Callers that need warning handling consume `Result.Warnings` directly.

## Migration Path

No migration needed for existing code — the only violation (`WarnFunc` in `internal/secrets`) has been removed.

## Status

Accepted — Implemented since v2.10.0

## References

- `cmd/deps.go` — `Deps` struct and `NewDeps()`
- `cmd/interfaces.go` — `ProcessRunner`, `WrapperService`, `UpdateService` interfaces
- `cmd/test_helpers.go` — `mockProcess`, `mockWrapper`, `mockUpdate`, `mockCrypto`, `testDeps()`
- `internal/secrets/secrets.go` — retired `WarnFunc` (removed)
