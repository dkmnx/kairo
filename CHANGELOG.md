# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
