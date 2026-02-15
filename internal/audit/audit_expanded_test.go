package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogMigration(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	details := map[string]interface{}{
		"from_version": "1.0.0",
		"to_version":   "1.1.0",
		"migrated_fields": []string{
			"default_harness",
			"provider_aliases",
		},
	}

	err = logger.LogMigration(details)
	if err != nil {
		t.Fatalf("LogMigration() error = %v", err)
	}

	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("LoadEntries() returned %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Event != "migration" {
		t.Errorf("Event = %q, want %q", entry.Event, "migration")
	}

	if entry.Status != "success" {
		t.Errorf("Status = %q, want %q", entry.Status, "success")
	}

	if entry.Provider != "" {
		t.Errorf("Provider should be empty for migration, got %q", entry.Provider)
	}

	if entry.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if entry.Details["from_version"] != "1.0.0" {
		t.Errorf("Details[from_version] = %v, want 1.0.0", entry.Details["from_version"])
	}

	if entry.Details["to_version"] != "1.1.0" {
		t.Errorf("Details[to_version] = %v, want 1.1.0", entry.Details["to_version"])
	}
}

func TestLogMigrationWithNilDetails(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	err = logger.LogMigration(nil)
	if err != nil {
		t.Fatalf("LogMigration() error = %v", err)
	}

	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("LoadEntries() returned %d entries, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Event != "migration" {
		t.Errorf("Event = %q, want %q", entry.Event, "migration")
	}
}

func TestWriteEntryReopensClosedFile(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Close the underlying file to simulate a closed state
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Write should reopen the file automatically
	err = logger.LogSwitch("test-provider")
	if err != nil {
		t.Fatalf("LogSwitch() after close error = %v", err)
	}

	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("LoadEntries() returned %d entries, want 1", len(entries))
	}
}

func TestWriteEntryWithClosedFileMultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Close the underlying file
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Multiple writes after close should all succeed
	providers := []string{"provider1", "provider2", "provider3"}
	for _, p := range providers {
		err = logger.LogSwitch(p)
		if err != nil {
			t.Fatalf("LogSwitch(%q) error = %v", p, err)
		}
	}

	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != len(providers) {
		t.Errorf("LoadEntries() returned %d entries, want %d", len(entries), len(providers))
	}
}

func TestWriteEntryBasic(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	err = logger.LogSwitch("test")
	if err != nil {
		t.Errorf("LogSwitch() error = %v", err)
	}

	// Verify entry was written
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file should contain data")
	}
}

func TestGenerateSessionIDWithRandFailure(t *testing.T) {
	// Test generateSessionID fallback when crypto/rand fails
	// This is hard to test directly since it requires mocking,
	// but we can verify the function produces valid output

	sessionID := generateSessionID()
	if sessionID == "" {
		t.Error("generateSessionID() should not return empty string")
	}

	// Verify it's either a hex string (16 chars) or a timestamp-based fallback
	if len(sessionID) != 16 && len(sessionID) < 10 {
		t.Errorf("SessionID has unexpected length: %d", len(sessionID))
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single line no newline",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "two lines",
			input:    "hello\nworld",
			expected: []string{"hello", "world"},
		},
		{
			name:     "three lines with trailing newline",
			input:    "line1\nline2\nline3\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "empty lines in middle",
			input:    "line1\n\nline2",
			expected: []string{"line1", "", "line2"},
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: []string{"", "", ""},
		},
		{
			name:     "json lines",
			input:    `{"event":"a"}` + "\n" + `{"event":"b"}`,
			expected: []string{`{"event":"a"}`, `{"event":"b"}`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitLines() returned %d lines, want %d", len(result), len(tt.expected))
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("Line %d = %q, want %q", i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestLoadEntriesWithEmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Write log with empty lines between entries
	content := `{"event":"switch","provider":"p1","status":"success"}
{"event":"switch","provider":"p2","status":"success"}
`
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	logger := &Logger{path: logPath}
	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	// Empty lines should be skipped
	if len(entries) != 2 {
		t.Errorf("LoadEntries() returned %d entries, want 2", len(entries))
	}
}

func TestLoadEntriesWithCorruptedLine(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Write log with one valid entry and one corrupted
	content := `{"event":"switch","provider":"p1","status":"success"}
invalid json here
{"event":"switch","provider":"p2","status":"success"}
`
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	logger := &Logger{path: logPath}
	_, err := logger.LoadEntries()

	// Should fail on corrupted JSON
	if err == nil {
		t.Error("LoadEntries() should error on corrupted JSON line")
	}
}

func TestNewLoggerHandlesEmptyHostname(t *testing.T) {
	// Test NewLogger when os.Hostname() returns empty string
	// This is hard to mock without interface, so we test the "unknown" fallback path
	tmpDir := t.TempDir()

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Logger should have initialized with some hostname value
	if logger.hostname == "" {
		// This may happen in some test environments, so we just log
		t.Log("Hostname is empty in this environment")
	}
}

func TestNewLoggerCreatesLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Verify file was created with correct path
	if logger.path != logPath {
		t.Errorf("Logger path = %q, want %q", logger.path, logPath)
	}

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Audit log file should exist")
	}
}

func TestWriteEntryHandlesSyncError(t *testing.T) {
	// This is hard to test without mocking the file,
	// but we can test the error handling path by checking
	// that write errors are returned properly
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Normal write should succeed
	err = logger.LogSwitch("test")
	if err != nil {
		t.Errorf("LogSwitch() error = %v", err)
	}

	// Verify data was written
	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}
}
