package cmd

import (
	"testing"
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
			got := RequiresAPIKey(tt.provider)
			if got != tt.want {
				t.Errorf("RequiresAPIKey(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}
