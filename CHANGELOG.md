# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-03-01

### Added

- Direct provider execution: use `kairo <provider> [args]` to switch and run in one command
- Default provider execution: use `kairo -- [args]` or just `kairo` to run with default provider
- Architecture Decision Records (ADRs) documenting key technical decisions
- TUI integration for provider configuration (tap editor)

### Changed

- **[BREAKING]:** Simplified CLI command structure - removed 8 subcommands
- **[BREAKING]:** `kairo config` functionality merged into `kairo setup`
- **[BREAKING]:** `kairo reset` renamed to `kairo delete`
- Banner simplified to text-only format with gray color
- Increased test coverage across cmd, crypto, and wrapper packages

### Removed

- **[BREAKING]:** Audit logging (`kairo audit` command, internal/audit package)
- **[BREAKING]:** Backup and restore (`kairo backup` command, internal/backup package)
- **[BREAKING]:** Metrics tracking (`kairo metrics` command, internal/performance package)
- **[BREAKING]:** Key recovery (`kairo recover` command, internal/recovery, internal/recoveryphrase packages)
- **[BREAKING]:** Key rotation (`kairo rotate` command)
- **[BREAKING]:** Provider status (`kairo status` command)
- **[BREAKING]:** Provider testing (`kairo test` command)
- **[BREAKING]:** Switch command (`kairo switch` - use `kairo <provider>` directly)
- **[BREAKING]:** Standalone completion command (use shell builtin completion)
- **[BREAKING]:** Obsolete documentation files (consolidated into reference structure)
- Internal package: internal/config/cache.go
- Internal package: internal/recovery/recovery.go
- Internal package: internal/recoveryphrase/recoveryphrase.go
- Integration tests for removed features

### Fixed

- Launch default provider when no arguments provided
- Pass harness args correctly when using `--` separator
- Check configureProvider error in setup command
- Remove unused parameters from executeWithAuth and handleSecretsError
- Resolve code review findings

### Refactored

- Reduced function complexity in cmd/root.go
- Centralized error handling and removed unused crypto code
- Use config structs for functions with >2 parameters
- Extract promptForField to eliminate DRY violation
- Split configureProvider into smaller functions
- Improve naming consistency and remove dead code
- Extract root.go Run into smaller helper functions
- Remove config transaction file (cmd/config_tx.go)

## [1.10.1] - 2026-02-28

### Added

- Fuzzing tests for input validation functions

### Changed

- Centralized all error definitions in `internal/errors` package
- Extracted `FormatSecrets` to shared config utility for reuse in reset command
- Removed harness selection from setup command (use `kairo harness set` instead)

### Fixed

- Audit export: fixed format validation and error handling for JSON/CSV exports
- Reset command: use direct config load instead of cache for consistency
- Reset command: renamed `integrityKey` to more accurate `identityKey`
- Recovery phrase: added max length validation and documented memory safety limitation

### Security

- Addressed critical HMAC security issues
- Addressed code review security findings
- Addressed remaining security warnings and suggestions

## [1.10.0] - 2026-02-21

### Added

- `--harness` flag on `switch` command to override harness per invocation (claude or qwen)
- Automatic `--auth-type anthropic` flag passed to Qwen harness
- Suppression of Node.js deprecation warnings in Claude/Qwen child processes
- Rollback mechanism for key rotation to restore old key on re-encryption failure
- Timeout protection for install script downloads in update command
- Warning logging for silently ignored errors in audit log rotation

### Changed

- Model now passed from provider config instead of `--model` flag (removed)
- Extracted duplicated secrets loading to shared `LoadAndDecryptSecrets()` function
- Centralized file name constants in `internal/config/paths.go`
- Replaced `fmt.Errorf` and string matching with typed errors throughout codebase
- Simplified `mergeEnvVars` with filter-based deduplication for stability
- Extracted signal handler, validation, and transaction logic to separate modules
- Use `RLock` for cache reads to reduce contention
- Use bit shifting for O(1) exponential backoff calculation
- Use `exec.LookPath("sh")` for cross-platform shell detection

### Deprecated

- Nothing

### Removed

- `--model` flag from switch command (model now from provider config)
- Unused `LoadSecrets` alias (use `LoadAndDecryptSecrets`)
- Unused file watcher code in `internal/config/watch.go`

### Fixed

- SSRF bypass in URL validation by using `Hostname()` to strip port numbers and handle IPv6 brackets
- Directory traversal vulnerability in `RestoreBackup` by validating extraction paths
- Config cache mutation issues by invalidating cache after `SaveConfig`
- TOCTOU race condition in config cache
- Goroutine leaks in signal handler with context cancellation
- Race condition in file watcher `checkForChanges`
- Missing mutex protection for global `configDir`
- Missing exit code 1 on Claude harness error failure
- Gofmt pre-commit hook not receiving filenames
- Empty/malformed secret entries causing silent failures

### Security

- Pin install script downloads to specific release tag instead of main branch to reduce supply chain risk
- Validate backup extraction paths to prevent directory traversal attacks
- Fix SSRF bypass in URL validation that allowed `localhost:8080` to evade blocked-host checks

## [1.9.0] - 2026-02-15

### Added

- **Multi-harness support**: Added support for Qwen Code CLI harness alongside Claude Code
  - New `kairo harness get` command to display current default harness
  - New `kairo harness set <harness>` command to set default harness (claude or qwen)
  - New `--harness` flag for `switch` command to override harness per invocation
  - New `--model` flag for `switch` command to override model (passed to Qwen CLI)
  - New `DefaultHarness` field in Config struct for persistent harness selection
  - Harness names validated and case-insensitive (Claude/Qwen/claude/qwen all valid)
  - Invalid harness names default to claude with warning message
  - Order of precedence: `--harness` flag → config default → claude
  - 286 new tests for harness functionality and wrapper environment variable support

### Changed

- **Wrapper script enhancement**: Enhanced wrapper script generation to support custom environment variable names
  - Qwen harness uses `ANTHROPIC_API_KEY` instead of `ANTHROPIC_AUTH_TOKEN`
  - Wrapper script now accepts optional `envVarName` parameter
  - Maintains backward compatibility with existing Claude harness behavior
  - Improved security for API key delivery across different CLI harnesses

### Documentation

- **README**: Updated with multi-harness support documentation and examples
- **Command documentation**: Updated cmd/README.md with harness commands and usage examples
- **User guide**: Updated docs/guides/user-guide.md with harness setup instructions

## [1.8.4] - 2026-02-14

### Added

- **Audit context**: Added hostname, username, and session ID to audit entries
  - Improves traceability across users, hosts, and sessions
  - Context captured at logger creation and applied to all entries
  - Session IDs generated as unique 16-character hex identifiers
  - Backward compatible with existing audit logs (new fields use `omitempty`)
  - 7 new tests for context field functionality

### Fixed

- **Environment variable deduplication**: Fixed duplicate environment variables in `switch` command
  - Custom providers could create duplicate `ANTHROPIC_BASE_URL` and other built-in env vars
  - Added `mergeEnvVars()` helper with proper deduplication (last occurrence wins)
  - Order of precedence: system env vars → built-in Kairo env vars → provider EnvVars → secrets
  - Invalid env var formats (no '=' or empty key) are skipped
  - 13 new tests for mergeEnvVars functionality and performance
- **Thread-safe audit reads**: Fixed race condition in `LoadEntries()` method
  - Previous implementation could read log file while writes were in progress
  - Changed from `sync.Mutex` to `sync.RWMutex` for concurrent read access
  - Multiple goroutines can now read audit entries without blocking each other
  - Updated documentation to reflect thread-safe status
  - 2 new tests for concurrent read and read/write scenarios
- **Model validation**: Fixed empty model names for custom providers
  - Custom providers now require non-empty model names
  - Built-in providers (like anthropic) can still use empty values
  - Whitespace trimmed before validation
  - URL validation already enforced via `validateBaseURL()` (HTTPS required, blocks localhost/private IPs)
  - 22 new tests covering model and URL validation

## [1.8.3] - 2026-02-14

### Fixed

- **Config migration**: Fixed provider model not being updated during `kairo update` when builtin default model changes (e.g., MiniMax-M2 to MiniMax-M2.5)
  - Migration now properly updates both `provider.Model` and `default_models` when builtin defaults change
  - Added comprehensive test coverage for migration scenarios
- **Pre-commit hooks**: Added auto-install of staticcheck in deps target to prevent hook failures

### Changed

- **Config version field**: Deprecated unused `version` field in config.yaml
  - Marked with `omitempty` to allow omission from new configs
  - Maintains backward compatibility with existing configs

## [1.8.2] - 2026-02-14

### Fixed

- **Dependencies**: Removed unused `golang.org/x/net` and `golang.org/x/text` dependencies to fix CI tidy check
- **Pre-commit**: Updated hooks to use Go 1.25.7 to match CI and prevent version mismatches

## [1.8.1] - 2026-02-14

### Added

- **MiniMax model update**: Updated default model from MiniMax to M2.5
- **Auto-update config migration**: When `kairo update` successfully updates the CLI, it now automatically syncs configured providers with new default models from the updated built-in provider definitions
  - New `default_models` field in config tracks which model was set as default
  - Migration logic preserves user-customized models (only updates providers using default models)
  - Displays config changes after successful update (e.g., "Config updates: zai: glm-4.7 -> glm-4.8")

## [1.8.0] - 2026-02-09

### Added

- **Key recovery commands**: New `kairo recover` commands for key restoration
  - `kairo recover identity` - Recover identity file from passphrase
  - `kairo recover key <provider>` - Recover provider-specific encryption keys
  - `kairo recover all` - Recover all keys in batch
- **Config caching**: Added file watcher for automatic cache invalidation when config files change
  - Config caching layer integrated into commands for faster startup
  - Automatic cache invalidation on file modifications
- **Error recovery**: Improved error messages with actionable recovery suggestions
- **UI enhancement**: Clear terminal screen before running Claude for cleaner output

### Fixed

- **Backup security**: Fixed Zip Slip vulnerability in archive extraction
- **Backup resources**: Fixed resource leaks in backup operations
  - Added proper deferred Close handling
  - Added zip archive verification before extraction
- **Test isolation**: Fixed configDir reset in TestGetConfigDir to avoid test pollution

### Changed

- **Build system**: Replaced Taskfile and Makefile with Justfile for command runner
- **Pre-commit hooks**: Added staticcheck for Go static analysis
- **Pre-commit hooks**: Changed Windows cmd to bash for Go hooks
- **Development tools**: Added Just runner installation to deps target
- **Documentation**: Added backup and recovery documentation
- **Documentation**: Added metrics documentation to README
- **Documentation**: Improved table formatting in README metrics section
- **Tests**: Added integration tests for backup and recovery
- **Tests**: Used filepath.Join for cross-platform temp paths in metrics tests
- **Code style**: Fixed whitespace and handled ignored errors in tests

### Security

- **Backup extraction**: Fixed Zip Slip vulnerability preventing path traversal attacks

## [1.7.1] - 2026-02-06

### Fixed

- **Windows file locking**: Close audit logger to prevent file lock on Windows when running update command
- **Update command**: Use platform-specific temp file extensions (.tmp on Windows, .tmp.XXXXXX on Unix) to avoid extension issues
- **Wrapper script execution**: Fixed wrapper script execution on Windows by using correct directory and extension handling
- **CI/CD**: Removed invalid deny-licenses configuration in dependency review workflow
- **CI/CD**: Fixed coverage report step in CI pipeline

### Changed

- **Go version**: Updated to 1.25.7 to fix crypto/tls vulnerability (CVE-2024-45338)
- **Pre-commit hooks**: Added Windows-compatible PowerShell pre-commit script for developers on Windows

### Documentation

- **AGENTS.md**: Updated with comprehensive AI agent context for better Claude Code integration

## [1.7.0] - 2026-01-31

### Added

- **API key validation**: Strengthened validation with provider-specific formats
  - Anthropic keys: Must start with `sk-ant-api0` followed by 76+ characters
  - Z.AI keys: Must start with `sk-zaic-` followed by 32+ characters
  - MiniMax keys: Must start with `eyJ` (JWT format) or custom validation
  - DeepSeek keys: Must start with `sk-` followed by 52+ characters
  - Kimi keys: Must start with `sk-` followed by 52+ characters
  - Clear error messages indicating expected format for each provider
- **Decryption error handling**: Fail early on decryption failures with actionable errors
  - Clear guidance when identity file is missing or wrong
  - Better error messages for malformed recipient files
  - Integration tests for decryption failure scenarios

### Fixed

- **Go version**: Updated to 1.25.6 to fix crypto/tls vulnerability (CVE-2024-45338)
- **Dependencies**: Updated golang.org/x/crypto to v0.45.0 for security fixes
- **CI/CD**: Fixed coverage report step and updated dependency review for PATENTS
- **Update command**: Simplified to use platform-appropriate install scripts

### Refactored

- **Audit logging**: Made audit logging errors visible to callers instead of silent failures
- **Private IP validation**: Extracted CIDR blocks to package-level constants for maintainability
- **State management**: Removed unnecessary dual state in reset and rotate commands
- **Platform detection**: Consolidated in cmd/rotate with pkg/env for consistency
- **Validation helpers**: Removed redundant nil check in validateCustomProviderName

### Test

- **Integration tests**: Added decryption failure scenario tests
- **Audit helpers**: Added comprehensive test coverage
- **Crypto package**: Added disk full error handling tests
- **Switch command**: Increased test coverage with new run tests
- **Race detection**: Fixed race conditions in integration tests

### Documentation

- **Package-level docs**: Added documentation to cmd, crypto, and wrapper packages
- **Function docs**: Added documentation to utility helper and security-critical private functions
- **Documentation standardization**: Standardized function documentation format

## [1.6.1] - 2026-01-28

### Fixed

- **Reset command**: Remove age.key file when resetting all providers to ensure clean state

### Changed

- **Documentation**: Updated changelog with version link for v1.6.0
- **Contributing guide**: Added pre-commit to Before Submitting section
- **Markdownlint**: Migrated configuration to markdownlint-cli2 format
- **AGENTS.md**: Created concise version for AI assistant context
- **README**: Fixed install command URLs in documentation table

## [1.6.0] - 2026-01-20

### Added

- **Config file extension**: Changed config filename from `config` to `config.yaml`
  - Better format recognition and editor support with YAML extension
  - Automatic migration from old format on first run
  - Original file backed up as `config.backup` (never deleted)
  - Migration includes YAML validation before conversion
  - Permission preservation during migration
  - Atomic operation with rollback on failure
  - Comprehensive test coverage (7 new migration tests)
- **Audit logging**: Added `LogMigration()` method for future migration event tracking

### Fixed

- **Windows installer**: Fixed hashtable access for checksum hash validation
- **Windows self-update**: Implemented binary swap-after-exit pattern for reliable updates

## [1.5.1] - 2026-01-19

### Changed

- **Model reference**: Updated anthropic model reference to glm-4.7-flash
  - Synchronized model name across tests, documentation, and provider registry
  - Ensures consistency with current API defaults

## [1.5.0] - 2026-01-18

### Added

- **Performance metrics**: New `kairo metrics` command for monitoring CLI operations
  - Track execution time, memory usage, and operation counts
  - Export metrics in JSON or CSV format for analysis
  - `internal/performance` package with comprehensive metrics collection
  - Detailed guide in `docs/guides/performance-metrics.md`
- **Retry and panic recovery**: New `internal/recovery` package for resilient error handling
  - Retry utility with configurable attempts, exponential backoff, and jitter
  - Panic recovery with optional stack trace logging
  - Context-aware timeouts with automatic cancellation
  - Comprehensive test coverage (800+ lines)
- **Cross-provider validation**: New `kairo config --validate-all` command
  - Validates all configured providers in a single run
  - Returns structured validation report with per-provider status
  - Useful for pre-flight checks before critical operations
- **Error recovery and rollback**: Automatic transaction rollback on config failures
  - Atomic config updates that preserve previous state on error
  - Rollback mechanism for `kairo config` and `kairo setup` commands
  - Improved error messages with recovery hints
- **Secure token passing**: Wrapper script for secure API key delivery
  - Replaces insecure pipe-based token passing
  - `internal/wrapper` package with platform-specific script generation
  - Tokens written to temp file with 0600 permissions, auto-cleaned
  - Comprehensive security documentation in `docs/architecture/wrapper-scripts.md`
- **Confirmation prompts**: Destructive operations now require user confirmation
  - `kairo reset` prompts before removing provider configuration
  - `kairo rotate` prompts before regenerating encryption keys
  - Can be bypassed with `--yes` flag for automation
- **Dependency vulnerability scanning**: New GitHub Actions workflow
  - Automated security scanning on every pull request
  - Uses `govulncheck` with SARIF output for GitHub integration
  - Replaced deprecated `deny-licenses` with `allow-licenses` policy

### Changed

- **Provider name validation**: Enhanced with length limits and reserved words
  - Maximum length: 32 characters
  - Reserved words: `default`, `all`, `config`, `reset`, `rotate`, `setup`, `switch`, `test`, `status`, `list`, `version`, `update`, `audit`, `completion`, `metrics`
  - Prevents conflicts with built-in commands
- **YAML strict mode**: Config parser now rejects unknown fields
  - Prevents typos from being silently ignored
  - Explicit error messages for unrecognized configuration keys
- **Error handling consolidation**: Merged duplicate errors packages
  - Consolidated `internal/config/errors`, `internal/crypto/errors` into `internal/errors`
  - Single source of truth for typed errors and error context
- **PowerShell completion**: Simplified deployment process
  - New `scripts/kairo-completion.ps1` standalone completion script
  - Can be sourced directly or installed via `kairo completion --save`
  - Improved Windows developer experience
- **Windows special character handling**: Replaced batch scripts with PowerShell
  - Better support for spaces, Unicode, and special characters in paths
  - Consistent behavior across all platforms

### Security

- **Secure token passing**: Replaced insecure `curl | sh` and `irm | iex` patterns
  - Update command now downloads to temp file with checksum verification
  - Windows installer uses temp file instead of direct execution
  - Wrapper script securely passes tokens via file descriptors
- **Audit log sanitization**: API keys now completely masked in audit logs
  - Previous implementation showed partial keys; now fully redacted
  - Format: `sk-***` instead of `sk-an***mnop`

### Fixed

- **Thread safety**: Fixed race conditions in signal handling
  - Used `sync.Once` to ensure single signal handler registration
  - Confirmation flag handling made thread-safe with RWMutex
  - All global state access now uses mutex-protected accessors
- **Config directory**: Improved Windows support and thread safety
  - Added mutex-protected `configDir` with proper RWMutex
  - Removed unused `configDirOnce` variable
  - Better handling of Windows paths with forward/backward slashes
- **Provider name regex**: Fixed to allow underscores and hyphens
  - Previous regex only allowed alphanumeric characters
  - Custom providers can now use `my-provider` or `my_provider` style names
- **Audit logging**: Added file `Sync()` for write durability
  - Prevents data loss on crashes or power failures
  - Ensures audit entries are flushed to disk
- **Secrets parsing**: Skip entries with empty keys
  - Prevents crashes on malformed environment variable entries
  - Graceful handling of edge cases in ParseSecrets
- **UI prompt functions**: Now return errors for proper error handling
  - `Prompt()`, `PromptWithDefault()`, `Confirm()` return error
  - Allows callers to handle user cancellation (Ctrl+C)
  - Improved test signal handling safety
- **Remove hardcoded /tmp paths**: Tests now use temp directories
  - Cross-platform compatibility (Windows uses different temp location)
  - Better isolation between test runs

### Test

- **Integration tests**: Added `cmd/integration_test.go` with end-to-end scenarios
  - Tests complete workflows (setup, config, switch, reset)
  - Validates cross-provider functionality
- **PowerShell escaping**: Added edge case tests for special characters
  - `scripts/test-powershell-escaping.ps1` for validation
  - Covers quotes, backticks, Unicode, and other special characters
- **Expanded coverage**: Significant test coverage improvements
  - `cmd/switch_test.go`: 1000+ lines of comprehensive tests
  - `cmd/metrics_test.go`: 200+ lines for metrics command
  - `cmd/update_test.go`: 350+ lines for update functionality
  - `internal/audit`: 350+ lines of audit logging tests
  - `internal/recovery`: 800+ lines of recovery utility tests
  - `internal/wrapper`: 400+ lines of wrapper script tests

### Documentation

- **Best practices guide**: New `docs/best-practices.md` with 600+ lines
  - Security guidelines for API key management
  - Multi-provider configuration examples
  - Performance optimization tips
  - Error handling patterns
- **Wrapper script architecture**: New `docs/architecture/wrapper-scripts.md`
  - Security design rationale
  - Platform-specific implementation details
  - Threat model and mitigation strategies
- **Advanced configuration**: Expanded `docs/guides/advanced-configuration.md`
  - Multi-provider setup examples
  - Custom provider configuration
  - Environment variable integration
- **Performance metrics guide**: New `docs/guides/performance-metrics.md`
  - Metrics collection overview
  - Export and analysis workflows
  - Integration with monitoring tools
- **Updated documentation**:
  - README with new features and examples
  - Architecture documentation with wrapper script details
  - AGENTS.md with updated provider name regex
  - Contributing guide with new test patterns

### Build

- **Taskfile**: Added Windows support for all build tasks
  - Cross-platform build, test, and lint commands
  - PowerShell scripts for Windows-specific operations
- **CI/CD improvements**:
  - Enhanced vulnerability scanning workflow
  - Better caching strategies for faster builds
  - Improved error reporting in CI logs

## [1.4.3] - 2026-01-13

### Fixed

- **Windows installer**: Fixed path escape character issue in default install directory
  - Changed `"$env:USERPROFILE\.local\bin"` to `Join-Path $env:USERPROFILE ".local\bin"`
  - Backslash before "local" was being interpreted as escape character, causing malformed paths
  - Path now correctly resolves to `C:\Users\username\.local\bin`
- **Update command**: Removed unused imports (`path/filepath`, `strings`) from cmd/update.go
- **Build**: Added Windows-specific Taskfile configuration using PowerShell script
  - Created `scripts/build.ps1` for proper git version detection on Windows
  - Fixed `task build` to display correct version on Windows

## [1.4.2] - 2026-01-13

### Fixed

- **Windows installer**: Added Get-FileHash compatibility for PowerShell 2.0+
  - The Get-FileHash cmdlet is only available in PowerShell 4.0+
  - Added Get-FileHashCompat function that falls back to .NET System.Security.Cryptography.SHA256 for older PowerShell versions
  - Installer now works on Windows 7 and earlier versions with PowerShell 2.0/3.0
- **Windows installer**: Fixed checksum regex to match GoReleaser format
  - Changed pattern from `^([a-f0-9]+)\s+\*($($BinaryName)_windows_)` to `^([a-f0-9]+)\s+($($BinaryName)_windows_\S+)`
  - GoReleaser generates checksums with two spaces instead of asterisk prefix
  - Checksum verification now works correctly for Windows binaries

## [1.4.1] - 2026-01-11

### Fixed

- **Install script**: Fixed version variable reference causing empty version in download URL
  - Changed `$version` to `$VERSION` in log statement to display version correctly
  - Prevents 404 errors when installing specific versions
- **Update command**: Added User-Agent header to GitHub API requests
  - Improves API request identification and reliability

## [1.4.0] - 2026-01-08

### Added

- **Self-update command**: New `kairo update` command to check for and install the latest version
  - Fetches latest release from GitHub API with configurable URL via `KAIRO_UPDATE_URL` environment variable
  - Cross-platform support: PowerShell installer for Windows, curl|sh for Unix
  - Version comparison using semver library with proper pre-release handling (alpha, beta, rc)
  - Timeout protection (10s) for API requests with context cancellation
  - Comprehensive error handling for network failures, timeouts, and invalid responses

### Test

- **Update command coverage**: Added 2 unit tests for update functionality
  - OS detection and install script URL selection

## [1.3.0] - 2026-01-07

### Added

- **Windows support**: PowerShell installer for Windows platforms
  - Run `irm https://raw.githubusercontent.com/dkmnx/kairo/main/scripts/install.ps1 | iex` to install
  - Supports AMD64, ARM64, and ARM7 architectures
  - Automatic binary placement in `%LOCALAPPDATA%\Programs\kairo`
  - Checksum verification for downloaded releases
  - Auto-save completion files to PowerShell Modules directory
- **Build automation**: Taskfile.yml with comprehensive build system
  - Modern task runner to replace Makefile
  - Build, test, and lint commands with proper output directories
  - Dependency management and verification tasks
  - Release automation with goreleaser integration
  - Cross-platform installation and cleanup utilities
  - Coverage reporting and race detection support

### Changed

- **Installer**: Change default install directory to user local bin (`~/.local/bin` on Unix, `%LOCALAPPDATA%\Programs` on Windows)

### Fixed

- **Windows installer**: Handle existing binary during reinstall
  - Explicitly remove existing binary before moving new version to avoid "file already exists" errors
- **Windows installer**: Improved error handling for locked binary
  - Clear error messages when binary is in use or permissions are insufficient
  - Suggests closing running processes or running as Administrator
- **Windows installer**: Use approved PowerShell verbs
  - Renamed functions to comply with PowerShell naming conventions (Get, Test, Install vs Show, Verify, Download)
- **Completion**: Standardize documentation and improve consistency
  - Use `--save` flag consistently across all shell examples
  - Capitalize shell names (Bash, Zsh, Fish, PowerShell) for consistency
  - Standardize language to "every new session"
  - Fix comment discrepancies in completion code and tests

### Test

- **Cross-platform compatibility**: Make tests work on Windows
  - Skip Unix-style file permission checks (0600) on Windows
  - Skip read-only directory tests on Windows (different behavior)
  - Fix TestGetConfigDirWithEnv to set both HOME and USERPROFILE
  - Skip TestConfigureProvider on Windows (requires TTY/PTY)

## [1.2.3] - 2026-01-06

### Fixed

- **Setup wizard**: Fixed `kairo setup` command that was failing with "configuration file not found" error
  - The `loadOrInitializeConfig()` function used `os.IsNotExist()` to check for missing config
  - Changed to use `errors.Is(err, config.ErrConfigNotFound)` for proper custom error detection
  - Setup wizard now correctly creates a new config when none exists
- **Provider order**: Fixed inconsistent provider ordering in setup wizard
  - `GetProviderList()` previously iterated over a Go map with non-deterministic order
  - Defined explicit `providerOrder` slice for consistent display: anthropic, zai, minimax, deepseek, kimi, custom

## [1.2.2] - 2026-01-04

### Fixed

- **Audit errors**: Audit logger failures now log to stderr instead of being silently ignored
  - Fixes potential data loss when audit logging fails (permission errors, disk full, etc.)
  - Added `logAuditEvent()` helper function for centralized error handling
- **Thread safety**: Fixed global variable race conditions in concurrent scenarios
  - Added mutex-protected accessors: `getVerbose()`, `setVerbose()`, `setConfigDir()`
  - Updated all production code to use thread-safe accessors for `verbose` flag reads
  - Updated all tests to use thread-safe accessors for global variable access
  - Added race condition tests in `cmd/race_test.go`
- **Lock optimization**: Reduced lock hold time in `getConfigDir()` by releasing before `env.GetConfigDir()`
- **Duplicate binding**: Removed duplicate `verbose` flag binding in `cmd/audit.go`
  - Persistent flags in `cmd/root.go` are automatically inherited by all subcommands
- **Flag parsing**: Removed redundant manual `--config` parsing in `Execute()` function
  - Cobra's persistent flag binding already handles this correctly

### Changed

- **Code quality**: Removed unused `getConfigDirRaw()` function (YAGNI)

## [1.2.1] - 2026-01-03

### Fixed

- **Shell completion**: Fixed `kairo completion fish` and other shell completion commands
  - The `__complete` and `__completeNoDesc` hidden Cobra commands were being intercepted and treated as provider names
  - Added check in `cmd/root.go` to allow these completion commands to pass through unchanged
  - Fish completions now properly list all subcommands and providers

### Documentation

- Updated documentation for audit feature and architecture
- Added `docs/guides/audit-guide.md` with comprehensive audit log usage documentation
- Updated `cmd/README.md` with new command reference, architecture diagrams, and audit integration docs
- Updated `docs/architecture/README.md` with audit logging architecture, error handling patterns, and design principles

## [1.2.0] - 2026-01-02

### Added

- **Audit logging**: New `kairo audit` command to track all configuration changes
  - `kairo audit list` - Human-readable list showing timestamp, provider, action, and changes
  - `kairo audit export -o file.csv` - Export to CSV format with changes column
  - `kairo audit export -o file.json -f json` - Export to JSON format
  - API keys masked in logs (e.g., `sk-an********mnop`)
  - Tracks: config, default, reset, rotate, setup, and switch commands

## [1.1.1] - 2026-01-02

### Added

- **Provider shorthand**: Use `kairo <provider>` instead of `kairo switch <provider>` for quicker provider switching
  - Arguments after provider name are passed through to Claude (e.g., `kairo anthropic --help`)
  - Unknown provider names now show "not configured" error instead of "unknown command"
  - All existing subcommands remain unaffected

### Fixed

- **Race conditions**: Fixed test flakiness caused by shared flag state in CLI tests
- **Test pollution**: Prevented argument handling from affecting subsequent tests
- **Code formatting**: Added auto-formatting to `make lint` for automatic gofmt fixes before checking
- **Style cleanup**: Fixed formatting in root_test.go

## [1.1.0] - 2025-12-29

### Added

- **Provider shorthand**: Use `kairo <provider>` instead of `kairo switch <provider>` for quicker provider switching
  - Arguments after provider name are passed through to Claude (e.g., `kairo anthropic --help`)
  - Unknown provider names now show "not configured" error instead of "unknown command"
  - All existing subcommands remain unaffected

## [1.0.2] - 2025-12-28

### Fixed

- **Update command**: Fixed checksum file download error by correcting filename construction
  - Script was using version with 'v' prefix (e.g., `kairo_v1.0.1_checksums.txt`)
  - Actual release asset uses format without prefix (e.g., `kairo_1.0.1_checksums.txt`)
  - Added version prefix stripping logic to match actual GoReleaser naming
- **Install script**: Fixed shellcheck warnings for safer script execution
  - Changed `trap` command to use single quotes to prevent early variable expansion
  - Fixed PATH export statement for proper variable display

## [1.0.1] - 2025-12-28

### Added

- **Structured error handling**: Implemented `internal/errors` package with typed errors (ConfigError, CryptoError, ValidationError, ProviderError, NetworkError, FileSystemError)
- **Error context**: All errors now include structured context and helpful hints for debugging
- **Comprehensive user guides**: Added new documentation:
  - Error handling examples (`docs/guides/error-handling-examples.md`)
  - Advanced configuration scenarios (`docs/guides/advanced-configuration.md`)
  - Claude Code integration examples (`docs/guides/claude-integration-examples.md`)
- **Enhanced troubleshooting**: Added 7 advanced troubleshooting scenarios to `docs/troubleshooting/README.md`
- **Setup helper tests**: Added tests for prompt functions (promptForProvider, promptForAPIKey, promptForBaseURL)
- **Update command tests**: Added tests for getEnvFunc, getLatestRelease, and versionGreaterThan
- **Helper function tests**: Added tests for parseIntOrZero, parseProviderSelection, validateCustomProviderName

### Changed

- **Internal refactoring**: Extracted 8 helper functions from `cmd/setup.go` for better maintainability:
  - `validateCustomProviderName`, `buildProviderConfig`, `getSortedSecretsKeys`
  - `formatSecretsFileContent`, `saveProviderConfigFile`, `validateAPIKey`, `validateBaseURL`
- **Improved error messages**: All config and crypto errors now use structured types with context
- **Better test coverage**: cmd package coverage increased from 35.2% to 40.5%
  - Added tests for edge cases and error paths in update.go
  - Increased coverage for version comparison from 71.4% to 100%
  - Increased coverage for getLatestRelease from 78.9% to 89.5%
- **README improvements**: Added usage examples, reorganized documentation section, added Direct Query Mode (`--`) documentation

### Fixed

- **Documentation**: Fixed 15 markdownlint errors across new documentation files
- **Error handling tests**: internal/errors package coverage improved from 78.1% to 100%

## [1.0.0] - 2025-12-28

### Added

- Production-ready release with complete feature set
- Comprehensive documentation suite (User Guide, Development Guide, Architecture, Troubleshooting, Contributing, Deployment Guide)
- 80%+ test coverage across all packages
- Multi-platform binary releases (Linux, macOS, Windows amd64/arm64)
- Homebrew tap support for easy installation
- Automated CI/CD pipeline with testing, linting, and releasing

### Changed

- Stabilized CLI interface for production use
- Standardized configuration format (YAML) for backward compatibility
- Achieved comprehensive test coverage across all modules

### Security

- Age (X25519) encryption for all API keys
- Automatic key rotation with `kairo rotate` command
- 0600 file permissions on sensitive files (config, secrets.age, age.key)
- Secrets decrypted in-memory only
- No plaintext API keys stored in configuration files

## [0.5.3] - 2025-12-28

### Added

- **Version tests**: Added `internal/version/version_test.go` with 5 tests for version parsing and validation
- **UI output tests**: Added `internal/ui/prompt_test.go` with comprehensive tests for all Print functions
- **cmd package tests**: Expanded test coverage from 29% to 36.3% with new tests for:
  - `ensureConfigDirectory`, `loadOrInitializeConfig`, `loadSecrets`
  - `parseProviderSelection`, `configureAnthropic`
  - `checkForUpdates` (update notification logic)
- **Semver dependency**: Added `github.com/Masterminds/semver/v3` for proper version comparison

### Changed

- **Semver parsing**: Replaced fragile string comparison with proper `Masterminds/semver` library
  - Fixes edge cases: `v0.9.0` vs `v0.10.0`, pre-release versions, multi-digit version numbers
- **getEnvValue()**: Implemented using `os.Getenv()` to enable `KAIRO_UPDATE_URL` environment variable override

### Fixed

- **UI constants**: Removed unused lowercase duplicate constants, all functions now use uppercase constants (Green, Yellow, Red, etc.)
- **Code cleanup**: Updated CODE_REVIEW.md to reflect all completed items

## [0.5.2] - 2025-12-27

### Fixed

- `kairo update`: Fixed install script execution so it properly runs instead of just printing to terminal

## [0.5.1] - 2025-12-27

### Changed

- UI: Introduced gray color for non-default providers in list and status output
- UI: Default provider now displays "(default)" indicator in list command

### Testing

- Increased test coverage across config, env, and ui packages (+6.1% overall)
- Added comprehensive tests for crypto package (loadRecipient edge cases, file errors)
- Added validation tests for isPrivateIP and isBlockedHost functions
- Added ui package tests for all Print functions (PrintSuccess, PrintWarn, PrintError, etc.)
- Added config tests for file not found, invalid YAML, empty providers, permissions

### Fixed

- Config: Fixed file permission check in crypto tests for root user environments
- Validation: Simplified isBlockedHost loop using slices.Contains

## [0.5.0] - 2025-12-27

### Added

- `kairo update` now automatically downloads and installs the latest version without requiring manual intervention

## [0.4.2] - 2025-12-27

### Changed

- List output: Improved formatting with ❯ prefix and aligned URL/Model labels
- Status output: Restructured to single-line format per provider
- Default provider now highlighted in green across list and status commands

### Fixed

- CI: Removed broken Homebrew update job that failed due to missing formula

### Documentation

- Added version badge to README
- Updated project description with key features
  - Fixed ASCII art formatting in README

## [0.4.1] - 2025-12-27

### Fixed

- Release workflow: Added `rm -rf .claude/` to goreleaser jobs to prevent dirty state errors
- Fixed lint issues: unchecked error returns in test HTTP handlers
- Fixed lint issues: removed unused helper functions (`getenv`, `createUpdateCommand`, `createVersionCommand`)

## [0.4.0] - 2025-12-27

### Added

- `kairo update` command to check for and report new releases
- Auto-update notification in `kairo version` command when new version available
- `make ci-local` and `make ci-local-list` targets for testing GitHub Actions locally with act

### Changed

- Upgraded to Go 1.25.5
- Updated golangci-lint to v1.62.0
- Simplified CI caching using `setup-go` built-in caching instead of manual `actions/cache`
- Updated CodeQL action to v4
- Updated build matrix to test Go 1.25.5 only

### Fixed

- CI workflow: Fixed gosec SARIF generation and upload issues
- CI workflow: Fixed golangci-lint Go version compatibility
- CI workflow: Fixed govulncheck dependency issues
- CI workflow: Added proper permissions block for GitHub Actions
- Release workflow: Fixed goreleaser path resolution
- Release workflow: Fixed cosign version (v2 → v2.5.2)
- Crypto test: Skip readonly directory test when running as root (Docker/act)

## [0.3.0] - 2025-12-27

### Added

- `reset` command for removing provider configurations
- `completion` command with configurable output paths (`kairo completion --write-compile-file`)
- Color highlighting for provider display in list and status commands
- Comprehensive test coverage (config, crypto, providers, validate, CLI commands)
- CI/CD workflows for testing, linting, and releasing
- Pre-commit configuration for code quality enforcement
- AGENTS.md with project guidelines, guardrails, and best practices
- Development tooling documentation

### Changed

- Providers now sorted with default provider first in `list` and `status` commands
- Extracted provider sorting logic into shared helper function
- Improved completion error handling for better user experience
- Documentation restructured with architecture, contributing, guides, and troubleshooting sections

### Fixed

- Completion error handling for edge cases

### Build

- GoReleaser v2 configuration updated
- Multi-platform release builds (Linux, Darwin, Windows, amd64/arm64)

## [0.2.2] - 2025-12-26

### Fixed

- **Critical:** Fixed secrets key format inconsistency - API keys now stored with uppercase provider names (e.g., `ZAI_API_KEY`) for consistent lookup across all commands
- Switch command now exits with status 1 when Claude execution fails
- Status command now correctly detects API keys stored with uppercase key format

### Changed

- Standardized secrets key format to use uppercase provider names consistently across `setup`, `config`, `status`, and `switch` commands
- Added constants for hardcoded Claude environment variable names (ANTHROPIC_BASE_URL, ANTHROPIC_MODEL, etc.)
- Custom provider names now validated to start with a letter and contain only alphanumeric characters, underscores, or hyphens
- Made `exec.Command` and `os.Exit` mockable in tests for better testability
- Refactored secrets parsing to use `config.ParseSecrets()` consistently instead of duplicate code

### Added

- Godoc comments for exported functions in `cmd` package
- Test suite for status command (`status_test.go`) with coverage for key format consistency

### Migration Note

After upgrading to v0.2.2, existing users need to reconfigure their providers to save API keys with the correct uppercase format:

```bash
kairo config <provider>
```

This ensures secrets are stored as `PROVIDER_API_KEY` (e.g., `ZAI_API_KEY`) instead of the previous lowercase format.

## [0.2.1] - 2025-12-26

### Changed

- Adjust alignment in banner output for better aesthetics

## [0.2.0] - 2025-12-26

### Added

- Comprehensive test suite for banner and version output
- Install script for quick cross-platform installation
- One-liner install command in README

### Changed

- Banner now displays version and provider in format `v0.1.0 - Provider`
- Date format in version output: `2025-12-26` instead of RFC3339 timestamp
- Upgrade to GoReleaser v2 with updated configuration format

### Fixed

- Version command now shows formatted date (YYYY-MM-DD)
- Banner includes version information when switching providers

## [0.1.0] - 2025-12-26

### Added

- Initial CLI implementation with Cobra framework
- Provider configuration management (Native Anthropic, Z.AI, MiniMax, Kimi, DeepSeek, Custom)
- Age encryption for API key storage (filippo.io/age)
- YAML configuration support
- Interactive setup wizard (`kairo setup`)
- Provider configuration commands (`kairo config <provider>`)
- Provider listing (`kairo list`)
- Provider switching and Claude execution (`kairo switch <provider>`)
- Default provider management (`kairo default [provider]`)
- Provider testing (`kairo test <provider>`)
- Multi-provider status check (`kairo status`)
- Version command (`kairo version`)
- Colored terminal output with ui package
- Input validation for API keys and URLs
- Config file permissions (0600)
- Release builds for Linux, Darwin, Windows (amd64, arm64)
- Homebrew tap support

### Security

- Encrypted secrets storage using age encryption
- Private key management (`age.key`)
- API keys never logged or exposed in plaintext

### Build

- GoReleaser integration for releases
- goreleaser.yaml configuration
- Install script for cross-platform installation

[2.0.0]: https://github.com/dkmnx/kairo/compare/v1.10.1...v2.0.0
[1.10.1]: https://github.com/dkmnx/kairo/compare/v1.10.0...v1.10.1
[1.10.0]: https://github.com/dkmnx/kairo/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/dkmnx/kairo/compare/v1.8.4...v1.9.0
[1.8.4]: https://github.com/dkmnx/kairo/compare/v1.8.3...v1.8.4
[1.8.3]: https://github.com/dkmnx/kairo/compare/v1.8.2...v1.8.3
[1.8.2]: https://github.com/dkmnx/kairo/compare/v1.8.1...v1.8.2
[1.8.1]: https://github.com/dkmnx/kairo/compare/v1.8.0...v1.8.1
[1.8.0]: https://github.com/dkmnx/kairo/compare/v1.7.1...v1.8.0
[1.7.1]: https://github.com/dkmnx/kairo/compare/v1.7.0...v1.7.1
[1.7.0]: https://github.com/dkmnx/kairo/compare/v1.6.1...v1.7.0
[1.6.1]: https://github.com/dkmnx/kairo/compare/v1.6.0...v1.6.1
[1.6.0]: https://github.com/dkmnx/kairo/compare/v1.5.1...v1.6.0
[1.5.1]: https://github.com/dkmnx/kairo/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/dkmnx/kairo/compare/v1.4.3...v1.5.0
[1.4.3]: https://github.com/dkmnx/kairo/compare/v1.4.2...v1.4.3
[1.4.2]: https://github.com/dkmnx/kairo/compare/v1.4.1...v1.4.2
[1.4.1]: https://github.com/dkmnx/kairo/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/dkmnx/kairo/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/dkmnx/kairo/compare/v1.2.3...v1.3.0
[1.2.3]: https://github.com/dkmnx/kairo/compare/v1.2.2...v1.2.3
[1.2.2]: https://github.com/dkmnx/kairo/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/dkmnx/kairo/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/dkmnx/kairo/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/dkmnx/kairo/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/dkmnx/kairo/compare/v1.0.2...v1.1.0
[1.0.2]: https://github.com/dkmnx/kairo/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/dkmnx/kairo/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/dkmnx/kairo/compare/v0.5.3...v1.0.0
[0.5.3]: https://github.com/dkmnx/kairo/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/dkmnx/kairo/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/dkmnx/kairo/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/dkmnx/kairo/compare/v0.4.2...v0.5.0
[0.4.2]: https://github.com/dkmnx/kairo/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/dkmnx/kairo/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/dkmnx/kairo/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/dkmnx/kairo/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/dkmnx/kairo/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/dkmnx/kairo/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/dkmnx/kairo/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/dkmnx/kairo/releases/tag/v0.1.0
