package main

import (
	"testing"

	"github.com/dkmnx/kairo/cmd"
)

func TestMainBootstrap(t *testing.T) {
	t.Run("version subcommand returns without error", func(t *testing.T) {
		cmd.SetTestArgs("version")
		err := cmd.Execute()
		if err != nil {
			t.Errorf("version should not error, got: %v", err)
		}
	})

	t.Run("list subcommand returns without error", func(t *testing.T) {
		cmd.SetTestArgs("list")
		err := cmd.Execute()
		if err != nil {
			t.Errorf("list should not error, got: %v", err)
		}
	})

	t.Run("completion subcommand returns error for invalid shell", func(t *testing.T) {
		cmd.SetTestArgs("completion", "unknown-shell")
		err := cmd.Execute()
		if err == nil {
			t.Error("completion with invalid shell should error")
		}
	})
}
