package providers

var BuiltInProviders = map[string]ProviderDefinition{
	"zai": {
		Name:           "Z.AI",
		BaseURL:        "https://api.z.ai/api/anthropic",
		Model:          "glm-5.1",
		RequiresAPIKey: true,
		EnvVars:        []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.7-flash"},
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

type ProviderDefinition struct {
	Name           string
	BaseURL        string
	Model          string
	EnvVars        []string
	RequiresAPIKey bool
}

func IsBuiltInProvider(name string) bool {
	_, ok := BuiltInProviders[name]
	return ok
}

func GetBuiltInProvider(name string) (ProviderDefinition, bool) {
	def, ok := BuiltInProviders[name]
	return def, ok
}

var providerOrder = []string{"zai", "minimax", "deepseek", "kimi"}

func GetProviderList() []string {
	return providerOrder
}

func RequiresAPIKey(name string) bool {
	def, ok := BuiltInProviders[name]
	if !ok {
		return true
	}
	return def.RequiresAPIKey
}
