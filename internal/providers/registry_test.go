package providers

import (
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestProviderList(t *testing.T) {
	providers := ProviderList()

	expected := []string{
		"zai", "minimax", "deepseek", "kimi",
		"anthropic", "openai", "google", "mistral",
		"groq", "cerebras", "cloudflare-workers-ai", "xai",
		"openrouter", "vercel-ai-gateway", "opencode", "huggingface",
		"fireworks", "azure-openai-responses", "minimax-cn",
		"custom",
	}

	if len(providers) != len(expected) {
		t.Errorf("ProviderList() returned %d providers, want %d", len(providers), len(expected))
	}

	sort.Strings(providers)
	sort.Strings(expected)

	for i, p := range providers {
		if p != expected[i] {
			t.Errorf("ProviderList()[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestProviderListContainsAllBuiltIns(t *testing.T) {
	providers := ProviderList()

	allBuiltins := []string{
		"zai", "minimax", "kimi", "deepseek",
		"anthropic", "openai", "google", "mistral",
		"groq", "cerebras", "cloudflare-workers-ai", "xai",
		"openrouter", "vercel-ai-gateway", "opencode", "huggingface",
		"fireworks", "azure-openai-responses", "minimax-cn",
		"custom",
	}

	for _, name := range allBuiltins {
		if !slices.Contains(providers, name) {
			t.Errorf("ProviderList() should contain %q", name)
		}
	}
}

func TestProviderListNoDuplicates(t *testing.T) {
	providers := ProviderList()

	seen := make(map[string]bool)
	for _, p := range providers {
		if seen[p] {
			t.Errorf("ProviderList() contains duplicate: %q", p)
		}
		seen[p] = true
	}
}

func TestRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"zai requires API key", "zai", true},
		{"minimax requires API key", "minimax", true},
		{"kimi requires API key", "kimi", true},
		{"deepseek requires API key", "deepseek", true},
		{"anthropic requires API key", "anthropic", true},
		{"openai requires API key", "openai", true},
		{"custom requires API key", "custom", true},
		{"unknown provider defaults to true", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiresAPIKey(tt.provider)
			if got != tt.want {
				t.Errorf("RequiresAPIKey(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestProviderRegistry_BuiltInProvider(t *testing.T) {
	r := NewRegistry()

	def, ok := r.BuiltInProvider("zai")
	if !ok {
		t.Fatal("expected zai in registry")
	}
	if def.Name != "Z.AI" {
		t.Errorf("Z.AI name = %q", def.Name)
	}
}

func TestProviderRegistry_IsBuiltInProvider(t *testing.T) {
	r := NewRegistry()

	if !r.IsBuiltInProvider("anthropic") {
		t.Error("anthropic should be built-in")
	}
	if r.IsBuiltInProvider("nonexistent") {
		t.Error("nonexistent should not be built-in")
	}
}

func TestProviderRegistry_RegisterCustom(t *testing.T) {
	r := NewRegistry()

	custom := map[string]CustomProviderDefinition{
		"my-llm": {
			Name:           "My LLM",
			BaseURL:        "https://api.example.com",
			Model:          "custom-model",
			RequiresAPIKey: true,
			APIKeyEnvVar:   "MY_LLM_API_KEY",
			MinKeyLength:   32,
		},
	}
	r.RegisterCustom(custom)

	def, ok := r.BuiltInProvider("my-llm")
	if !ok {
		t.Fatal("expected my-llm in registry")
	}
	if def.Name != "My LLM" {
		t.Errorf("Name = %q", def.Name)
	}
	if !r.IsBuiltInProvider("my-llm") {
		t.Error("my-llm should be recognized as built-in")
	}
}

func TestProviderRegistry_RegisterCustomOverridesBuiltIn(t *testing.T) {
	r := NewRegistry()

	custom := map[string]CustomProviderDefinition{
		"zai": {
			Name:    "Custom ZAI",
			BaseURL: "https://custom.z.ai",
			Model:   "custom-model",
		},
	}
	r.RegisterCustom(custom)

	def, ok := r.BuiltInProvider("zai")
	if !ok {
		t.Fatal("expected zai in registry")
	}
	if def.Name != "Custom ZAI" {
		t.Errorf("Name = %q, want Custom ZAI", def.Name)
	}
	if def.BaseURL != "https://custom.z.ai" {
		t.Errorf("BaseURL = %q", def.BaseURL)
	}
}

func TestProviderRegistry_ProviderList(t *testing.T) {
	r := NewRegistry()

	custom := map[string]CustomProviderDefinition{
		"my-llm": {Name: "My LLM"},
	}
	r.RegisterCustom(custom)

	names := r.ProviderList()
	foundCustom := false
	for _, n := range names {
		if n == "my-llm" {
			foundCustom = true
		}
	}
	if !foundCustom {
		t.Error("ProviderList missing custom provider")
	}
}

func TestProviderRegistry_ClearCustom(t *testing.T) {
	r := NewRegistry()

	r.RegisterCustom(map[string]CustomProviderDefinition{
		"temp": {Name: "Temp"},
	})
	if !r.IsBuiltInProvider("temp") {
		t.Fatal("temp should be registered")
	}

	r.ClearCustom()
	if r.IsBuiltInProvider("temp") {
		t.Error("temp should be removed after ClearCustom")
	}
}

func TestProviderRegistry_RequiresAPIKeyCustom(t *testing.T) {
	r := NewRegistry()

	r.RegisterCustom(map[string]CustomProviderDefinition{
		"no-key": {
			Name:           "No Key",
			RequiresAPIKey: false,
		},
	})

	if r.RequiresAPIKey("no-key") {
		t.Error("no-key should not require API key")
	}
	if !r.RequiresAPIKey("unknown") {
		t.Error("unknown providers should require API key by default")
	}
}

func TestProviderRegistry_DefaultRegistry(t *testing.T) {
	def, ok := DefaultRegistry.BuiltInProvider("zai")
	if !ok {
		t.Fatal("zai should exist in DefaultRegistry")
	}
	if def.Name != "Z.AI" {
		t.Errorf("Name = %q", def.Name)
	}
}

func TestIsBuiltInProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"zai is built-in", "zai", true},
		{"minimax is built-in", "minimax", true},
		{"kimi is built-in", "kimi", true},
		{"deepseek is built-in", "deepseek", true},
		{"anthropic is built-in", "anthropic", true},
		{"openai is built-in", "openai", true},
		{"google is built-in", "google", true},
		{"custom is built-in", "custom", true},
		{"empty string is not built-in", "", false},
		{"unknown provider is not built-in", "unknown", false},
		{"case sensitive - ANTHROPIC is not built-in", "ANTHROPIC", false},
		{"typo is not built-in", "anthropicx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBuiltInProvider(tt.provider)
			if got != tt.want {
				t.Errorf("IsBuiltInProvider(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestBuiltInProvider(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantExists   bool
		wantName     string
		wantBaseURL  string
		wantModel    string
		wantRequires bool
	}{
		{
			name:         "zai returns correct definition",
			provider:     "zai",
			wantExists:   true,
			wantName:     "Z.AI",
			wantBaseURL:  "https://api.z.ai/api/anthropic",
			wantModel:    "glm-5.1",
			wantRequires: true,
		},
		{
			name:         "minimax returns correct definition",
			provider:     "minimax",
			wantExists:   true,
			wantName:     "MiniMax",
			wantBaseURL:  "https://api.minimax.io/anthropic",
			wantModel:    "MiniMax-M2.7",
			wantRequires: true,
		},
		{
			name:         "kimi returns correct definition",
			provider:     "kimi",
			wantExists:   true,
			wantName:     "Moonshot AI",
			wantBaseURL:  "https://api.kimi.com/coding/",
			wantModel:    "kimi-for-coding",
			wantRequires: true,
		},
		{
			name:         "deepseek returns correct definition",
			provider:     "deepseek",
			wantExists:   true,
			wantName:     "DeepSeek AI",
			wantBaseURL:  "https://api.deepseek.com/anthropic",
			wantModel:    "deepseek-v4-pro[1m]",
			wantRequires: true,
		},
		{
			name:         "anthropic returns correct definition",
			provider:     "anthropic",
			wantExists:   true,
			wantName:     "Anthropic",
			wantBaseURL:  "",
			wantModel:    "",
			wantRequires: true,
		},
		{
			name:         "openai returns correct definition",
			provider:     "openai",
			wantExists:   true,
			wantName:     "OpenAI",
			wantBaseURL:  "",
			wantModel:    "",
			wantRequires: true,
		},
		{
			name:         "custom returns correct definition",
			provider:     "custom",
			wantExists:   true,
			wantName:     "Custom Provider",
			wantBaseURL:  "",
			wantModel:    "",
			wantRequires: true,
		},
		{
			name:         "unknown provider returns false",
			provider:     "unknown",
			wantExists:   false,
			wantName:     "",
			wantBaseURL:  "",
			wantModel:    "",
			wantRequires: true,
		},
		{
			name:         "empty provider returns false",
			provider:     "",
			wantExists:   false,
			wantName:     "",
			wantBaseURL:  "",
			wantModel:    "",
			wantRequires: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, got := BuiltInProvider(tt.provider)
			if got != tt.wantExists {
				t.Errorf("BuiltInProvider(%q) returned exists=%v, want %v", tt.provider, got, tt.wantExists)
				return
			}
			if tt.wantExists {
				if def.Name != tt.wantName {
					t.Errorf("BuiltInProvider(%q).Name = %q, want %q", tt.provider, def.Name, tt.wantName)
				}
				if def.BaseURL != tt.wantBaseURL {
					t.Errorf("BuiltInProvider(%q).BaseURL = %q, want %q", tt.provider, def.BaseURL, tt.wantBaseURL)
				}
				if def.Model != tt.wantModel {
					t.Errorf("BuiltInProvider(%q).Model = %q, want %q", tt.provider, def.Model, tt.wantModel)
				}
				if def.RequiresAPIKey != tt.wantRequires {
					t.Errorf("BuiltInProvider(%q).RequiresAPIKey = %v, want %v", tt.provider, def.RequiresAPIKey, tt.wantRequires)
				}
			}
		})
	}
}

func TestAPIKeyEnvVarFor(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantVar  string
		wantOk   bool
	}{
		{"zai has API key env var", "zai", "ZAI_API_KEY", true},
		{"minimax has API key env var", "minimax", "MINIMAX_API_KEY", true},
		{"kimi has API key env var", "kimi", "KIMI_API_KEY", true},
		{"deepseek has API key env var", "deepseek", "DEEPSEEK_API_KEY", true},
		{"anthropic has API key env var", "anthropic", "ANTHROPIC_API_KEY", true},
		{"openai has API key env var", "openai", "OPENAI_API_KEY", true},
		{"google has API key env var", "google", "GEMINI_API_KEY", true},
		{"mistral has API key env var", "mistral", "MISTRAL_API_KEY", true},
		{"groq has API key env var", "groq", "GROQ_API_KEY", true},
		{"cerebras has API key env var", "cerebras", "CEREBRAS_API_KEY", true},
		{"cloudflare-workers-ai has API key env var", "cloudflare-workers-ai", "CLOUDFLARE_API_KEY", true},
		{"xai has API key env var", "xai", "XAI_API_KEY", true},
		{"openrouter has API key env var", "openrouter", "OPENROUTER_API_KEY", true},
		{"vercel-ai-gateway has API key env var", "vercel-ai-gateway", "AI_GATEWAY_API_KEY", true},
		{"opencode has API key env var", "opencode", "OPENCODE_API_KEY", true},
		{"huggingface has API key env var", "huggingface", "HF_TOKEN", true},
		{"fireworks has API key env var", "fireworks", "FIREWORKS_API_KEY", true},
		{"azure-openai-responses has API key env var", "azure-openai-responses", "AZURE_OPENAI_API_KEY", true},
		{"minimax-cn has API key env var", "minimax-cn", "MINIMAX_CN_API_KEY", true},
		{"custom has no API key env var", "custom", "", false},
		{"unknown provider returns false", "unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVar, gotOk := APIKeyEnvVarFor(tt.provider)
			if gotOk != tt.wantOk {
				t.Errorf("APIKeyEnvVarFor(%q) ok = %v, want %v", tt.provider, gotOk, tt.wantOk)
			}
			if gotVar != tt.wantVar {
				t.Errorf("APIKeyEnvVarFor(%q) var = %q, want %q", tt.provider, gotVar, tt.wantVar)
			}
		})
	}
}

func TestComputeProviderOrder_ContainsAllBuiltIns(t *testing.T) {
	order := computeProviderOrder()

	if len(order) != len(builtInProviders) {
		t.Errorf("providerOrder has %d entries, builtInProviders has %d", len(order), len(builtInProviders))
	}

	seen := make(map[string]bool, len(order))
	for _, name := range order {
		if seen[name] {
			t.Errorf("providerOrder contains duplicate: %q", name)
		}
		seen[name] = true

		if _, ok := builtInProviders[name]; !ok {
			t.Errorf("providerOrder contains %q which is not in builtInProviders", name)
		}
	}

	for name := range builtInProviders {
		if !seen[name] {
			t.Errorf("builtInProviders contains %q which is missing from providerOrder", name)
		}
	}
}

func TestProviderPriority_AllEntriesExistInBuiltInProviders(t *testing.T) {
	for _, name := range providerPriority {
		if _, ok := builtInProviders[name]; !ok {
			t.Errorf("providerPriority contains %q which is not in builtInProviders — remove stale entry or add the provider", name)
		}
	}
}

func TestComputeProviderOrder_PriorityFirst(t *testing.T) {
	order := computeProviderOrder()

	for i, prioName := range providerPriority {
		if i >= len(order) {
			break
		}
		if order[i] != prioName {
			t.Errorf("providerOrder[%d] = %q, want priority entry %q", i, order[i], prioName)
		}
	}
}

func TestBuiltInProviderEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		wantVars  int
		hasPrefix string
	}{
		{"zai has env vars", "zai", 1, "ANTHROPIC_DEFAULT_HAIKU_MODEL"},
		{"minimax has env vars", "minimax", 2, "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT"},
		{"kimi has env vars", "kimi", 2, "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT"},
		{"deepseek has env vars", "deepseek", 5, "ANTHROPIC_DEFAULT_HAIKU_MODEL"},
		{"anthropic has no env vars", "anthropic", 0, ""},
		{"openai has no env vars", "openai", 0, ""},
		{"custom has no env vars", "custom", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := BuiltInProvider(tt.provider)
			if !ok {
				t.Skip("provider not found")
			}

			if tt.wantVars == 0 {
				if len(def.EnvVars) != 0 {
					t.Errorf("BuiltInProvider(%q).EnvVars = %v, want empty", tt.provider, def.EnvVars)
				}
			} else {
				if len(def.EnvVars) != tt.wantVars {
					t.Errorf("BuiltInProvider(%q).EnvVars count = %d, want %d", tt.provider, len(def.EnvVars), tt.wantVars)
				}
				if tt.hasPrefix != "" {
					found := false
					for _, env := range def.EnvVars {
						if len(env) >= len(tt.hasPrefix) && env[:len(tt.hasPrefix)] == tt.hasPrefix {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("BuiltInProvider(%q).EnvVars = %v, want env var with prefix %q", tt.provider, def.EnvVars, tt.hasPrefix)
					}
				}
			}
		})
	}
}

func TestKeyFormat_validateForKey(t *testing.T) {
	t.Run("empty key passes", func(t *testing.T) {
		err := KeyFormatMin32.validateForKey("")
		if err != nil {
			t.Errorf("empty key should pass: %v", err)
		}
	})

	t.Run("whitespace only passes", func(t *testing.T) {
		err := KeyFormatMin32.validateForKey("   ")
		if err != nil {
			t.Errorf("whitespace key should pass: %v", err)
		}
	})

	t.Run("too short", func(t *testing.T) {
		err := KeyFormatMin32.validateForKey("short")
		if err == nil {
			t.Error("expected error for short key")
		}
	})

	t.Run("at minimum length", func(t *testing.T) {
		err := KeyFormatMin32.validateForKey(validKey)
		if err != nil {
			t.Errorf("key at minimum length should pass: %v", err)
		}

		err2 := KeyFormatMin32.validateForKey(tooShortKey)
		if err2 == nil {
			t.Error("key below minimum length should fail")
		}
	})

	t.Run("prefix mismatch", func(t *testing.T) {
		err := KeyFormatAnthropic.validateForKey(validKey)
		if err == nil {
			t.Error("expected error for key without sk-ant- prefix")
		}
	})

	t.Run("prefix match passes", func(t *testing.T) {
		err := KeyFormatAnthropic.validateForKey(validAnthropicKey)
		if err != nil {
			t.Errorf("key with correct prefix should pass: %v", err)
		}
	})

	t.Run("pattern mismatch", func(t *testing.T) {
		kf := KeyFormat{Pattern: `^[a-f0-9]+$`, compiled: nil}
		// trigger regex compilation
		err := kf.validateForKey("contains-upper-and-dashes")
		if err == nil {
			t.Error("expected error for key not matching hex pattern")
		}
		if !strings.Contains(err.Error(), "API key format") {
			t.Errorf("error should mention format: %v", err)
		}
	})

	t.Run("pattern match passes", func(t *testing.T) {
		kf := KeyFormat{Pattern: `^[a-f0-9]{32,}$`, compiled: nil}
		hexKey := strings.Repeat("a", 32)
		err := kf.validateForKey(hexKey)
		if err != nil {
			t.Errorf("key matching hex pattern should pass: %v", err)
		}
	})

	t.Run("invalid pattern causes error", func(t *testing.T) {
		kf := KeyFormat{Pattern: `[invalid(regex`}
		err := kf.validateForKey(validKey)
		if err == nil {
			t.Error("expected error for invalid pattern")
		}
		// the compiled pattern stays nil since regexp.Compile failed
		// running validateForKey again should re-attempt compilation and fail again
		err2 := kf.validateForKey(validKey)
		if err2 == nil {
			t.Error("expected same error on second attempt")
		}
	})

	t.Run("compiled regex reused on second call", func(t *testing.T) {
		kf := KeyFormat{Pattern: `^[a-z]+$`, compiled: nil}
		err := kf.validateForKey("validlowercase")
		if err != nil {
			t.Fatalf("first call should pass: %v", err)
		}
		if kf.compiled == nil {
			t.Fatal("regex should be compiled after first call")
		}
		err2 := kf.validateForKey("validlowercase")
		if err2 != nil {
			t.Errorf("second call should also pass: %v", err2)
		}
	})

	t.Run("prefix satisfied but too short", func(t *testing.T) {
		err := KeyFormatAnthropic.validateForKey("sk-ant-short")
		if err == nil {
			t.Error("expected error for anthropic-prefixed short key")
		}
		if !strings.Contains(err.Error(), "minimum") {
			t.Errorf("error should mention minimum length: %v", err)
		}
	})
}

func TestProviderDefinition_ValidateAPIKey(t *testing.T) {
	t.Run("empty key", func(t *testing.T) {
		def := BuiltInProviderNoMap("test")
		err := def.ValidateAPIKey("")
		if err == nil {
			t.Error("expected error for empty API key")
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		def := BuiltInProviderNoMap("test")
		err := def.ValidateAPIKey("  ")
		if err == nil {
			t.Error("expected error for whitespace-only API key")
		}
	})

	t.Run("valid key passes", func(t *testing.T) {
		def := BuiltInProviderNoMap("test")
		err := def.ValidateAPIKey(validKey)
		if err != nil {
			t.Errorf("valid key should pass: %v", err)
		}
	})

	t.Run("anthropic valid", func(t *testing.T) {
		def, ok := BuiltInProvider("anthropic")
		if !ok {
			t.Fatal("anthropic provider not found")
		}
		err := def.ValidateAPIKey(validAnthropicKey)
		if err != nil {
			t.Errorf("valid anthropic key should pass: %v", err)
		}
	})

	t.Run("anthropic wrong prefix", func(t *testing.T) {
		def, ok := BuiltInProvider("anthropic")
		if !ok {
			t.Fatal("anthropic provider not found")
		}
		err := def.ValidateAPIKey(validKey)
		if err == nil {
			t.Error("expected error for non-anthropic prefixed key")
		}
	})

	t.Run("anthropic too short", func(t *testing.T) {
		def, ok := BuiltInProvider("anthropic")
		if !ok {
			t.Fatal("anthropic provider not found")
		}
		err := def.ValidateAPIKey("sk-ant-short")
		if err == nil {
			t.Error("expected error for short anthropic key")
		}
	})

	t.Run("openai valid", func(t *testing.T) {
		def, ok := BuiltInProvider("openai")
		if !ok {
			t.Fatal("openai provider not found")
		}
		err := def.ValidateAPIKey(validOpenAIKey)
		if err != nil {
			t.Errorf("valid openai key should pass: %v", err)
		}
	})

	t.Run("groq valid", func(t *testing.T) {
		def, ok := BuiltInProvider("groq")
		if !ok {
			t.Fatal("groq provider not found")
		}
		err := def.ValidateAPIKey(validGroqKey)
		if err != nil {
			t.Errorf("valid groq key should pass: %v", err)
		}
	})

	t.Run("openrouter valid", func(t *testing.T) {
		def, ok := BuiltInProvider("openrouter")
		if !ok {
			t.Fatal("openrouter provider not found")
		}
		err := def.ValidateAPIKey(validOpenRouterKey)
		if err != nil {
			t.Errorf("valid openrouter key should pass: %v", err)
		}
	})

	t.Run("custom with default key format", func(t *testing.T) {
		def, ok := BuiltInProvider("custom")
		if !ok {
			t.Fatal("custom provider not found")
		}
		err := def.ValidateAPIKey(tooShortKey)
		if err == nil {
			t.Error("expected error for short key on custom (min 20)")
		}
		validCustomKey := strings.Repeat("x", 20)
		err = def.ValidateAPIKey(validCustomKey)
		if err != nil {
			t.Errorf("valid custom key should pass: %v", err)
		}
	})

	t.Run("zai valid", func(t *testing.T) {
		def, ok := BuiltInProvider("zai")
		if !ok {
			t.Fatal("zai provider not found")
		}
		err := def.ValidateAPIKey(validZAIKey)
		if err != nil {
			t.Errorf("zai key %q: %v", validZAIKey, err)
		}
	})
}

func BuiltInProviderNoMap(name string) ProviderDefinition {
	return ProviderDefinition{
		Name:      name,
		KeyFormat: KeyFormatMin32,
	}
}

var (
	validKey           = strings.Repeat("x", 32)
	tooShortKey        = strings.Repeat("x", 15)
	validAnthropicKey  = "sk-ant-" + strings.Repeat("x", 26)
	validOpenAIKey     = "sk-" + strings.Repeat("x", 30)
	validGroqKey       = "gsk_" + strings.Repeat("x", 29)
	validOpenRouterKey = "sk-or-" + strings.Repeat("x", 27)
	validZAIKey        = strings.Repeat("y", 32)
)
