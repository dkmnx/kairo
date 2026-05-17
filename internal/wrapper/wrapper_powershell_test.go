package wrapper

import (
	"strings"
	"testing"
)

func TestEscapePowerShellArg_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "''"},
		{"simple", "hello", "'hello'"},
		{"with spaces", "hello world", "'hello world'"},
		{"single quote", "can't", "'can''t'"},
		{"double quote", `say "hi"`, `'say \"hi\"'`},
		{"dollar sign", "$HOME", "'`$HOME'"},
		{"backtick", "foo`bar", "'foo``bar'"},
		{"complex", `$env:PATH = "C:\test"`, "'`$env:PATH = \\\"C:\\test\\\"'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapePowerShellArg_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"newline", "line1\nline2", "'line1`nline2'"},
		{"carriage return", "line1\rline2", "'line1`rline2'"},
		{"crlf", "line1\r\nline2", "'line1`r`nline2'"},
		{"tab", "col1\tcol2", "'col1`tcol2'"},
		{"backspace", "back\bspace", "'back`bspace'"},
		{"null", "before\x00after", "'before`0after'"},
		{"unicode emoji", "🚀🎉", "'🚀🎉'"},
		{"unicode chinese", "你好世界", "'你好世界'"},
		{"windows path long", `C:\Users\JohnDoe\AppData\Local\Programs\Claude\claude.exe`, "'C:\\Users\\JohnDoe\\AppData\\Local\\Programs\\Claude\\claude.exe'"},
		{"windows path with spaces", `C:\Program Files\My App\file.txt`, "'C:\\Program Files\\My App\\file.txt'"},
		{"semicolon", "test; cmd", "'test`; cmd'"},
		{"pipe", "test | calc", "'test `| calc'"},
		{"variable style", "$myVar", "'`$myVar'"},
		{"env variable", "$env:PATH", "'`$env:PATH'"},
		{"at sign", "@()", "'@()'"},
		{"percent", "100%", "'100``%'"},
		{"multi-line", "line1\nline2", "'line1`nline2'"},
		{"json string", `{"key":"value"}`, "'{\\\"key\\\":\\\"value\\\"}'"},
		{"base64", "SGVsbG8gV29ybGQ=", "'SGVsbG8gV29ybGQ='"},
		{"prompt with quotes", `Say "hello" to me`, "'Say \\\"hello\\\" to me'"},
		{"curl command", `curl -H "Authorization: Bearer $TOKEN" https://api.example.com`, "'curl -H \\\"Authorization: Bearer `$TOKEN\\\" https://api.example.com'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapePowerShellArg_CommandInjectionPrevention(t *testing.T) {
	injectionPatterns := []struct {
		name  string
		input string
	}{
		{"cmd execl", "$(cmd.exe /c whoami)"},
		{"powershell execl", "$(powershell.exe -Command 'Get-Process')"},
		{"backtick exec", "`whoami"},
		{"command substitution", "$((whoami))"},
		{"env substitution", "$env:COMPUTERNAME"},
	}

	for _, tt := range injectionPatterns {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			// Result should be wrapped in single quotes
			if !strings.HasPrefix(result, "'") || !strings.HasSuffix(result, "'") {
				t.Errorf("Result should be wrapped in single quotes, got: %q", result)
			}
			// Dollar signs should be escaped as `$
			if strings.Contains(result, "$(") && !strings.Contains(result, "`$(") {
				t.Errorf("Command substitution not properly escaped in: %q", result)
			}
			// Backticks should be doubled or part of valid escape
			if strings.Contains(result, "`") && !strings.Contains(result, "``") &&
				!strings.Contains(result, "`$") && !strings.Contains(result, "`n") &&
				!strings.Contains(result, "`r") && !strings.Contains(result, "`t") {
				t.Errorf("Backtick not properly escaped in: %q", result)
			}
		})
	}
}

func TestEscapePowerShellArg_SemicolonAndPipe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"semicolon", "test; cmd", "'test`; cmd'"},
		{"pipe", "test | calc", "'test `| calc'"},
		{"both", "cmd1; cmd2 | grep", "'cmd1`; cmd2 `| grep'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePowerShellArg(tt.input)
			if result != tt.expected {
				t.Errorf("EscapePowerShellArg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
