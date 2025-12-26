package providers

// BuiltInProviders contains the definitions of all supported built-in providers.
var BuiltInProviders = map[string]ProviderDefinition{
	"anthropic": {
		Name:           "Native Anthropic",
		BaseURL:        "",
		Model:          "",
		RequiresAPIKey: false,
	},
	"zai": {
		Name:           "Z.AI",
		BaseURL:        "https://api.z.ai/api/anthropic",
		Model:          "glm-4.7",
		RequiresAPIKey: true,
		EnvVars:        []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.5-air"},
	},
	"minimax": {
		Name:           "MiniMax",
		BaseURL:        "https://api.minimax.io/anthropic",
		Model:          "Minimax-M2.1",
		RequiresAPIKey: true,
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=120",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=24576",
		},
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
	},
	"deepseek": {
		Name:           "DeepSeek AI",
		BaseURL:        "https://api.deepseek.com/anthropic",
		Model:          "deepseek-chat",
		RequiresAPIKey: true,
		EnvVars: []string{
			"API_TIMEOUT_MS=600000",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		},
	},
	"custom": {
		Name:           "Custom Provider",
		BaseURL:        "",
		Model:          "",
		RequiresAPIKey: true,
	},
}

// ProviderDefinition contains the configuration for a provider.
type ProviderDefinition struct {
	Name           string
	BaseURL        string
	Model          string
	EnvVars        []string
	RequiresAPIKey bool
}

// IsBuiltInProvider returns true if the given name is a built-in provider.
func IsBuiltInProvider(name string) bool {
	_, ok := BuiltInProviders[name]
	return ok
}

// GetBuiltInProvider returns the provider definition and whether it exists.
func GetBuiltInProvider(name string) (ProviderDefinition, bool) {
	def, ok := BuiltInProviders[name]
	return def, ok
}

// GetProviderList returns a list of all built-in provider names.
func GetProviderList() []string {
	providers := make([]string, 0, len(BuiltInProviders))
	for name := range BuiltInProviders {
		providers = append(providers, name)
	}
	return providers
}

// RequiresAPIKey returns true if the provider requires an API key to configure.
func RequiresAPIKey(name string) bool {
	def, ok := BuiltInProviders[name]
	if !ok {
		return true
	}
	return def.RequiresAPIKey
}
