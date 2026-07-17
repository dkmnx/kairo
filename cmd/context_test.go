package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

// testCLI is a shared *CLIContext used by tests that need to set up a config
// directory. Tests should use this instead of relying on CLIContextFromCmd
// when no cobra.Command is available.
var testCLI = NewCLIContext()

func TestMustCLIContextFromCmd_PanicsWithoutContext(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(cmd.Context())

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic from MustCLIContextFromCmd")
		}
	}()

	MustCLIContextFromCmd(cmd)
}

func TestMustCLIContextFromCmd_ReturnsContext(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cliCtx := NewCLIContext()
	cmd.SetContext(WithCLIContext(context.Background(), cliCtx))

	got := MustCLIContextFromCmd(cmd)
	if got != cliCtx {
		t.Error("MustCLIContextFromCmd() returned wrong context")
	}
}
