# `cmd/` package

The `cmd/` package implements the kairo CLI command tree using
[spf13/cobra](https://github.com/spf13/cobra). It is the only package
allowed to import `cmd/`'s siblings plus `spf13/cobra`; all business
logic lives in `internal/`.

## File map

| File                      | Concern                                                                                                                         |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `root.go`                 | Root command, `Execute()`, `loadRootConfig`, `runPiProvider` / `runStandardProvider`                                            |
| `interfaces.go`           | Service interfaces (Process, Wrapper, Update, Crypto)                                                                           |
| `deps.go`                 | Production adapters that satisfy the interfaces                                                                                 |
| `context.go`              | `CLIContext`, `defaultCLIContext`, `WithCLIContext`                                                                             |
| `setup.go`                | Interactive setup wizard entry point                                                                                            |
| `setup_config.go`         | `EnsureConfigDir`, `LoadConfig`, `AddAndSaveProvider`, `LoadSecrets`, `SaveSecrets`, `ResetSecretsFiles`                        |
| `setup_configdir_test.go` | Tests for config-dir resolution                                                                                                 |
| `setup_provider.go`       | `ProviderDefinition`, `ResolveProviderName`, `BuildProviderConfig`                                                              |
| `setup_prompts.go`        | Interactive prompts (`promptForAPIKey`, `promptForBaseURL`, `promptForModel`, `promptForEnvKey`, `promptForProvider`)           |
| `execution.go`            | `ExecutionConfig`, `WrapperCmd`, `buildWrapperCommand`                                                                          |
| `execution_env.go`        | `BuildProviderEnv`, `BuildPiEnvVars`, `BuildBuiltInEnvVars`, env-var merge logic                                                |
| `execution_harness.go`    | `executePi`, `runHarnessExec`, `executeWithAuth`, `executeWithoutAuth`, `lookUpHarnessBinary`, `reportHarnessError`, `handlePi` |
| `execution_error.go`      | `handleConfigError`, `isBinaryOutdatedError`, `promptUpgrade`, `handleSecretsError`                                             |
| `util.go`                 | `requireConfigDir`, `loadConfigOrExit`, `loadConfigOrEmpty`, `mergeEnvVars` (delegates to `internal/envutil`)                   |
| `default.go`              | `kairo default [provider]` command                                                                                              |
| `list.go`                 | `kairo list` command                                                                                                            |
| `delete.go`               | `kairo delete [provider]` command, `deleteProviderSecrets`                                                                      |
| `harness.go`              | `kairo harness get                                                                                                              | set` subcommands, `resolveHarness` |
| `version.go`              | `kairo version`, `checkForUpdates`                                                                                              |
| `update.go`               | `kairo update` command, cosign/checksum verification                                                                            |
| `completion.go`           | `kairo completion` command and shell scripts                                                                                    |
| `test_helpers.go`         | `testCmd`, `testEchoCmd`, `mockProcess`, `mockWrapper`, `mockUpdate`, `testDeps`                                                |
| `deps_test.go`            | `NewDeps` smoke test and interface conformance                                                                                  |
## Lifecycle of `CLIContext`

A single package-level `defaultCLIContext` is the only `*CLIContext` used
in production. It holds:

- The resolved config directory (lazily via `ConfigDirResolver`).
- The verbosity flag.
- A `*config.ConfigCache` keyed by config directory.
- A `context.Context` for the lifetime of the CLI.
- A `*Deps` containing the four service interfaces.

The context is attached to each `cobra.Command` via `cmd.SetContext`
inside `rootCmd.PersistentPreRun`. `CLIContextFromCmd` retrieves it.

Tests should not rely on the global; use `NewCLIContext()` to construct
isolated contexts and inject them with `cliCtx.SetDeps(...)`.

## Dependency injection

External boundaries (process exec, wrapper script gen, self-update,
crypto) are behind four interfaces in `interfaces.go`. Production uses
`NewDeps()` to wire the real implementations. Tests use `testDeps` to
provide `*mockProcess` / `*mockWrapper` / `*mockUpdate` that record calls
or return canned values. The pattern lets us unit-test the full CLI
flow without spawning real processes.

## Conventions

- Run methods on Cobra commands `cmd *cobra.Command` and accept
  `*cobra.Command` for output. The `ExecutionConfig` struct aggregates
  everything a downstream `execute*` function needs.
- For test isolation, every test file uses `t.TempDir()` and
  `t.Cleanup(...)` to restore the global `defaultCLIContext`.
- Doc comments on every exported identifier (Google Go Style Guide).
- The `cmd/` package imports no `internal/cmd` (there is none) and is
  the only place `spf13/cobra` is imported.
