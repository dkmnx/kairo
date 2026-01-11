package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/config"
	"github.com/dkmnx/kairo/internal/providers"
)

func TestPrintSuccess(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintSuccess("test message")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("✓")) {
		t.Error("PrintSuccess should contain checkmark")
	}
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Error("PrintSuccess should contain message")
	}
	_ = output
}

func TestPrintWarn(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintWarn("test warning")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("⚠")) {
		t.Error("PrintWarn should contain warning symbol")
	}
	_ = output
}

func TestPrintError(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintError("test error")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stderr = originalStderr

	if !bytes.Contains(buf.Bytes(), []byte("✗")) {
		t.Error("PrintError should contain X symbol")
	}
}

func TestPrintInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintInfo("info message")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	if !bytes.Contains(buf.Bytes(), []byte("info message")) {
		t.Error("PrintInfo should contain message")
	}
}

func TestPrintHeader(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintHeader("header text")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	if !bytes.Contains(buf.Bytes(), []byte("header text")) {
		t.Error("PrintHeader should contain message")
	}
}

func TestPrintSection(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintSection("section name")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	if !bytes.Contains(buf.Bytes(), []byte("section name")) {
		t.Error("PrintSection should contain section name")
	}
	if !bytes.Contains(buf.Bytes(), []byte("===")) {
		t.Error("PrintSection should contain section delimiters")
	}
}

func TestPrintWhite(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintWhite("white text")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	if !bytes.Contains(buf.Bytes(), []byte("white text")) {
		t.Error("PrintWhite should contain message")
	}
}

func TestPrintGray(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintGray("gray text")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	if !bytes.Contains(buf.Bytes(), []byte("gray text")) {
		t.Error("PrintGray should contain message")
	}
}

func TestPrintDefault(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintDefault("provider name")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("provider name")) {
		t.Error("PrintDefault should contain provider name")
	}
	if !bytes.Contains(buf.Bytes(), []byte("(default)")) {
		t.Error("PrintDefault should contain '(default)'")
	}
	_ = output
}

func TestColorReset(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintWhite("message with reset")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("message with reset")) {
		t.Error("Output should contain message after color codes")
	}
	if !bytes.Contains(buf.Bytes(), []byte(Reset)) {
		t.Error("Output should contain reset code")
	}
	_ = output
}

func TestPromptSecret(t *testing.T) {
	t.Run("reads password successfully", func(t *testing.T) {
		// Note: PromptSecret uses term.ReadPassword which requires a TTY.
		// In CI/testing environments without a TTY, this will fail.
		// We test that the function signature and basic behavior work.
		// This test is skipped in non-TTY environments automatically.

		// Create a pipe for stdin
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		// Write test input
		go func() {
			_, _ = pw.WriteString("test-password-123\n")
			pw.Close()
		}()

		// Redirect stdin
		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		// Capture stdout
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result, err := PromptSecret("Enter password")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// term.ReadPassword may fail in non-TTY environments
		// We accept both success and expected failure
		if err == nil {
			if result != "test-password-123" {
				t.Errorf("PromptSecret() = %q, want %q", result, "test-password-123")
			}

			output := buf.String()
			if !strings.Contains(output, "Enter password") {
				t.Error("PromptSecret should display prompt")
			}
		} else {
			// Expected error in non-TTY environment - this is acceptable
			t.Skipf("PromptSecret requires TTY: %v", err)
		}
	})

	t.Run("returns empty string for empty input", func(t *testing.T) {
		// Similar to above - may fail in non-TTY environments
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("\n")
			pw.Close()
		}()

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result, err := PromptSecret("Enter key")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if err == nil && result != "" {
			t.Errorf("PromptSecret() = %q, want empty string", result)
		} else if err != nil {
			t.Skipf("PromptSecret requires TTY: %v", err)
		}
	})
}

func TestPrompt(t *testing.T) {
	t.Run("reads input successfully", func(t *testing.T) {
		// Create pipe for stdin
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		// Write input before reading
		go func() {
			_, _ = pw.WriteString("my-input-value\n")
			pw.Close()
		}()

		// Small delay to ensure input is available
		time.Sleep(10 * time.Millisecond)

		// Redirect stdin
		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		// Capture stdout
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := Prompt("Enter value")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if result != "my-input-value" {
			t.Errorf("Prompt() = %q, want %q", result, "my-input-value")
		}

		output := buf.String()
		if !strings.Contains(output, "Enter value") {
			t.Error("Prompt should display prompt")
		}
		if !strings.Contains(output, ": ") {
			t.Error("Prompt should display colon and space")
		}
	})

	t.Run("returns empty string for empty input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := Prompt("Enter name")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if result != "" {
			t.Errorf("Prompt() = %q, want empty string", result)
		}
	})

	t.Run("handles whitespace-only input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("   \n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := Prompt("Enter text")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		// fmt.Scanln trims whitespace, so whitespace-only input returns empty string
		if result != "" {
			t.Errorf("Prompt() = %q, want empty string (fmt.Scanln trims whitespace)", result)
		}
	})
}

func TestPromptWithDefault(t *testing.T) {
	t.Run("uses user input when provided", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("custom-value\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := PromptWithDefault("Enter API key", "default-key-123")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if result != "custom-value" {
			t.Errorf("PromptWithDefault() = %q, want %q", result, "custom-value")
		}

		output := buf.String()
		if !strings.Contains(output, "Enter API key") {
			t.Error("PromptWithDefault should display prompt")
		}
		if !strings.Contains(output, "[default-key-123]") {
			t.Error("PromptWithDefault should display default value in brackets")
		}
	})

	t.Run("uses default when input is empty", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := PromptWithDefault("Enter URL", "https://api.example.com")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if result != "https://api.example.com" {
			t.Errorf("PromptWithDefault() = %q, want default value", result)
		}
	})

	t.Run("handles empty default value", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("user-provided\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := PromptWithDefault("Enter value", "")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if result != "user-provided" {
			t.Errorf("PromptWithDefault() = %q, want user-provided", result)
		}

		output := buf.String()
		if strings.Contains(output, "[]") {
			t.Error("PromptWithDefault should not display empty brackets when default is empty")
		}
	})
}

func TestPrintBanner(t *testing.T) {
	t.Run("prints banner with version and provider", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintBanner("1.0.0-dev", "Z.AI")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		// Check for ASCII art characters
		if !strings.Contains(output, "████") {
			t.Error("PrintBanner should contain ASCII art characters")
		}

		// Check for version
		if !strings.Contains(output, "1.0.0-dev") {
			t.Error("PrintBanner should display version")
		}

		// Check for provider
		if !strings.Contains(output, "Z.AI") {
			t.Error("PrintBanner should display provider name")
		}

		// Check for bold formatting
		if !strings.Contains(output, Bold) {
			t.Error("PrintBanner should use bold formatting")
		}
	})

	t.Run("handles custom provider name", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintBanner("2.0.0", "mycustomprovider")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if !strings.Contains(output, "2.0.0") {
			t.Error("PrintBanner should display custom version")
		}

		if !strings.Contains(output, "mycustomprovider") {
			t.Error("PrintBanner should display custom provider name")
		}
	})
}

func TestPrintProviderOption(t *testing.T) {
	t.Run("prints configured provider with checkmark", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		secrets := map[string]string{}

		PrintProviderOption(1, "Native Anthropic", cfg, secrets, "anthropic")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if !strings.Contains(output, "✓") {
			t.Error("PrintProviderOption should show checkmark for configured provider")
		}
		if !strings.Contains(output, "Native Anthropic") {
			t.Error("PrintProviderOption should display provider name")
		}
		if !strings.Contains(output, "1.") {
			t.Error("PrintProviderOption should display option number")
		}
	})

	t.Run("prints unconfigured provider without checkmark", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{}

		PrintProviderOption(2, "Z.AI", cfg, secrets, "zai")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if strings.Contains(output, "✓") {
			t.Error("PrintProviderOption should NOT show checkmark for unconfigured provider")
		}
		if !strings.Contains(output, "Z.AI") {
			t.Error("PrintProviderOption should display provider name")
		}
		if !strings.Contains(output, "2.") {
			t.Error("PrintProviderOption should display option number")
		}
	})

	t.Run("prints provider with API key in secrets", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{
			"ZAI_API_KEY": "sk-test-key-123",
		}

		PrintProviderOption(3, "Z.AI", cfg, secrets, "zai")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if !strings.Contains(output, "✓") {
			t.Error("PrintProviderOption should show checkmark when API key exists in secrets")
		}
	})

	t.Run("case-insensitive secret key matching", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{
			"zai_api_key": "sk-test-key", // lowercase key
		}

		PrintProviderOption(4, "Z.AI", cfg, secrets, "zai")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if !strings.Contains(output, "✓") {
			t.Error("PrintProviderOption should handle lowercase secret keys")
		}
	})
}

func TestIsProviderConfigured(t *testing.T) {
	t.Run("anthropic configured without API key", func(t *testing.T) {
		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		secrets := map[string]string{}

		result := isProviderConfigured(cfg, secrets, "anthropic")
		if !result {
			t.Error("isProviderConfigured should return true for configured anthropic")
		}
	})

	t.Run("anthropic not configured", func(t *testing.T) {
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{}

		result := isProviderConfigured(cfg, secrets, "anthropic")
		if result {
			t.Error("isProviderConfigured should return false for unconfigured anthropic")
		}
	})

	t.Run("provider with API key in secrets (uppercase)", func(t *testing.T) {
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{
			"ZAI_API_KEY": "sk-test-key",
		}

		result := isProviderConfigured(cfg, secrets, "zai")
		if !result {
			t.Error("isProviderConfigured should return true when uppercase API key exists")
		}
	})

	t.Run("provider with API key in secrets (lowercase)", func(t *testing.T) {
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{
			"zai_api_key": "sk-test-key",
		}

		result := isProviderConfigured(cfg, secrets, "zai")
		if !result {
			t.Error("isProviderConfigured should handle lowercase secret keys")
		}
	})

	t.Run("provider without API key", func(t *testing.T) {
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{}

		result := isProviderConfigured(cfg, secrets, "zai")
		if result {
			t.Error("isProviderConfigured should return false when API key is missing")
		}
	})

	t.Run("provider with other secrets but no API key", func(t *testing.T) {
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{
			"OTHER_SECRET": "some-value",
		}

		result := isProviderConfigured(cfg, secrets, "zai")
		if result {
			t.Error("isProviderConfigured should return false when provider-specific API key is missing")
		}
	})

	t.Run("custom provider with API key", func(t *testing.T) {
		cfg := &config.Config{
			Providers: make(map[string]config.Provider),
		}
		secrets := map[string]string{
			"MYPROVIDER_API_KEY": "sk-custom-key",
		}

		result := isProviderConfigured(cfg, secrets, "myprovider")
		if !result {
			t.Error("isProviderConfigured should return true for custom provider with API key")
		}
	})

	t.Run("multiple providers - check specific one", func(t *testing.T) {
		cfg := &config.Config{
			Providers: map[string]config.Provider{
				"anthropic": {Name: "Native Anthropic"},
			},
		}
		secrets := map[string]string{
			"ZAI_API_KEY":     "sk-zai-key",
			"MINIMAX_API_KEY": "sk-minimax-key",
		}

		// Check zai - should be true
		result := isProviderConfigured(cfg, secrets, "zai")
		if !result {
			t.Error("isProviderConfigured should return true for zai")
		}

		// Check minimax - should be true
		result = isProviderConfigured(cfg, secrets, "minimax")
		if !result {
			t.Error("isProviderConfigured should return true for minimax")
		}

		// Check deepseek - should be false
		result = isProviderConfigured(cfg, secrets, "deepseek")
		if result {
			t.Error("isProviderConfigured should return false for deepseek (no API key)")
		}
	})
}

func TestProviderRequirements(t *testing.T) {
	t.Run("anthropic does not require API key", func(t *testing.T) {
		requiresKey := providers.RequiresAPIKey("anthropic")
		if requiresKey {
			t.Error("anthropic should not require API key")
		}
	})

	t.Run("zai requires API key", func(t *testing.T) {
		requiresKey := providers.RequiresAPIKey("zai")
		if !requiresKey {
			t.Error("zai should require API key")
		}
	})

	t.Run("custom requires API key", func(t *testing.T) {
		requiresKey := providers.RequiresAPIKey("custom")
		if !requiresKey {
			t.Error("custom should require API key")
		}
	})
}

func TestConfirm(t *testing.T) {
	t.Run("returns true for 'yes' input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("yes\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		result := Confirm("Are you sure?")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		if !result {
			t.Error("Confirm() should return true for 'yes' input")
		}

		output := buf.String()
		if !strings.Contains(output, "Are you sure?") {
			t.Error("Confirm should display prompt message")
		}
	})

	t.Run("returns true for 'y' input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("y\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		result := Confirm("Proceed?")
		if !result {
			t.Error("Confirm() should return true for 'y' input")
		}
	})

	t.Run("returns true for 'YES' (uppercase)", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("YES\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		result := Confirm("Continue?")
		if !result {
			t.Error("Confirm() should return true for 'YES' (case-insensitive)")
		}
	})

	t.Run("returns false for 'no' input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("no\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		result := Confirm("Delete all?")
		if result {
			t.Error("Confirm() should return false for 'no' input")
		}
	})

	t.Run("returns false for 'n' input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("n\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		result := Confirm("Destroy data?")
		if result {
			t.Error("Confirm() should return false for 'n' input")
		}
	})

	t.Run("returns false for arbitrary input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("maybe\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		result := Confirm("Confirm action?")
		if result {
			t.Error("Confirm() should return false for non-yes/no input")
		}
	})

	t.Run("returns false for empty input", func(t *testing.T) {
		pr, pw, _ := os.Pipe()
		defer pr.Close()
		defer pw.Close()

		go func() {
			_, _ = pw.WriteString("\n")
			pw.Close()
		}()

		time.Sleep(10 * time.Millisecond)

		originalStdin := os.Stdin
		os.Stdin = pr
		defer func() { os.Stdin = originalStdin }()

		result := Confirm("Confirm?")
		if result {
			t.Error("Confirm() should return false for empty input")
		}
	})
}
