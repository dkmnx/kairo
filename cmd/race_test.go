package cmd

import (
	"runtime"
	"sync"
	"testing"
)

func TestGlobalVariableRaceCondition(t *testing.T) {
	t.Run("concurrent configDir access - should NOT detect race with -race flag", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping race test in short mode")
		}

		var wg sync.WaitGroup
		originalConfigDir := getConfigDir()
		defer func() { setConfigDir(originalConfigDir) }()

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < 100; j++ {
					setConfigDir("/tmp/test" + string(rune(id%3)))
					_ = getConfigDir()
				}
			}(i)
		}

		wg.Wait()
		t.Log("configDir access - should have no race with mutex protection")
	})

	t.Run("concurrent verbose access - should NOT detect race with -race flag", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping race test in short mode")
		}

		var wg sync.WaitGroup
		originalVerbose := getVerbose()
		defer func() { setVerbose(originalVerbose) }()

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < 100; j++ {
					setVerbose((j % 2) == 0)
					_ = getVerbose()
					runtime.Gosched()
				}
			}(i)
		}

		wg.Wait()
		t.Log("verbose access - should have no race with setVerbose()/getVerbose() mutex protection")
	})
}
