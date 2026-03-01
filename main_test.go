package main

import (
	"os"
	"testing"
)

func TestMainBootstrap(t *testing.T) {
	t.Run("main does not panic on help", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "--help"}

		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
					t.Errorf("main() panicked: %v", r)
				}
			}()
			main()
		}()

		if panicked {
			t.Error("main() should not panic")
		}
	})

	t.Run("main does not panic on version flag", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "version"}

		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
					t.Errorf("main() panicked with version command: %v", r)
				}
			}()
			main()
		}()

		if panicked {
			t.Error("main() should not panic on version command")
		}
	})

	t.Run("main does not panic on list command", func(t *testing.T) {
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"kairo", "list"}

		panicked := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			main()
		}()

		if panicked {
			t.Error("main() should not panic on list command")
		}
	})
}
