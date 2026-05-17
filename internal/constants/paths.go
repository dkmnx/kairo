package constants

// KeyFileName and SecretsFileName are the default file names for the
// encryption key and encrypted secrets stored in the config directory.
const (
	KeyFileName     = "age.key"
	SecretsFileName = "secrets.age"
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
