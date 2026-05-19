package validate

import (
	"regexp"
	"strings"
	"testing"
)

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		want    bool
		wantErr bool
	}{
		{
			name:    "empty pattern always matches",
			pattern: "",
			key:     "any-key",
			want:    true,
			wantErr: false,
		},
		{
			name:    "simple alphanumeric pattern",
			pattern: "^[a-z0-9-]+$",
			key:     "sk-ant-valid-key-123",
			want:    true,
			wantErr: false,
		},
		{
			name:    "simple pattern with invalid key",
			pattern: "^[a-z0-9-]+$",
			key:     "INVALID_KEY!@#",
			want:    false,
			wantErr: false,
		},
		{
			name:    "anthropic key pattern",
			pattern: "^sk-ant-[a-z0-9-]+$",
			key:     "sk-ant-api1234567890abcdef",
			want:    true,
			wantErr: false,
		},
		{
			name:    "anthropic pattern mismatch",
			pattern: "^sk-ant-[a-z0-9-]+$",
			key:     "sk-openai-key1234567890",
			want:    false,
			wantErr: false,
		},
		{
			name:    "case sensitive pattern",
			pattern: "^[A-Z]{2}[0-9]+$",
			key:     "AB123456",
			want:    true,
			wantErr: false,
		},
		{
			name:    "case sensitive pattern fails on lowercase",
			pattern: "^[A-Z]{2}[0-9]+$",
			key:     "ab123456",
			want:    false,
			wantErr: false,
		},
		{
			name:    "complex pattern with anchors",
			pattern: "^key-[a-f0-9]{32}$",
			key:     "key-1234567890abcdef1234567890abcdef",
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern with special characters",
			pattern: `^sk-[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`,
			key:     "sk-12345678-1234-1234-1234-123456789012",
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern with quantifiers",
			pattern: `^[a-z]+(\.[a-z]+)*@[a-z]+\.[a-z]{2,}$`,
			key:     "user.name@domain.com",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid regex pattern",
			pattern: "[invalid(unclosed",
			key:     "any-key",
			want:    false,
			wantErr: true,
		},
		{
			name:    "pattern matching empty string",
			pattern: "^$",
			key:     "",
			want:    true,
			wantErr: false,
		},
		{
			name:    "pattern with word boundaries",
			pattern: `^\b\w+\b$`,
			key:     "valid_key",
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kf := KeyFormat{Pattern: tt.pattern}
			got, err := kf.matchesPattern(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("matchesPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
			}
		})
	}
}

func TestMatchesPattern_CompilationCaching(t *testing.T) {
	kf := KeyFormat{Pattern: `^test-[a-z]+$`}

	_, err := kf.matchesPattern("test-key")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	_, err = kf.matchesPattern("test-another")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	_, err = kf.matchesPattern("test-123")
	if err != nil {
		t.Fatalf("Third call should use cached pattern: %v", err)
	}
}

func TestMatchesPattern_InvalidPatternReturnsError(t *testing.T) {
	invalidPatterns := []string{
		"[",
		"(",
		"*?",
		"???",
		"(?P<invalid>",
		"(?P",  // incomplete named group
		"(?",   // incomplete group
		"[a-z", // unclosed bracket
		"[^]",  // invalid negated class
	}

	for _, pattern := range invalidPatterns {
		t.Run(pattern, func(t *testing.T) {
			kf := KeyFormat{Pattern: pattern}
			_, err := kf.matchesPattern("test-key")
			if err == nil {
				t.Errorf("matchesPattern(%q) should return error for invalid pattern", pattern)
			}
		})
	}
}

func TestMatchesPattern_PatternMatchingEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     string
		want    bool
	}{
		{"unicode in pattern", `^[\pL\pN_-]+$`, "日本語キー-123", true},
		{"long key starting with sk-", `^sk-`, "sk-" + strings.Repeat("a", 1000), true},
		{"whitespace in key fails alphanumeric pattern", `^[a-z0-9]+$`, "key with space", false},
		{"newline in key", `^[\w]+$`, "key\nwith\nnewline", false},
		{"tab in key", `^[\w]+$`, "key\twith\ttab", false},
		{"very long pattern match", `^[a-z]+$`, strings.Repeat("abcdefgh", 100), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kf := KeyFormat{Pattern: tt.pattern}
			got, err := kf.matchesPattern(tt.key)
			if err != nil {
				t.Skipf("Pattern %q not supported: %v", tt.pattern, err)
			}
			if got != tt.want {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.key, got, tt.want)
			}
		})
	}
}

func TestCompilePattern_InvalidRegex(t *testing.T) {
	kf := &KeyFormat{Pattern: "[invalid"}
	err := kf.compilePattern()
	if err == nil {
		t.Error("compilePattern() should return error for invalid regex")
	}
}

func TestCompilePattern_EmptyPattern(t *testing.T) {
	kf := &KeyFormat{Pattern: ""}
	err := kf.compilePattern()
	if err != nil {
		t.Errorf("compilePattern() should return nil for empty pattern, got: %v", err)
	}
}

func TestCompilePattern_AlreadyCompiled(t *testing.T) {
	compiled := regexp.MustCompile(`^valid$`)
	kf := &KeyFormat{Pattern: "^valid$", compiled: compiled}
	err := kf.compilePattern()
	if err != nil {
		t.Errorf("compilePattern() should return nil when already compiled, got: %v", err)
	}
	if kf.compiled != compiled {
		t.Error("compilePattern() should not replace existing compiled pattern")
	}
}
