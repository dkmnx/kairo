// Package providers defines the built-in provider registry with names, base URLs,
// default models, environment variables, API key requirements, and key format rules.
package providers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dkmnx/kairo/internal/errors"
)

const (
	MinAPIKeyLength     = 32
	DefaultMinKeyLength = 20
)

// KeyFormat holds minimum length, prefix, and pattern rules for API key validation.
type KeyFormat struct {
	MinLength int
	Prefix    string
	Pattern   string
	compiled  *regexp.Regexp
}

func (kf *KeyFormat) validateForKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	if kf.MinLength > 0 && len(key) < kf.MinLength {
		return fmt.Errorf("API key too short (minimum %d characters, got %d)", kf.MinLength, len(key))
	}
	if kf.Prefix != "" && !strings.HasPrefix(key, kf.Prefix) {
		return fmt.Errorf("API key must start with '%s'", kf.Prefix)
	}
	if kf.Pattern != "" {
		if kf.compiled == nil {
			compiled, err := regexp.Compile(kf.Pattern)
			if err != nil {
				return fmt.Errorf("invalid key pattern for provider: %w", err)
			}
			kf.compiled = compiled
		}
		if !kf.compiled.MatchString(key) {
			return fmt.Errorf("API key format is invalid")
		}
	}

	return nil
}

var (
	KeyFormatMin32      = KeyFormat{MinLength: MinAPIKeyLength}
	KeyFormatAnthropic  = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "sk-ant-"}
	KeyFormatOpenAI     = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "sk-"}
	KeyFormatGroq       = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "gsk_"}
	KeyFormatOpenRouter = KeyFormat{MinLength: MinAPIKeyLength, Prefix: "sk-or-"}
	DefaultKeyFormat    = KeyFormat{MinLength: DefaultMinKeyLength}
)

// builtInProviders maps provider short names to their definitions.
var builtInProviders = map[string]ProviderDefinition{
	"zai": {
		Name:           "Z.AI",
		BaseURL:        "https://api.z.ai/api/anthropic",
		Model:          "glm-5.1",
		RequiresAPIKey: true,
		EnvVars:        []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash"},
		APIKeyEnvVar:   "ZAI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"minimax": {
		Name:           "MiniMax",
		BaseURL:        "https://api.minimax.io/anthropic",
		Model:          "MiniMax-M2.7",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=120",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=24576",
		},
		APIKeyEnvVar: "MINIMAX_API_KEY",
		KeyFormat:    KeyFormatMin32,
	},
	"kimi": {
		Name:           "Moonshot AI",
		BaseURL:        "https://api.kimi.com/coding/",
		Model:          "kimi-for-coding",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=240",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=200000",
		},
		APIKeyEnvVar: "KIMI_API_KEY",
		KeyFormat:    KeyFormatMin32,
	},
	"deepseek": {
		Name:           "DeepSeek AI",
		BaseURL:        "https://api.deepseek.com/anthropic",
		Model:          "deepseek-v4-pro[1m]",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_DEFAULT_HAIKU_MODEL=deepseek-v4-flash",
			"CLAUDE_CODE_SUBAGENT_MODEL=deepseek-v4-flash",
			"CLAUDE_CODE_EFFORT_LEVEL=max",
			"API_TIMEOUT_MS=600000",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		},
		APIKeyEnvVar: "DEEPSEEK_API_KEY",
		KeyFormat:    KeyFormatMin32,
	},
	"anthropic": {
		Name:           "Anthropic",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "ANTHROPIC_API_KEY",
		KeyFormat:      KeyFormatAnthropic,
	},
	"openai": {
		Name:           "OpenAI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "OPENAI_API_KEY",
		KeyFormat:      KeyFormatOpenAI,
	},
	"google": {
		Name:           "Google",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "GEMINI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"mistral": {
		Name:           "Mistral",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "MISTRAL_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"groq": {
		Name:           "Groq",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "GROQ_API_KEY",
		KeyFormat:      KeyFormatGroq,
	},
	"cerebras": {
		Name:           "Cerebras",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "CEREBRAS_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"cloudflare-workers-ai": {
		Name:           "Cloudflare Workers AI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "CLOUDFLARE_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"xai": {
		Name:           "xAI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "XAI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"openrouter": {
		Name:           "OpenRouter",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "OPENROUTER_API_KEY",
		KeyFormat:      KeyFormatOpenRouter,
	},
	"vercel-ai-gateway": {
		Name:           "Vercel AI Gateway",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "AI_GATEWAY_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"opencode": {
		Name:           "OpenCode",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "OPENCODE_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"huggingface": {
		Name:           "Hugging Face",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "HF_TOKEN",
		KeyFormat:      KeyFormatMin32,
	},
	"fireworks": {
		Name:           "Fireworks",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "FIREWORKS_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"azure-openai-responses": {
		Name:           "Azure OpenAI",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "AZURE_OPENAI_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"minimax-cn": {
		Name:           "MiniMax (CN)",
		RequiresAPIKey: true,
		APIKeyEnvVar:   "MINIMAX_CN_API_KEY",
		KeyFormat:      KeyFormatMin32,
	},
	"custom": {
		Name:           "Custom Provider",
		BaseURL:        "",
		Model:          "",
		RequiresAPIKey: true,
		KeyFormat:      DefaultKeyFormat,
	},
}

// ProviderDefinition describes a built-in provider's display name, default
// base URL, model, environment variables, API key requirements, and key format.
type ProviderDefinition struct {
	Name           string
	BaseURL        string
	Model          string
	EnvVars        []string
	RequiresAPIKey bool
	APIKeyEnvVar   string
	KeyFormat      KeyFormat
}

// ValidateAPIKey checks the given key against this provider's key format rules.
func (d ProviderDefinition) ValidateAPIKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: API key cannot be empty or whitespace", d.Name))
	}

	if err := d.KeyFormat.validateForKey(key); err != nil {
		return errors.NewError(errors.ValidationError,
			fmt.Sprintf("%s: %s", d.Name, err))
	}

	return nil
}

// IsBuiltInProvider reports whether name is a recognized built-in provider.
func IsBuiltInProvider(name string) bool {
	_, ok := builtInProviders[name]

	return ok
}

// BuiltInProvider returns the definition for the named built-in provider.
func BuiltInProvider(name string) (ProviderDefinition, bool) {
	def, ok := builtInProviders[name]

	return def, ok
}

// providerOrder defines the canonical display order for providers.
// It must contain exactly the same keys as builtInProviders.
var providerOrder = []string{
	"zai", "minimax", "deepseek", "kimi",
	"anthropic", "openai", "google", "mistral",
	"groq", "cerebras", "cloudflare-workers-ai", "xai",
	"openrouter", "vercel-ai-gateway", "opencode", "huggingface",
	"fireworks", "azure-openai-responses", "minimax-cn",
	"custom",
}

// ProviderList returns the ordered list of built-in provider names.
// Entries not present in builtInProviders are silently excluded.
func ProviderList() []string {
	result := make([]string, 0, len(providerOrder))
	for _, name := range providerOrder {
		if _, ok := builtInProviders[name]; ok {
			result = append(result, name)
		}
	}

	return result
}

// RequiresAPIKey reports whether the named provider requires an API key.
func RequiresAPIKey(name string) bool {
	def, ok := builtInProviders[name]
	if !ok {
		return true
	}

	return def.RequiresAPIKey
}

// APIKeyEnvVarFor returns the environment variable name for the named
// provider's API key, if one is defined.
func APIKeyEnvVarFor(name string) (string, bool) {
	def, ok := builtInProviders[name]
	if !ok {
		return "", false
	}
	if def.APIKeyEnvVar == "" {
		return "", false
	}

	return def.APIKeyEnvVar, true
}
