package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
	"github.com/dkmnx/kairo/internal/ui"
)

func providerStatusIcon(cfg *config.Config, secrets map[string]string, provider string) string {
	if !providers.RequiresAPIKey(provider) {
		if _, exists := cfg.Providers[provider]; exists {
			return ui.Green + "[x]" + ui.Reset
		}
		return "  "
	}

	apiKeyKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(provider))
	for k := range secrets {
		if k == apiKeyKey {
			return ui.Green + "[x]" + ui.Reset
		}
	}
	return "  "
}

func TestPrintBanner(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		provider config.Provider
		harness  string
		wantSub  string
	}{
		{
			name:    "banner with version model and provider",
			version: "v0.1.0",
			provider: config.Provider{
				Model: "claude-sonnet-4-20250514",
				Name:  "MiniMax",
			},
			harness: "claude",
			wantSub: "v0.1.0 · claude-sonnet-4-20250514 · MiniMax",
		},
		{
			name:    "banner with custom provider and model",
			version: "vdev",
			provider: config.Provider{
				Model: "custom-model",
				Name:  "Custom Provider",
			},
			harness: "claude",
			wantSub: "vdev · custom-model · Custom Provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			done := make(chan struct{})
			go func() {
				ui.PrintBanner(ui.Banner{
					Version:      tt.version,
					ModelName:    tt.provider.Model,
					ProviderName: tt.provider.Name,
					Harness:      tt.harness,
				})
				w.Close()
				close(done)
			}()

			<-done

			os.Stdout = oldStdout
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Logf("Warning: io.Copy failed: %v", err)
			}
			r.Close()
			output := buf.String()
			if !strings.Contains(output, tt.wantSub) {
				t.Errorf("banner output does not contain expected substring %q, got: %q", tt.wantSub, output)
			}
		})
	}
}

func TestProviderStatusIcon(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.Provider{
			"zai": {Name: "Z.AI"},
		},
	}

	tests := []struct {
		name     string
		provider string
		secrets  map[string]string
		cfg      *config.Config
		wantIcon string
	}{
		{
			name:     "zai configured",
			provider: "zai",
			secrets:  map[string]string{"ZAI_API_KEY": "key"},
			cfg:      cfg,
			wantIcon: "[x]",
		},
		{
			name:     "zai not configured",
			provider: "zai",
			secrets:  map[string]string{},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			wantIcon: "  ",
		},
		{
			name:     "minimax with key",
			provider: "minimax",
			secrets:  map[string]string{"MINIMAX_API_KEY": "key"},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			wantIcon: "[x]",
		},
		{
			name:     "minimax without key",
			provider: "minimax",
			secrets:  map[string]string{},
			cfg:      &config.Config{Providers: map[string]config.Provider{}},
			wantIcon: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerStatusIcon(tt.cfg, tt.secrets, tt.provider)
			if !strings.Contains(got, "[x]") && tt.wantIcon == "[x]" {
				t.Errorf("providerStatusIcon() should contain checkmark")
			}
			if !strings.Contains(got, "  ") && tt.wantIcon == "  " {
				t.Errorf("providerStatusIcon() should contain spaces")
			}
		})
	}
}

func TestExitOptionDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantExit bool
	}{
		{"empty string", "", true},
		{"done", "done", true},
		{"lowercase q", "q", true},
		{"exit", "exit", true},
		{"number", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed := strings.TrimSpace(tt.input)
			got := trimmed == "" || trimmed == "done" || trimmed == "q" || trimmed == "exit"
			if got != tt.wantExit {
				t.Errorf("exit detection for %q = %v, want %v", tt.input, got, tt.wantExit)
			}
		})
	}
}
