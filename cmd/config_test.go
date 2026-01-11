package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestProviderDefaults(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		wantDefaultURL   bool
		wantURL          string
		wantDefaultModel bool
		wantModel        string
	}{
		{
			name:             "anthropic has no defaults",
			provider:         "anthropic",
			wantDefaultURL:   false,
			wantDefaultModel: false,
		},
		{
			name:             "zai has default URL and model",
			provider:         "zai",
			wantDefaultURL:   true,
			wantURL:          "https://api.z.ai/api/anthropic",
			wantDefaultModel: true,
			wantModel:        "glm-4.7",
		},
		{
			name:             "minimax has default URL and model",
			provider:         "minimax",
			wantDefaultURL:   true,
			wantURL:          "https://api.minimax.io/anthropic",
			wantDefaultModel: true,
			wantModel:        "Minimax-M2.1",
		},
		{
			name:             "kimi has default URL and model",
			provider:         "kimi",
			wantDefaultURL:   true,
			wantURL:          "https://api.kimi.com/coding/",
			wantDefaultModel: true,
			wantModel:        "kimi-for-coding",
		},
		{
			name:             "deepseek has default URL and model",
			provider:         "deepseek",
			wantDefaultURL:   true,
			wantURL:          "https://api.deepseek.com/anthropic",
			wantDefaultModel: true,
			wantModel:        "deepseek-chat",
		},
		{
			name:             "custom has no defaults",
			provider:         "custom",
			wantDefaultURL:   false,
			wantDefaultModel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := providers.GetBuiltInProvider(tt.provider)
			if !ok {
				t.Fatalf("GetBuiltInProvider(%q) = false, want true", tt.provider)
			}

			if tt.wantDefaultURL {
				if def.BaseURL == "" {
					t.Errorf("GetBuiltInProvider(%q).BaseURL = empty, want %q", tt.provider, tt.wantURL)
				}
			} else {
				if def.BaseURL != "" {
					t.Errorf("GetBuiltInProvider(%q).BaseURL = %q, want empty", tt.provider, def.BaseURL)
				}
			}

			if tt.wantDefaultModel {
				if def.Model == "" {
					t.Errorf("GetBuiltInProvider(%q).Model = empty, want %q", tt.provider, tt.wantModel)
				}
			} else {
				if def.Model != "" {
					t.Errorf("GetBuiltInProvider(%q).Model = %q, want empty", tt.provider, def.Model)
				}
			}
		})
	}
}

func TestProviderEnvVars(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		wantEnvVars  bool
		envVarPrefix string
	}{
		{
			name:        "anthropic has no env vars",
			provider:    "anthropic",
			wantEnvVars: false,
		},
		{
			name:         "zai has env vars",
			provider:     "zai",
			wantEnvVars:  true,
			envVarPrefix: "ANTHROPIC_DEFAULT_HAIKU_MODEL",
		},
		{
			name:         "minimax has env vars",
			provider:     "minimax",
			wantEnvVars:  true,
			envVarPrefix: "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT",
		},
		{
			name:         "kimi has env vars",
			provider:     "kimi",
			wantEnvVars:  true,
			envVarPrefix: "ANTHROPIC_SMALL_FAST_MODEL_TIMEOUT",
		},
		{
			name:         "deepseek has env vars",
			provider:     "deepseek",
			wantEnvVars:  true,
			envVarPrefix: "API_TIMEOUT_MS",
		},
		{
			name:        "custom has no env vars",
			provider:    "custom",
			wantEnvVars: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, ok := providers.GetBuiltInProvider(tt.provider)
			if !ok {
				t.Fatalf("GetBuiltInProvider(%q) = false, want true", tt.provider)
			}

			if tt.wantEnvVars {
				if len(def.EnvVars) == 0 {
					t.Errorf("GetBuiltInProvider(%q).EnvVars = empty, want non-empty", tt.provider)
				}
				if tt.envVarPrefix != "" {
					found := false
					for _, env := range def.EnvVars {
						if len(env) >= len(tt.envVarPrefix) && env[:len(tt.envVarPrefix)] == tt.envVarPrefix {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("GetBuiltInProvider(%q).EnvVars = %v, want env var with prefix %q", tt.provider, def.EnvVars, tt.envVarPrefix)
					}
				}
			} else {
				if len(def.EnvVars) > 0 {
					t.Errorf("GetBuiltInProvider(%q).EnvVars = %v, want empty", tt.provider, def.EnvVars)
				}
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
		{"anthropic is builtin", "anthropic", true},
		{"zai is builtin", "zai", true},
		{"minimax is builtin", "minimax", true},
		{"kimi is builtin", "kimi", true},
		{"deepseek is builtin", "deepseek", true},
		{"custom is builtin", "custom", true},
		{"unknown provider", "unknown", false},
		{"empty provider", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providers.IsBuiltInProvider(tt.provider)
			if got != tt.want {
				t.Errorf("IsBuiltInProvider(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestProviderConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Providers: map[string]config.Provider{
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
			},
		},
		DefaultProvider: "zai",
	}

	if err := config.SaveConfig(tmpDir, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loadedCfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	zaiProvider, ok := loadedCfg.Providers["zai"]
	if !ok {
		t.Fatal("zai provider not found in loaded config")
	}
	if zaiProvider.BaseURL != "https://api.z.ai/api/anthropic" {
		t.Errorf("zai BaseURL = %q, want %q", zaiProvider.BaseURL, "https://api.z.ai/api/anthropic")
	}
	if zaiProvider.Model != "glm-4.7" {
		t.Errorf("zai Model = %q, want %q", zaiProvider.Model, "glm-4.7")
	}

	minimaxProvider, ok := loadedCfg.Providers["minimax"]
	if !ok {
		t.Fatal("minimax provider not found in loaded config")
	}
	if minimaxProvider.BaseURL != "https://api.minimax.io/anthropic" {
		t.Errorf("minimax BaseURL = %q, want %q", minimaxProvider.BaseURL, "https://api.minimax.io/anthropic")
	}

	if loadedCfg.DefaultProvider != "zai" {
		t.Errorf("DefaultProvider = %q, want %q", loadedCfg.DefaultProvider, "zai")
	}
}

func TestGetConfigDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot find home directory")
	}

	var expectedDir string
	if runtime.GOOS == "windows" {
		expectedDir = filepath.Join(home, "AppData", "Roaming", "kairo")
	} else {
		expectedDir = filepath.Join(home, ".config", "kairo")
	}
	dir := getConfigDir()
	if dir != expectedDir {
		t.Errorf("getConfigDir() = %q, want %q", dir, expectedDir)
	}
}
