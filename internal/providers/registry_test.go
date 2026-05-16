package providers

import (
	"sort"
	"testing"
)

func TestGetProviderList(t *testing.T) {
	providers := GetProviderList()

	expected := []string{
		"zai", "minimax", "deepseek", "kimi",
		"anthropic", "openai", "google", "mistral",
		"groq", "cerebras", "cloudflare-workers-ai", "xai",
		"openrouter", "vercel-ai-gateway", "opencode", "huggingface",
		"fireworks", "azure-openai-responses", "minimax-cn",
	}

	if len(providers) != len(expected) {
		t.Errorf("GetProviderList() returned %d providers, want %d", len(providers), len(expected))
	}

	sort.Strings(providers)
	sort.Strings(expected)

	for i, p := range providers {
		if p != expected[i] {
			t.Errorf("GetProviderList()[%d] = %q, want %q", i, p, expected[i])
		}
	}
}

func TestGetProviderListContainsAllBuiltIns(t *testing.T) {
	providers := GetProviderList()

	allBuiltins := []string{
		"zai", "minimax", "kimi", "deepseek",
		"anthropic", "openai", "google", "mistral",
		"groq", "cerebras", "cloudflare-workers-ai", "xai",
		"openrouter", "vercel-ai-gateway", "opencode", "huggingface",
		"fireworks", "azure-openai-responses", "minimax-cn",
	}

	for _, name := range allBuiltins {
		found := false
		for _, p := range providers {
			if p == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetProviderList() should contain %q", name)
		}
	}
}

func TestGetProviderListNoDuplicates(t *testing.T) {
	providers := GetProviderList()

	seen := make(map[string]bool)
	for _, p := range providers {
		if seen[p] {
			t.Errorf("GetProviderList() contains duplicate: %q", p)
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

func TestGetBuiltInProvider(t *testing.T) {
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
			def, got := GetBuiltInProvider(tt.provider)
			if got != tt.wantExists {
				t.Errorf("GetBuiltInProvider(%q) returned exists=%v, want %v", tt.provider, got, tt.wantExists)
				return
			}
			if tt.wantExists {
				if def.Name != tt.wantName {
					t.Errorf("GetBuiltInProvider(%q).Name = %q, want %q", tt.provider, def.Name, tt.wantName)
				}
				if def.BaseURL != tt.wantBaseURL {
					t.Errorf("GetBuiltInProvider(%q).BaseURL = %q, want %q", tt.provider, def.BaseURL, tt.wantBaseURL)
				}
				if def.Model != tt.wantModel {
					t.Errorf("GetBuiltInProvider(%q).Model = %q, want %q", tt.provider, def.Model, tt.wantModel)
				}
				if def.RequiresAPIKey != tt.wantRequires {
					t.Errorf("GetBuiltInProvider(%q).RequiresAPIKey = %v, want %v", tt.provider, def.RequiresAPIKey, tt.wantRequires)
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
			def, ok := GetBuiltInProvider(tt.provider)
			if !ok {
				t.Skip("provider not found")
			}

			if tt.wantVars == 0 {
				if len(def.EnvVars) != 0 {
					t.Errorf("GetBuiltInProvider(%q).EnvVars = %v, want empty", tt.provider, def.EnvVars)
				}
			} else {
				if len(def.EnvVars) != tt.wantVars {
					t.Errorf("GetBuiltInProvider(%q).EnvVars count = %d, want %d", tt.provider, len(def.EnvVars), tt.wantVars)
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
						t.Errorf("GetBuiltInProvider(%q).EnvVars = %v, want env var with prefix %q", tt.provider, def.EnvVars, tt.hasPrefix)
					}
				}
			}
		})
	}
}
