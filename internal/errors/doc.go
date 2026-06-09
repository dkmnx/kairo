// Package errors defines Kairo-specific error types with structured context.
//
// # Context Keys
//
// Canonical context keys used in production code with WithContext().
// Test files may use additional ad-hoc keys (e.g. "action", "host", "port")
// that are not part of this convention.
//
//	"path"         - file path
//	"config_dir"   - configuration directory path
//	"key_path"     - encryption key file path
//	"secrets_path" - encrypted secrets file path
//	"old_path"     - previous file path (for migration errors)
//	"new_path"     - target file path (for migration errors)
//	"provider"     - provider name
//	"hint"         - user-facing troubleshooting hint
//	"model"        - model name
//	"env_var"      - environment variable name
//	"mode"         - file permission mode
//	"expected"     - expected value (e.g. hash)
//	"actual"       - actual value (e.g. hash)
//	"output"       - command output
//
// Keep keys lowercase_snake_case and consistent across all packages.
package errors
