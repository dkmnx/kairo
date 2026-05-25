package harness

import (
	"fmt"
	"strings"
)

const (
	Claude = "claude"
	Qwen   = "qwen"
	Pi     = "pi"
	Crush  = "crush"
)

// IsValid reports whether name is one of the supported harnesses.
func IsValid(name string) bool {
	return name == Claude || name == Qwen || name == Pi || name == Crush
}

// Resolve returns the effective harness given a flag override and config default.
// Falls back to "claude" when both are empty or the resolved value is unrecognized.
func Resolve(flagHarness, configHarness string) string {
	h := flagHarness
	if h == "" {
		h = configHarness
	}
	if h == "" {
		return Claude
	}
	if !IsValid(h) {
		return Claude
	}

	return h
}

// Dispatch returns the display name, environment variable name, and any extra
// CLI arguments for the given harness configuration.
func Dispatch(h, providerName, model string) (displayName, envVarName string, extraArgs []string) {
	switch h {
	case Qwen:
		return "Qwen", "ANTHROPIC_API_KEY", []string{"--auth-type", "anthropic", "--model", model}
	case Crush:
		return "Crush", APIKeyEnvVar(providerName), nil
	case Pi:
		return "Pi", "", nil
	default:
		return "Claude", "", nil
	}
}

// YoloFlag returns the harness-specific flag for skipping permission prompts.
func YoloFlag(h string) string {
	switch h {
	case Qwen, Crush:
		return "--yolo"
	case Pi:
		return ""
	default:
		return "--dangerously-skip-permissions"
	}
}

// PiEnvVars constructs the environment variables for the Pi harness.
func PiEnvVars(providerName, model string) []string {
	return []string{
		fmt.Sprintf("PI_PROVIDER=%s", providerName),
		fmt.Sprintf("PI_MODEL=%s", model),
	}
}

// BuiltInEnvVars constructs the standard Anthropic environment variables for a provider.
func BuiltInEnvVars(baseURL, model string) []string {
	return []string{
		fmt.Sprintf("ANTHROPIC_BASE_URL=%s", baseURL),
		fmt.Sprintf("ANTHROPIC_MODEL=%s", model),
		fmt.Sprintf("ANTHROPIC_HAIKU_MODEL=%s", model),
		fmt.Sprintf("ANTHROPIC_SONNET_MODEL=%s", model),
		fmt.Sprintf("ANTHROPIC_OPUS_MODEL=%s", model),
		fmt.Sprintf("ANTHROPIC_SMALL_FAST_MODEL=%s", model),
		"NODE_OPTIONS=--no-deprecation",
	}
}

// APIKeyEnvVar returns the conventional environment variable name for a provider's API key.
func APIKeyEnvVar(providerName string) string {
	return fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
}
