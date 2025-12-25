package cmd

import (
	"bytes"
	"testing"
)

func TestRootCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("output is empty")
	}
}

func TestRootFlagsExist(t *testing.T) {
	if rootCmd.Flags().Lookup("config") == nil {
		t.Error("--config flag not found")
	}
	if rootCmd.Flags().Lookup("verbose") == nil {
		t.Error("--verbose flag not found")
	}
}
