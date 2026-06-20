package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGenOutput(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Restore unconditionally even if main() panics.
	defer func() { os.Stdout = old }()
	os.Stdout = w

	main()

	w.Close()

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	output := buf.String()
	if output == "" {
		t.Fatal("gen output is empty")
	}

	if !strings.Contains(output, "| Provider") {
		t.Error("output should contain provider table header '| Provider'")
	}

	if !strings.Contains(output, "anthropic") {
		t.Error("output should contain provider 'anthropic'")
	}

	if !strings.Contains(output, "|---") {
		t.Error("output should contain markdown table separator '|---'")
	}
}
