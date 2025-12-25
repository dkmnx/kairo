package providers

var BuiltInProviders = map[string]ProviderDefinition{
	"anthropic": {
		Name:    "Native Anthropic",
		BaseURL: "",
		Model:   "",
	},
	"zai": {
		Name:    "Z.AI",
		BaseURL: "https://api.z.ai/api/anthropic",
		Model:   "glm-4.7",
		EnvVars: []string{"ANTHROPIC_DEFAULT_HAIKU_MODEL=glm-4.5-air"},
	},
	"minimax": {
		Name:    "MiniMax",
		BaseURL: "https://api.minimax.io/anthropic",
		Model:   "Minimax-M2.1",
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=120",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=24576",
		},
	},
	"kimi": {
		Name:    "Moonshot AI",
		BaseURL: "https://api.kimi.com/coding/",
		Model:   "kimi-for-coding",
		EnvVars: []string{
			"ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT=240",
			"ANTHROPIC_SMALL_FAST_MAX_TOKENS=200000",
		},
	},
	"deepseek": {
		Name:    "DeepSeek AI",
		BaseURL: "https://api.deepseek.com/anthropic",
		Model:   "deepseek-chat",
		EnvVars: []string{
			"API_TIMEOUT_MS=600000",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
		},
	},
	"custom": {
		Name:    "Custom Provider",
		BaseURL: "",
		Model:   "",
	},
}

type ProviderDefinition struct {
	Name    string
	BaseURL string
	Model   string
	EnvVars []string
}

func IsBuiltInProvider(name string) bool {
	_, ok := BuiltInProviders[name]
	return ok
}

func GetBuiltInProvider(name string) (ProviderDefinition, bool) {
	def, ok := BuiltInProviders[name]
	return def, ok
}

func GetProviderList() []string {
	providers := make([]string, 0, len(BuiltInProviders))
	for name := range BuiltInProviders {
		providers = append(providers, name)
	}
	return providers
}
