package providers

import (
	"sort"
	"testing"
)

func TestGetProviderList(t *testing.T) {
	providers := GetProviderList()

	expected := []string{"anthropic", "custom", "deepseek", "kimi", "minimax", "zai"}

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

	allBuiltins := []string{"anthropic", "zai", "minimax", "kimi", "deepseek", "custom"}

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
		{"anthropic does not require API key", "anthropic", false},
		{"zai requires API key", "zai", true},
		{"minimax requires API key", "minimax", true},
		{"kimi requires API key", "kimi", true},
		{"deepseek requires API key", "deepseek", true},
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
		{"anthropic is built-in", "anthropic", true},
		{"zai is built-in", "zai", true},
		{"minimax is built-in", "minimax", true},
		{"kimi is built-in", "kimi", true},
		{"deepseek is built-in", "deepseek", true},
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
			name:         "anthropic returns correct definition",
			provider:     "anthropic",
			wantExists:   true,
			wantName:     "Native Anthropic",
			wantBaseURL:  "",
			wantModel:    "",
			wantRequires: false,
		},
		{
			name:         "zai returns correct definition",
			provider:     "zai",
			wantExists:   true,
			wantName:     "Z.AI",
			wantBaseURL:  "https://api.z.ai/api/anthropic",
			wantModel:    "glm-4.7",
			wantRequires: true,
		},
		{
			name:         "minimax returns correct definition",
			provider:     "minimax",
			wantExists:   true,
			wantName:     "MiniMax",
			wantBaseURL:  "https://api.minimax.io/anthropic",
			wantModel:    "Minimax-M2.1",
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
			wantModel:    "deepseek-chat",
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

func TestBuiltInProviderEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		wantVars  int
		hasPrefix string
	}{
		{"anthropic has no env vars", "anthropic", 0, ""},
		{"zai has env vars", "zai", 1, "ANTHROPIC_DEFAULT_HAIKU_MODEL"},
		{"minimax has env vars", "minimax", 2, "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT"},
		{"kimi has env vars", "kimi", 2, "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT"},
		{"deepseek has env vars", "deepseek", 2, "API_TIMEOUT_MS"},
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
