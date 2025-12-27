# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.4.2]: https://github.com/dkmnx/kairo/compare/v0.4.1...v0.4.2
[0.4.0]: https://github.com/dkmnx/kairo/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/dkmnx/kairo/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/dkmnx/kairo/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/dkmnx/kairo/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/dkmnx/kairo/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/dkmnx/kairo/releases/tag/v0.1.0
