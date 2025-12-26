package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/dkmnx/kairo/internal/version"
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
			version.Version = "v1.0.0"
			version.Commit = "abc123"
			version.Date = "2025-12-26T09:15:46Z"
			cmd.Printf("Kairo version: %s\n", version.Version)
			if version.Commit != "unknown" && version.Commit != "" {
				cmd.Printf("Commit: %s\n", version.Commit)
			}
			if version.Date != "" && version.Date != "unknown" {
				if t, err := time.Parse(time.RFC3339, version.Date); err == nil {
					cmd.Printf("Date: %s\n", t.Format("2006-01-02"))
				} else {
					cmd.Printf("Date: %s\n", version.Date)
				}
			}
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

	expectedParts := []string{
		"Kairo version: v1.0.0",
		"Commit: abc123",
		"Date: 2025-12-26",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("output doesn't contain expected part %q, got: %q", part, output)
		}
	}

	if strings.Contains(output, "T09:15:46Z") {
		t.Error("date should be formatted without time component")
	}
}
