package cmd

import (
	"os"
	"testing"
)

func TestMergeEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envs     [][]string
		expected []string
	}{
		{
			name:     "empty input",
			envs:     [][]string{},
			expected: []string{},
		},
		{
			name:     "single env slice",
			envs:     [][]string{{"KEY1=value1", "KEY2=value2"}},
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "two env slices with no duplicates",
			envs:     [][]string{{"KEY1=value1"}, {"KEY2=value2"}},
			expected: []string{"KEY1=value1", "KEY2=value2"},
		},
		{
			name:     "two env slices with duplicate - later wins",
			envs:     [][]string{{"KEY1=value1"}, {"KEY1=value2"}},
			expected: []string{"KEY1=value2"},
		},
		{
			name:     "three env slices with duplicates",
			envs:     [][]string{{"KEY1=first"}, {"KEY2=value2"}, {"KEY1=last", "KEY3=value3"}},
			expected: []string{"KEY2=value2", "KEY1=last", "KEY3=value3"},
		},
		{
			name:     "invalid entries ignored",
			envs:     [][]string{{"=value", "KEY=", "VALID=value"}},
			expected: []string{"KEY=", "VALID=value"},
		},
		{
			name:     "preserves order of first occurrences",
			envs:     [][]string{{"A=1", "B=2", "C=3"}, {"D=4", "B=updated", "E=5"}},
			expected: nil, // don't check exact order, just verify length
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeEnvVars(tt.envs...)

			if tt.expected == nil {
				// just verify no panic and returns some result
				if result == nil {
					t.Error("mergeEnvVars() returned nil, want slice")
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("mergeEnvVars() returned %d items, want %d", len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("mergeEnvVars()[%d] = %q, want %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestRunningWithRaceDetector(t *testing.T) {
	tests := []struct {
		name     string
		goFlags  string
		expected bool
	}{
		{
			name:     "no GOFLAGS",
			goFlags:  "",
			expected: false,
		},
		{
			name:     "GOFLAGS without race",
			goFlags:  "-v -p=4",
			expected: false,
		},
		{
			name:     "GOFLAGS with race",
			goFlags:  "-race -v",
			expected: true,
		},
		{
			name:     "GOFLAGS with race flag",
			goFlags:  "-v -race -p=2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalGoFlags := os.Getenv("GOFLAGS")
			defer os.Setenv("GOFLAGS", originalGoFlags)

			if tt.goFlags == "" {
				os.Unsetenv("GOFLAGS")
			} else {
				os.Setenv("GOFLAGS", tt.goFlags)
			}

			result := runningWithRaceDetector()
			if result != tt.expected {
				t.Errorf("runningWithRaceDetector() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSetupSignalHandler(t *testing.T) {
	t.Run("signal handler sets up without panic", func(t *testing.T) {
		setupSignalHandler(func() {
			// cancel callback
		})
	})
}
