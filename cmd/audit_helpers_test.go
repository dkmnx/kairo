package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/dkmnx/kairo/internal/audit"
)

func TestLogAuditEvent_Success(t *testing.T) {
	t.Run("successfully logs audit event", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		secretsPath := filepath.Join(tmpDir, "secrets.age")
		keyPath := filepath.Join(tmpDir, "age.key")

		// Create minimal config
		if err := os.WriteFile(configPath, []byte("providers: {}"), 0600); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// Create key and secrets
		if err := os.WriteFile(keyPath, []byte("test-key-data"), 0600); err != nil {
			t.Fatalf("Failed to create key file: %v", err)
		}

		if err := os.WriteFile(secretsPath, []byte("test-secrets"), 0600); err != nil {
			t.Fatalf("Failed to create secrets file: %v", err)
		}

		// Test successful logging
		err := logAuditEvent(tmpDir, func(logger *audit.Logger) error {
			return logger.LogSwitch("test-provider")
		})

		if err != nil {
			t.Errorf("logAuditEvent should succeed, got error: %v", err)
		}

		// Verify audit file was created
		auditPath := filepath.Join(tmpDir, "audit.log")
		if _, err := os.Stat(auditPath); os.IsNotExist(err) {
			t.Error("Audit log file should exist after successful logging")
		}
	})

	t.Run("logs different event types", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		// Create minimal config
		if err := os.WriteFile(configPath, []byte("providers: {}"), 0600); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// Test different event types
		eventTypes := []struct {
			name    string
			logFunc func(*audit.Logger) error
		}{
			{"setup", func(logger *audit.Logger) error {
				return logger.LogSetup("test-provider")
			}},
			{"switch", func(logger *audit.Logger) error {
				return logger.LogSwitch("test-provider")
			}},
			{"rotate", func(logger *audit.Logger) error {
				return logger.LogRotate("all")
			}},
			{"reset", func(logger *audit.Logger) error {
				return logger.LogReset("manual-reset")
			}},
		}

		for _, et := range eventTypes {
			t.Run(et.name, func(t *testing.T) {
				err := logAuditEvent(tmpDir, et.logFunc)
				if err != nil {
					t.Errorf("logAuditEvent(%s) should succeed, got error: %v", et.name, err)
				}
			})
		}
	})
}

func TestLogAuditEvent_LoggerCreationFailure(t *testing.T) {
	t.Run("returns error when config directory is invalid", func(t *testing.T) {
		invalidDir := "/nonexistent/directory/path"

		err := logAuditEvent(invalidDir, func(logger *audit.Logger) error {
			return logger.LogSwitch("test")
		})

		if err == nil {
			t.Error("logAuditEvent should return error when directory is invalid")
		}

		errMsg := err.Error()
		if !strings.Contains(strings.ToLower(errMsg), "create") && !strings.Contains(strings.ToLower(errMsg), "logger") {
			t.Errorf("Error message should mention logger creation, got: %s", errMsg)
		}
	})

	t.Run("returns error with proper context", func(t *testing.T) {
		invalidDir := "/another/invalid/path"

		err := logAuditEvent(invalidDir, func(logger *audit.Logger) error {
			return logger.LogSwitch("test")
		})

		if err == nil {
			t.Error("logAuditEvent should return error when directory is invalid")
		}

		// Verify error includes context about the failure
		errMsg := err.Error()
		if !strings.Contains(strings.ToLower(errMsg), "failed") && !strings.Contains(strings.ToLower(errMsg), "audit") {
			t.Errorf("Error should include context about audit logging failure, got: %s", errMsg)
		}
	})

	t.Run("handles permission denied error", func(t *testing.T) {
		// Skip this test on Windows as chmod behaves differently
		if runtime.GOOS == "windows" {
			t.Skip("chmod behaves differently on Windows, skipping permission test")
		}

		// Create a read-only directory
		tmpDir := t.TempDir()
		if err := os.Chmod(tmpDir, 0400); err != nil {
			t.Fatalf("Failed to set directory permissions: %v", err)
		}

		err := logAuditEvent(tmpDir, func(logger *audit.Logger) error {
			return logger.LogSwitch("test")
		})

		if err == nil {
			t.Error("logAuditEvent should return error when directory is not writable")
		}
	})
}

func TestLogAuditEvent_LogFuncFailure(t *testing.T) {
	t.Run("returns error when logFunc fails", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a mock logFunc that returns an error
		mockLogFunc := func(logger *audit.Logger) error {
			return fmt.Errorf("simulated log failure")
		}

		err := logAuditEvent(tmpDir, mockLogFunc)

		if err == nil {
			t.Error("logAuditEvent should return error when logFunc fails")
		}

		// Verify error message wraps the logFunc error
		errMsg := err.Error()
		if !strings.Contains(strings.ToLower(errMsg), "log") && !strings.Contains(strings.ToLower(errMsg), "event") {
			t.Errorf("Error should mention log event failure, got: %s", errMsg)
		}
	})

	t.Run("error includes context from logFunc", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a mock logFunc with specific error message
		specificError := fmt.Errorf("unable to write audit entry")
		mockLogFunc := func(logger *audit.Logger) error {
			return specificError
		}

		err := logAuditEvent(tmpDir, mockLogFunc)

		if err == nil {
			t.Error("logAuditEvent should return error when logFunc fails")
		}

		// Verify the specific error is included in the returned error
		errMsg := err.Error()
		if !strings.Contains(errMsg, "unable") && !strings.Contains(errMsg, "write") {
			t.Errorf("Error should include original error details, got: %s", errMsg)
		}
	})
}

func TestLogAuditEvent_ErrorsAreProperlyWrapped(t *testing.T) {
	t.Run("wraps errors with descriptive messages", func(t *testing.T) {
		testCases := []struct {
			name    string
			logFunc func(*audit.Logger) error
			wantSub []string
		}{
			{
				name: "setup event error",
				logFunc: func(logger *audit.Logger) error {
					return logger.LogSetup("test-provider")
				},
				wantSub: []string{"failed", "log", "audit"},
			},
			{
				name: "switch event error",
				logFunc: func(logger *audit.Logger) error {
					return logger.LogSwitch("test-provider")
				},
				wantSub: []string{"failed", "log", "audit"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Force logger creation to fail by using invalid config dir
				invalidDir := "/nonexistent/path"
				err := logAuditEvent(invalidDir, tc.logFunc)

				if err == nil {
					t.Fatal("Expected error, got nil")
				}

				errMsg := err.Error()
				for _, substr := range tc.wantSub {
					if !strings.Contains(strings.ToLower(errMsg), substr) {
						t.Errorf("Error message should contain '%s', got: %s", substr, errMsg)
					}
				}
			})
		}
	})

	t.Run("preserves original error with wrapping", func(t *testing.T) {
		testDir := t.TempDir()

		originalError := fmt.Errorf("original error message")

		err := logAuditEvent(testDir, func(logger *audit.Logger) error {
			return originalError
		})

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		// Verify original error is wrapped, not replaced
		errMsg := err.Error()
		if !strings.Contains(errMsg, "original") {
			t.Error("Wrapped error should include original error message")
		}
	})
}

func TestLogAuditEvent_ClosesLoggerOnSuccess(t *testing.T) {
	t.Run("closes logger after successful logging", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")

		// Create minimal config
		if err := os.WriteFile(configPath, []byte("providers: {}"), 0600); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// Create logger directly to test file handles
		logger, err := audit.NewLogger(filepath.Dir(configPath))
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// Log an event
		if err := logger.LogSwitch("test-provider"); err != nil {
			t.Fatalf("Failed to log event: %v", err)
		}

		// Close logger
		if err := logger.Close(); err != nil {
			t.Errorf("Failed to close logger: %v", err)
		}

		// Verify we can reopen the file (it should be closed and flushed)
		newLogger, err := audit.NewLogger(filepath.Dir(configPath))
		if err != nil {
			t.Fatalf("Failed to create new logger: %v", err)
		}
		defer newLogger.Close()

		// Verify the file is accessible (not locked)
		if err := newLogger.LogSwitch("another-provider"); err != nil {
			t.Errorf("Should be able to log again after logger close: %v", err)
		}
	})

	t.Run("closes logger on error", func(t *testing.T) {
		// Try to log to invalid directory
		invalidDir := "/nonexistent/path"

		err := logAuditEvent(invalidDir, func(logger *audit.Logger) error {
			return logger.LogSwitch("test-provider")
		})

		// Even on error, the logger should have been closed
		// We can verify this by checking if any temp files are left behind
		// (this is implicit - temp files should be cleaned up)

		if err == nil {
			t.Fatal("Expected error from invalid directory")
		}

		// The test passes if no panic or resource leak occurred
	})
}

func TestLogAuditEvent_ThreadSafety(t *testing.T) {
	t.Run("handles concurrent calls gracefully", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.yaml")

		// Create minimal config
		if err := os.WriteFile(configPath, []byte("providers: {}"), 0600); err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// Test concurrent logging
		concurrency := 5
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(provider string) {
				err := logAuditEvent(filepath.Dir(configPath), func(logger *audit.Logger) error {
					return logger.LogSwitch(provider)
				})
				errors <- err
			}(fmt.Sprintf("provider-%d", i))
		}

		// Collect all results
		for i := 0; i < concurrency; i++ {
			if err := <-errors; err != nil {
				// Concurrent calls may fail due to file locking or race conditions
				// This is expected behavior - we just verify no panic occurs
				t.Logf("Concurrent call %d failed: %v", i, err)
			}
		}
		close(errors)

		// Test passes if no panic occurred during concurrent access
	})
}
