package providers

import (
	"strings"
	"testing"
)

func TestProviderTableMarkdown(t *testing.T) {
	result := ProviderTableMarkdown()

	if result == "" {
		t.Fatal("ProviderTableMarkdown() returned empty string")
	}

	if !strings.HasPrefix(result, "| Provider |") {
		t.Error("ProviderTableMarkdown() should start with header row")
	}

	if !strings.Contains(result, "|----------|") {
		t.Error("ProviderTableMarkdown() should contain separator row")
	}

	if !strings.Contains(result, "anthropic") {
		t.Error("ProviderTableMarkdown() should contain anthropic provider")
	}
}

func TestEscapeMarkdownPipe(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"no pipes", "no pipes"},
		{"has|pipe", "has\\|pipe"},
		{"a|b|c", "a\\|b\\|c"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapeMarkdownPipe(tt.input)
		if result != tt.expected {
			t.Errorf("escapeMarkdownPipe(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
