package cmd

import (
	"testing"

	"github.com/dkmnx/kairo/internal/harness"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestRequiresAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{"built-in provider with key", "zai", true},
		{"built-in anthropic", "anthropic", true},
		{"unknown provider defaults to true", "unknown-provider", true},
		{"empty provider defaults to true", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providers.RequiresAPIKey(tt.provider)
			if got != tt.want {
				t.Errorf("providers.RequiresAPIKey(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestBuildPiEnvVars(t *testing.T) {
	envVars := harness.PiEnvVars("zai", "glm-5.1")

	hasProvider := false
	hasModel := false
	for _, v := range envVars {
		if v == "PI_PROVIDER=zai" {
			hasProvider = true
		}
		if v == "PI_MODEL=glm-5.1" {
			hasModel = true
		}
	}

	if !hasProvider {
		t.Error("missing PI_PROVIDER")
	}
	if !hasModel {
		t.Error("missing PI_MODEL")
	}
}

func TestPiAPIKeyEnvVarMapping(t *testing.T) {
	tests := []struct {
		provider string
		envVar   string
		ok       bool
	}{
		{"zai", "ZAI_API_KEY", true},
		{"minimax", "MINIMAX_API_KEY", true},
		{"deepseek", "DEEPSEEK_API_KEY", true},
		{"kimi", "KIMI_API_KEY", true},
		{"unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			envVar, ok := providers.APIKeyEnvVarFor(tt.provider)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if envVar != tt.envVar {
				t.Errorf("envVar = %q, want %q", envVar, tt.envVar)
			}
		})
	}
}

func TestHarnessAPIKeyEnvVar(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"zai", "ZAI_API_KEY"},
		{"minimax", "MINIMAX_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			var got string
			if envVar, ok := providers.APIKeyEnvVarFor(tt.provider); ok {
				got = envVar
			} else {
				got = harness.APIKeyEnvVar(tt.provider)
			}
			if got != tt.want {
				t.Errorf("HarnessAPIKeyEnvVar(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestYoloModeFlag(t *testing.T) {
	tests := []struct {
		name    string
		harness string
		want    string
	}{
		{"claude", harness.Claude, "--dangerously-skip-permissions"},
		{"qwen", harness.Qwen, "--yolo"},
		{"pi", harness.Pi, ""},
		{"crush", harness.Crush, "--yolo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := harness.YoloFlag(tt.harness)
			if got != tt.want {
				t.Errorf("harness.YoloFlag(%q) = %q, want %q", tt.harness, got, tt.want)
			}
		})
	}
}
