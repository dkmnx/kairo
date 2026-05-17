package wrapper

import (
	"strings"
	"testing"
)

func TestGenerateUnixScript(t *testing.T) {
	cfg := ScriptConfig{
		AuthDir:   "/tmp/auth",
		TokenPath: "/tmp/auth/token",
		CliPath:   "/usr/bin/claude",
		CliArgs:   []string{"--help", "--verbose"},
	}

	content := generateUnixScript("ANTHROPIC_AUTH_TOKEN", cfg)

	if !strings.Contains(content, "#!/bin/sh") {
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

	if !strings.Contains(content, "#!/bin/sh") {
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
