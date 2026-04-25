package ui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
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

	if !bytes.Contains(buf.Bytes(), []byte("✓")) {
		t.Error("PrintSuccess should contain checkmark")
	}
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Error("PrintSuccess should contain message")
	}
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

	if !bytes.Contains(buf.Bytes(), []byte("⚠")) {
		t.Error("PrintWarn should contain warning symbol")
	}
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

func TestColorReset(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintWhite("message with reset")

	w.Close()
	_, _ = buf.ReadFrom(r)
	os.Stdout = originalStdout

	if !bytes.Contains(buf.Bytes(), []byte("message with reset")) {
		t.Error("Output should contain message after color codes")
	}
	if !bytes.Contains(buf.Bytes(), []byte(Reset)) {
		t.Error("Output should contain reset code")
	}
}

func TestPrintBanner(t *testing.T) {
	t.Run("prints banner with version model and provider", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintBanner("1.0.0-dev", "claude-sonnet-4-20250514", "Z.AI")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if !strings.Contains(output, "kairo") {
			t.Error("PrintBanner should contain kairo")
		}

		if !strings.Contains(output, "1.0.0-dev") {
			t.Error("PrintBanner should display version")
		}

		if !strings.Contains(output, "claude-sonnet-4-20250514") {
			t.Error("PrintBanner should display model")
		}

		if !strings.Contains(output, "Z.AI") {
			t.Error("PrintBanner should display provider name")
		}

		if !strings.Contains(output, Gray) {
			t.Error("PrintBanner should use gray formatting")
		}
	})

	t.Run("handles custom provider and model", func(t *testing.T) {
		buf := new(bytes.Buffer)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintBanner("2.0.0", "custom-model", "mycustomprovider")

		w.Close()
		_, _ = buf.ReadFrom(r)
		os.Stdout = originalStdout

		output := buf.String()

		if !strings.Contains(output, "2.0.0") {
			t.Error("PrintBanner should display custom version")
		}

		if !strings.Contains(output, "custom-model") {
			t.Error("PrintBanner should display custom model")
		}

		if !strings.Contains(output, "mycustomprovider") {
			t.Error("PrintBanner should display custom provider name")
		}
	})
}

func TestProviderRequirements(t *testing.T) {
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

		result, err := Confirm("Are you sure?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}

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

		result, err := Confirm("Proceed?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}
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

		result, err := Confirm("Continue?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}
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

		result, err := Confirm("Delete all?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}
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

		result, err := Confirm("Destroy data?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}
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

		result, err := Confirm("Confirm action?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}
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

		result, err := Confirm("Confirm?")
		if err != nil {
			t.Fatalf("Confirm() error = %v", err)
		}
		if result {
			t.Error("Confirm() should return false for empty input")
		}
	})
}

func TestClearScreen(t *testing.T) {
	t.Run("executes without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ClearScreen() panicked: %v", r)
			}
		}()

		ClearScreen()
	})

	t.Run("uses correct command for platform", func(t *testing.T) {
		ClearScreen()
	})
}

func TestErrUserCancelled(t *testing.T) {
	t.Run("error is defined and can be checked", func(t *testing.T) {
		if kairoerrors.ErrUserCancelled.Error() != "user cancelled input" {
			t.Errorf("ErrUserCancelled.Error() = %q, want %q", kairoerrors.ErrUserCancelled.Error(), "user cancelled input")
		}

		if !errors.Is(kairoerrors.ErrUserCancelled, kairoerrors.ErrUserCancelled) {
			t.Error("ErrUserCancelled should be equal to itself via errors.Is")
		}
	})
}

func TestIsInterrupted(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "os.ErrClosed",
			err:      os.ErrClosed,
			expected: true,
		},
		{
			name:     "io.EOF",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "error with 'interrupted' message",
			err:      errors.New("operation interrupted"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInterrupted(tt.err)
			if result != tt.expected {
				t.Errorf("isInterrupted(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsEmptyInput(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "io.EOF",
			err:      io.EOF,
			expected: false,
		},
		{
			name:     "os.ErrClosed (interrupted)",
			err:      os.ErrClosed,
			expected: false,
		},
		{
			name:     "error with 'interrupted' message",
			err:      errors.New("operation interrupted"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyInput(tt.err)
			if result != tt.expected {
				t.Errorf("isEmptyInput(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
