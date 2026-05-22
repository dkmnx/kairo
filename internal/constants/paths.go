// Package constants defines file names, permission modes, timeouts, environment
// variable names, and GitHub repository URLs used throughout the kairo application.
package constants

import (
	"os"
	"time"
)

// KeyFileName and SecretsFileName are the default file names for the
// encryption key and encrypted secrets stored in the config directory.
const (
	KeyFileName     = "age.key"
	SecretsFileName = "secrets.age"
)

// File and directory permission modes used across the application.
var (
	// DirPermSecure is used for directories containing sensitive data (0700).
	DirPermSecure = os.FileMode(0o700)

	// DirPermDefault is used for general-purpose directories (0o755).
	DirPermDefault = os.FileMode(0o755)

	// FilePermSecure is used for files containing sensitive data (0o600).
	FilePermSecure = os.FileMode(0o600)

	// FilePermDefault is used for general-purpose files (0o644).
	FilePermDefault = os.FileMode(0o644)

	// FilePermExec is used for executable scripts (0o755).
	FilePermExec = os.FileMode(0o755)
)

// Timeout durations used across the application.
const (
	// ConfigCacheTTL is how long a cached config is considered valid.
	ConfigCacheTTL = 5 * time.Minute

	// RequestTimeout is the default timeout for HTTP requests to external APIs.
	RequestTimeout = 10 * time.Second
)

// Environment variable names for Anthropic-compatible provider configuration.
const (
	EnvBaseURL     = "ANTHROPIC_BASE_URL"
	EnvModel       = "ANTHROPIC_MODEL"
	EnvHaikuModel  = "ANTHROPIC_DEFAULT_HAIKU_MODEL"
	EnvSonnetModel = "ANTHROPIC_DEFAULT_SONNET_MODEL"
	EnvOpusModel   = "ANTHROPIC_DEFAULT_OPUS_MODEL"
	EnvSmallFast   = "ANTHROPIC_SMALL_FAST_MODEL"
	EnvAuthToken   = "ANTHROPIC_AUTH_TOKEN"
)
