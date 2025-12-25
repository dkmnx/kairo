package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.Printf("Kairo version: %s\n", version)
		},
	}

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("version output is empty")
	}
	if !strings.Contains(output, "Kairo version:") {
		t.Errorf("output doesn't contain version: %q", output)
	}
}
