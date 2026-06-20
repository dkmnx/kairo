package wrapper

import (
	"strings"
	"testing"
)

func TestShellQuotePOSIX(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", "''"},
		{"simple", "hello", "'hello'"},
		{"with spaces", "hello world", "'hello world'"},
		{"dollar", "$HOME", "'$HOME'"},
		{"backtick", "`ls`", "'`ls`'"},
		{"single quote inside", "it's", "'it'\\''s'"},
		{"multiple single quotes", "'a' 'b'", "''\\''a'\\'' '\\''b'\\'''"},
		{"trailing single quote", "end'", "'end'\\'''"},
		{"leading single quote", "'start", "''\\''start'"},
		{"form feed", "\x0c", "'\x0c'"},
		{"non-ascii", "héllo 🚀", "'héllo 🚀'"},
		{"flag with equals", "--flag=\"a b\"", "'--flag=\"a b\"'"},
		{"backslash", "C:\\path", "'C:\\path'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuotePOSIX(tt.input)
			if got != tt.want {
				t.Errorf("shellQuotePOSIX(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateUnixScript(t *testing.T) {
	cfg := ScriptConfig{
		AuthDir:   "/tmp/auth",
		TokenPath: "/tmp/auth/token",
		CliPath:   "/usr/bin/claude",
		CliArgs:   []string{"--help", "--verbose"},
	}

	content := generateUnixScript("ANTHROPIC_AUTH_TOKEN", cfg)

	if !strings.HasPrefix(content, "#!/bin/sh") {
		t.Error("Unix script should start with shebang")
	}
	if !strings.Contains(content, "export ANTHROPIC_AUTH_TOKEN=") {
		t.Error("Unix script should export auth token")
	}
	if !strings.Contains(content, "rm -f") {
		t.Error("Unix script should remove token file")
	}
	if !strings.Contains(content, "exec") {
		t.Error("Unix script should use exec")
	}
	if !strings.Contains(content, "--help") || !strings.Contains(content, "--verbose") {
		t.Error("Unix script should include CLI args")
	}
}

func TestGenerateUnixScript_CustomEnvVar(t *testing.T) {
	cfg := ScriptConfig{
		AuthDir:   "/tmp/auth",
		TokenPath: "/tmp/auth/token",
		CliPath:   "/usr/bin/claude",
		CliArgs:   []string{},
	}

	content := generateUnixScript("MY_API_KEY", cfg)

	if !strings.Contains(content, "export MY_API_KEY=") {
		t.Error("Unix script should use custom env var")
	}
}

func TestGenerateUnixScript_POSIXQuotingRoundTrip(t *testing.T) {
	cfg := ScriptConfig{
		AuthDir:   "/tmp/auth",
		TokenPath: "/tmp/auth/token with spaces",
		CliPath:   "/usr/local/bin/claude",
		CliArgs:   []string{"--prompt", "Hello $HOME `whoami` and it's fine", "--flag='a b'"},
	}

	content := generateUnixScript("ANTHROPIC_AUTH_TOKEN", cfg)

	// Token path is single-quoted, not expanded
	if !strings.Contains(content, "'/tmp/auth/token with spaces'") {
		t.Error("token path should be POSIX single-quoted")
	}

	// CliPath is single-quoted
	if !strings.Contains(content, "'/usr/local/bin/claude'") {
		t.Error("cli path should be POSIX single-quoted")
	}

	// Dollar signs are literal inside single quotes, not expanded
	if !strings.Contains(content, "'Hello $HOME `whoami` and it'\\''s fine'") {
		t.Error("args with $ and backticks should be POSIX single-quoted, not expanded")
	}

	// All single quotes inside args are escaped
	if !strings.Contains(content, "'--flag='\\''a b'\\'''") {
		t.Error("single quotes inside args should be escaped for POSIX")
	}
}

func TestGenerateScriptContent_Windows(t *testing.T) {
	cfg := ScriptConfig{
		AuthDir:   "/tmp/auth",
		TokenPath: "/tmp/auth/token",
		CliPath:   "C:\\claude.exe",
		CliArgs:   []string{"--help"},
	}

	content := generateScriptContent(true, "ANTHROPIC_AUTH_TOKEN", cfg)

	if !strings.Contains(content, "\r\n") {
		t.Error("Windows script should use CRLF line endings")
	}
	if !strings.Contains(content, "$env:ANTHROPIC_AUTH_TOKEN") {
		t.Error("Windows script should set env var")
	}
	if !strings.Contains(content, "Get-Content") {
		t.Error("Windows script should read token file")
	}
	if !strings.Contains(content, "Remove-Item") {
		t.Error("Windows script should clean up token file")
	}
}

func TestGenerateScriptContent_Unix(t *testing.T) {
	cfg := ScriptConfig{
		AuthDir:   "/tmp/auth",
		TokenPath: "/tmp/auth/token",
		CliPath:   "/usr/bin/claude",
		CliArgs:   []string{"--help", "--verbose"},
	}

	content := generateScriptContent(false, "ANTHROPIC_AUTH_TOKEN", cfg)

	if !strings.HasPrefix(content, "#!/bin/sh") {
		t.Error("Unix script should have shebang")
	}
	if !strings.Contains(content, "export ANTHROPIC_AUTH_TOKEN") {
		t.Error("Unix script should export env var")
	}
	if !strings.Contains(content, "rm -f") {
		t.Error("Unix script should remove token file")
	}
	if !strings.Contains(content, "exec") {
		t.Error("Unix script should use exec")
	}
	if !strings.Contains(content, "--help") || !strings.Contains(content, "--verbose") {
		t.Error("Unix script should include all args")
	}
}
