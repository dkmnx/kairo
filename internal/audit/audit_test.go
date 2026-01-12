package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	if logger.path != logPath {
		t.Errorf("path = %v, want %v", logger.path, logPath)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("audit.log file was not created")
	}
}

func TestNewLoggerCreatesFileWithPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	_, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Skip strict permission check on Windows (doesn't support Unix-style 0600)
	if runtime.GOOS != "windows" {
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("File permissions = %o, want 0600", perm)
		}
	}
}

func TestLogSwitch(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("anthropic")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(data) == 0 {
		t.Fatal("audit.log is empty after LogSwitch")
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "switch" {
		t.Errorf("Event = %q, want %q", entry.Event, "switch")
	}

	if entry.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "anthropic")
	}

	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestLogConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	changes := []Change{
		{Field: "api_key", Old: "", New: "***"},
		{Field: "model", Old: "old-model", New: "new-model"},
		{Field: "base_url", Old: "https://old.com", New: "https://new.com"},
	}
	err = logger.LogConfig("zai", "update", changes)
	if err != nil {
		t.Fatalf("LogConfig() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "config" {
		t.Errorf("Event = %q, want %q", entry.Event, "config")
	}

	if entry.Provider != "zai" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "zai")
	}

	if entry.Action != "update" {
		t.Errorf("Action = %q, want %q", entry.Action, "update")
	}

	if len(entry.Changes) != 3 {
		t.Fatalf("Changes count = %d, want 3", len(entry.Changes))
	}

	if entry.Changes[0].Field != "api_key" {
		t.Errorf("Changes[0].Field = %q, want %q", entry.Changes[0].Field, "api_key")
	}
	if entry.Changes[0].New != "***" {
		t.Errorf("Changes[0].New = %q, want %q", entry.Changes[0].New, "***")
	}

	if entry.Changes[1].Field != "model" {
		t.Errorf("Changes[1].Field = %q, want %q", entry.Changes[1].Field, "model")
	}
	if entry.Changes[1].Old != "old-model" {
		t.Errorf("Changes[1].Old = %q, want %q", entry.Changes[1].Old, "old-model")
	}
	if entry.Changes[1].New != "new-model" {
		t.Errorf("Changes[1].New = %q, want %q", entry.Changes[1].New, "new-model")
	}
}

func TestLogRotate(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogRotate("all")
	if err != nil {
		t.Fatalf("LogRotate() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "rotate" {
		t.Errorf("Event = %q, want %q", entry.Event, "rotate")
	}

	if entry.Provider != "all" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "all")
	}
}

func TestLogDefault(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogDefault("minimax")
	if err != nil {
		t.Fatalf("LogDefault() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "default" {
		t.Errorf("Event = %q, want %q", entry.Event, "default")
	}

	if entry.Provider != "minimax" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "minimax")
	}
}

func TestLogReset(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogReset("kimi")
	if err != nil {
		t.Fatalf("LogReset() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "reset" {
		t.Errorf("Event = %q, want %q", entry.Event, "reset")
	}

	if entry.Provider != "kimi" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "kimi")
	}
}

func TestLogSetup(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSetup("deepseek")
	if err != nil {
		t.Fatalf("LogSetup() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "setup" {
		t.Errorf("Event = %q, want %q", entry.Event, "setup")
	}

	if entry.Provider != "deepseek" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "deepseek")
	}
}

func TestLoggerAppendsEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("anthropic")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	err = logger.LogSwitch("zai")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := splitLinesForTest(string(data))
	if len(lines) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(lines))
	}
}

func TestAuditEntryTimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	before := time.Now().UTC()
	err = logger.LogSwitch("test")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}
	after := time.Now().UTC()

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("Timestamp %v not within expected range [%v, %v]", entry.Timestamp, before, after)
	}
}

func TestLoadEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("anthropic")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	err = logger.LogSwitch("zai")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	err = logger.LogDefault("minimax")
	if err != nil {
		t.Fatalf("LogDefault() error = %v", err)
	}

	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("LoadEntries() returned %d entries, want 3", len(entries))
	}

	if entries[0].Event != "switch" || entries[0].Provider != "anthropic" {
		t.Errorf("First entry = %+v, want switch to anthropic", entries[0])
	}

	if entries[1].Event != "switch" || entries[1].Provider != "zai" {
		t.Errorf("Second entry = %+v, want switch to zai", entries[1])
	}

	if entries[2].Event != "default" || entries[2].Provider != "minimax" {
		t.Errorf("Third entry = %+v, want default to minimax", entries[2])
	}
}

func TestLoadEntriesEmptyLog(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("LoadEntries() returned %d entries, want 0", len(entries))
	}
}

func TestLoadEntriesInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	if err := os.WriteFile(logPath, []byte("invalid json\n"), 0600); err != nil {
		t.Fatal(err)
	}

	logger := &Logger{path: logPath}
	_, err := logger.LoadEntries()
	if err == nil {
		t.Error("LoadEntries() should error on invalid JSON")
	}
}

func TestLoadEntriesNonExistent(t *testing.T) {
	logger := &Logger{path: "/nonexistent/audit.log"}
	_, err := logger.LoadEntries()
	if err == nil {
		t.Error("LoadEntries() should error on non-existent file")
	}
}

func splitLinesForTest(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// NEW TESTS: Status and Error fields

func TestAuditEntryHasStatusField(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("test-provider")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Status == "" {
		t.Error("Status field should be present")
	}

	if entry.Status != "success" {
		t.Errorf("Status = %q, want %q", entry.Status, "success")
	}
}

func TestAuditEntryHasErrorField(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSwitch("test-provider")
	if err != nil {
		t.Fatalf("LogSwitch() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Error field should exist but be empty for successful operations
	if entry.Error != "" {
		t.Errorf("Error = %q, want empty string for successful operation", entry.Error)
	}
}

func TestLogSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	details := map[string]interface{}{
		"previous_provider": "old-provider",
		"cli_version":       "1.0.0",
	}

	err = logger.LogSuccess("switch", "new-provider", details)
	if err != nil {
		t.Fatalf("LogSuccess() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "switch" {
		t.Errorf("Event = %q, want %q", entry.Event, "switch")
	}

	if entry.Provider != "new-provider" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "new-provider")
	}

	if entry.Status != "success" {
		t.Errorf("Status = %q, want %q", entry.Status, "success")
	}

	if entry.Error != "" {
		t.Errorf("Error = %q, want empty string", entry.Error)
	}

	if len(entry.Details) == 0 {
		t.Error("Details should not be empty")
	}

	if entry.Details["previous_provider"] != "old-provider" {
		t.Errorf("Details[previous_provider] = %v, want %v", entry.Details["previous_provider"], "old-provider")
	}

	if entry.Details["cli_version"] != "1.0.0" {
		t.Errorf("Details[cli_version] = %v, want %v", entry.Details["cli_version"], "1.0.0")
	}
}

func TestLogFailure(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	details := map[string]interface{}{
		"attempted_provider": "broken-provider",
		"reason":             "connection timeout",
	}

	err = logger.LogFailure("switch", "broken-provider", "failed to connect: timeout", details)
	if err != nil {
		t.Fatalf("LogFailure() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Event != "switch" {
		t.Errorf("Event = %q, want %q", entry.Event, "switch")
	}

	if entry.Provider != "broken-provider" {
		t.Errorf("Provider = %q, want %q", entry.Provider, "broken-provider")
	}

	if entry.Status != "failure" {
		t.Errorf("Status = %q, want %q", entry.Status, "failure")
	}

	if entry.Error == "" {
		t.Error("Error should not be empty for failed operation")
	}

	if entry.Error != "failed to connect: timeout" {
		t.Errorf("Error = %q, want %q", entry.Error, "failed to connect: timeout")
	}

	if len(entry.Details) == 0 {
		t.Error("Details should not be empty")
	}

	if entry.Details["attempted_provider"] != "broken-provider" {
		t.Errorf("Details[attempted_provider] = %v, want %v", entry.Details["attempted_provider"], "broken-provider")
	}
}

func TestLogSuccessWithNilDetails(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogSuccess("setup", "test-provider", nil)
	if err != nil {
		t.Fatalf("LogSuccess() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Status != "success" {
		t.Errorf("Status = %q, want %q", entry.Status, "success")
	}

	// Details should be nil or empty map when nil is passed
	if len(entry.Details) > 0 {
		t.Errorf("Details = %v, want empty or nil", entry.Details)
	}
}

func TestLogFailureWithNilDetails(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	err = logger.LogFailure("setup", "test-provider", "setup failed", nil)
	if err != nil {
		t.Fatalf("LogFailure() error = %v", err)
	}

	data, err := os.ReadFile(logger.path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if entry.Status != "failure" {
		t.Errorf("Status = %q, want %q", entry.Status, "failure")
	}

	if entry.Error != "setup failed" {
		t.Errorf("Error = %q, want %q", entry.Error, "setup failed")
	}
}

func TestLoggerClose(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	// Log something before closing
	err = logger.LogSwitch("test-before-close")
	if err != nil {
		t.Fatalf("LogSwitch() before close error = %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify entries were written before close
	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("LoadEntries() returned %d entries, want 1", len(entries))
	}

	if entries[0].Event != "switch" {
		t.Errorf("Event = %q, want %q", entries[0].Event, "switch")
	}
}

func TestLoggerConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := NewLogger(tmpDir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	numEntriesPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numEntriesPerGoroutine; j++ {
				err := logger.LogSwitch(fmt.Sprintf("provider-%d-%d", id, j))
				if err != nil {
					t.Errorf("LogSwitch() error in goroutine %d: %v", id, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all entries were written
	entries, err := logger.LoadEntries()
	if err != nil {
		t.Fatalf("LoadEntries() error = %v", err)
	}

	expectedEntries := numGoroutines * numEntriesPerGoroutine
	if len(entries) != expectedEntries {
		t.Errorf("LoadEntries() returned %d entries, want %d", len(entries), expectedEntries)
	}
}
