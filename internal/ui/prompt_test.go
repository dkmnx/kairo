package ui

import (
	"bytes"
	"os"
	"testing"
)

func TestPrintSuccess(t *testing.T) {
	buf := new(bytes.Buffer)
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintSuccess("test message")

	w.Close()
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
	buf.ReadFrom(r)
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
