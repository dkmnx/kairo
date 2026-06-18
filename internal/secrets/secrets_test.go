package secrets

import (
	"os"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]string{},
		},
		{
			name:     "single key-value",
			input:    "KEY=value",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "multiple key-values",
			input:    "KEY1=value1\nKEY2=value2\nKEY3=value3",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2", "KEY3": "value3"},
		},
		{
			name:     "value with equals sign",
			input:    "KEY=a=b=c",
			expected: map[string]string{"KEY": "a=b=c"},
		},
		{
			name:     "empty lines ignored",
			input:    "\n\nKEY1=value1\n\nKEY2=value2\n\n",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "lines without equals ignored",
			input:    "KEY1=value1\nnoequals\nKEY2=value2",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "trailing newline",
			input:    "KEY=value\n",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "real world secrets format",
			input:    "ZAI_API_KEY=sk-test-key123\nMINIMAX_API_KEY=sk-another-key456\n",
			expected: map[string]string{"ZAI_API_KEY": "sk-test-key123", "MINIMAX_API_KEY": "sk-another-key456"},
		},
		{
			name:     "comment lines ignored",
			input:    "# this is a comment\nKEY=value\n# another comment",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "indented comment ignored",
			input:    "  # indented comment\nKEY=value",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "comment between entries",
			input:    "KEY1=val1\n# comment\nKEY2=val2",
			expected: map[string]string{"KEY1": "val1", "KEY2": "val2"},
		},
		{
			name:     "whitespace trimmed from lines, not key-value parts",
			input:    "  KEY1 =val1  \n  KEY2=val2",
			expected: map[string]string{"KEY1 ": "val1", "KEY2": "val2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Parse() returned %d entries, want %d", len(result), len(tt.expected))
				return
			}
			for key, value := range tt.expected {
				if result[key] != value {
					t.Errorf("Parse()[%q] = %q, want %q", key, result[key], value)
				}
			}
		})
	}
}

func TestParseEmptyKey(t *testing.T) {
	result := Parse("=value")
	if _, ok := result[""]; ok {
		t.Errorf("Parse() should skip entries with empty keys, got entry for empty key")
	}
	if len(result) != 0 {
		t.Errorf("Parse() length = %d, want 0 (empty keys should be skipped)", len(result))
	}
}

func TestParseEmptyValue(t *testing.T) {
	result := Parse("KEY=")
	if _, ok := result["KEY"]; ok {
		t.Errorf("Parse() should skip entries with empty values, got entry for KEY")
	}
	if len(result) != 0 {
		t.Errorf("Parse() length = %d, want 0 (empty values should be skipped)", len(result))
	}
}

func TestParseNewlines(t *testing.T) {
	result := Parse("KEY1=value1\nKEY2=value\nwith\nnewline\nKEY3=value3")

	if result["KEY1"] != "value1" {
		t.Errorf("Parse()[KEY1] = %q, want %q", result["KEY1"], "value1")
	}
	if result["KEY3"] != "value3" {
		t.Errorf("Parse()[KEY3] = %q, want %q", result["KEY3"], "value3")
	}

	if result["KEY2"] != "value" {
		t.Errorf("Parse()[KEY2] = %q, want %q", result["KEY2"], "value")
	}

	if _, exists := result["with"]; exists {
		t.Error("Parse() should skip line 'with' (no =)")
	}
	if _, exists := result["newline"]; exists {
		t.Error("Parse() should skip line 'newline' (no =)")
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		secrets  map[string]string
		expected string
	}{
		{
			name:     "empty map",
			secrets:  map[string]string{},
			expected: "",
		},
		{
			name:     "single entry",
			secrets:  map[string]string{"KEY": "value"},
			expected: "KEY=value\n",
		},
		{
			name:     "multiple entries sorted",
			secrets:  map[string]string{"Z_KEY": "val3", "A_KEY": "val1", "M_KEY": "val2"},
			expected: "A_KEY=val1\nM_KEY=val2\nZ_KEY=val3\n",
		},
		{
			name:     "skips empty key",
			secrets:  map[string]string{"": "value", "KEY": "value"},
			expected: "KEY=value\n",
		},
		{
			name:     "skips empty value",
			secrets:  map[string]string{"KEY": "", "OTHER": "value"},
			expected: "OTHER=value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Format(tt.secrets)
			if result != tt.expected {
				t.Errorf("Format() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatRoundTrip(t *testing.T) {
	original := map[string]string{
		"ZAI_API_KEY":   "sk-test-123",
		"ANTHROPIC_KEY": "sk-ant-456",
		"DEEPSEEK_KEY":  "sk-deep-789",
	}

	formatted := Format(original)
	parsed := Parse(formatted)

	if len(parsed) != len(original) {
		t.Errorf("Round trip: got %d entries, want %d", len(parsed), len(original))
	}

	for key, value := range original {
		if parsed[key] != value {
			t.Errorf("Round trip: parsed[%q] = %q, want %q", key, parsed[key], value)
		}
	}
}

func TestParseWithStatsWarnings(t *testing.T) {
	result := ParseWithStats("=secret_value\nKEY=\nVALID=invalid\nNO_EQUALS")

	if result.SkippedCount != 3 {
		t.Errorf("SkippedCount = %d, want 3", result.SkippedCount)
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for malformed entries")
	}

	warningText := strings.Join(result.Warnings, " ")
	if !strings.Contains(warningText, "empty key") {
		t.Error("Expected warning about empty key")
	}
	if !strings.Contains(warningText, "empty value") {
		t.Error("Expected warning about empty value")
	}
}

// TestParseWithStatsNoGlobalSideEffect verifies that ParseWithStats does not
// write to stderr via a global side-effect. Warnings are returned in Result.Warnings.
func TestParseWithStatsNoGlobalSideEffect(t *testing.T) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	defer func() { os.Stderr = oldStderr }()

	result := ParseWithStats("=empty_key\nKEY=\nNO_EQUALS\n")

	w.Close()
	stderrOutput := make([]byte, 4096)
	n, _ := r.Read(stderrOutput)
	stderrStr := string(stderrOutput[:n])
	r.Close()

	if len(result.Warnings) == 0 {
		t.Fatal("Expected warnings in Result.Warnings")
	}
	if stderrStr != "" {
		t.Errorf("Expected nothing on stderr, got: %q", stderrStr)
	}
}

// TestParseStillReturnsWarnings verifies warnings are present in Result.Warnings.
func TestParseStillReturnsWarnings(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		skippedCount  int
		wantWarnings  int
		warnSubstring string
	}{
		{
			name:          "empty key",
			input:         "=value",
			skippedCount:  1,
			wantWarnings:  1,
			warnSubstring: "empty key",
		},
		{
			name:          "empty value",
			input:         "KEY=",
			skippedCount:  1,
			wantWarnings:  1,
			warnSubstring: "empty value",
		},
		{
			name:          "no equals sign",
			input:         "NOEQUALS",
			skippedCount:  1,
			wantWarnings:  0, // lines without = are counted but have no specific warning
			warnSubstring: "",
		},
		{
			name:          "mixed malformed",
			input:         "OK=1\n=bad\nKEY2=",
			skippedCount:  2,
			wantWarnings:  2,
			warnSubstring: "empty key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWithStats(tt.input)
			if result.SkippedCount != tt.skippedCount {
				t.Errorf("SkippedCount = %d, want %d", result.SkippedCount, tt.skippedCount)
			}
			if len(result.Warnings) != tt.wantWarnings {
				t.Errorf("len(Warnings) = %d, want %d", len(result.Warnings), tt.wantWarnings)
			}
			if tt.wantWarnings > 0 && tt.warnSubstring != "" {
				warnText := strings.Join(result.Warnings, " ")
				if !strings.Contains(warnText, tt.warnSubstring) {
					t.Errorf("Warnings should contain %q, got: %s", tt.warnSubstring, warnText)
				}
			}
		})
	}
}
