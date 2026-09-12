package harness

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	Claude = "claude"
	Qwen   = "qwen"
	Pi     = "pi"
	Crush  = "crush"
)

// Supported returns harness names in PATH-detection priority order.
// Used when no harness is configured: the first installed CLI wins.
func Supported() []string {
	return []string{Pi, Claude, Qwen, Crush}
}

// IsValid reports whether name is one of the supported harnesses.
func IsValid(name string) bool {
	return name == Claude || name == Qwen || name == Pi || name == Crush
}

// Resolve returns the first valid harness among flag then config.
// Empty string means neither is a valid configured value; callers should
// then DetectInstalled and, if that is also empty, prompt the user to
// install a supported harness.
func Resolve(flagHarness, configHarness string) string {
	if IsValid(flagHarness) {
		return flagHarness
	}
	if IsValid(configHarness) {
		return configHarness
	}

	return ""
}

// DetectInstalled returns the first supported harness found on PATH
// (in Supported order), or "" when none are installed.
func DetectInstalled(lookPath func(file string) (string, error)) string {
	if lookPath == nil {
		return ""
	}
	for _, name := range Supported() {
		if _, err := lookPath(name); err == nil {
			return name
		}
	}

	return ""
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

// PiEnvVars returns environment variables for the Pi harness.
func PiEnvVars(providerName, model string) []string {
	return []string{
		fmt.Sprintf("PI_PROVIDER=%s", providerName),
		fmt.Sprintf("PI_MODEL=%s", model),
	}
}

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]`)

// APIKeyEnvVar returns the conventional environment variable name for a provider's API key.
// Non-alphanumeric characters in the provider name are replaced with underscores.
func APIKeyEnvVar(providerName string) string {
	sanitized := nonAlphanumeric.ReplaceAllString(strings.ToUpper(providerName), "_")

	return fmt.Sprintf("%s_API_KEY", sanitized)
}
