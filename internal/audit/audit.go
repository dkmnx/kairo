package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AuditEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Event     string                 `json:"event"`
	Provider  string                 `json:"provider,omitempty"`
	Action    string                 `json:"action,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Changes   []Change               `json:"changes,omitempty"`
}

type Change struct {
	Field string `json:"field"`
	Old   string `json:"old,omitempty"`
	New   string `json:"new,omitempty"`
}

type Logger struct {
	path string
	f    *os.File
	mu   sync.Mutex
}

func NewLogger(configDir string) (*Logger, error) {
	logPath := filepath.Join(configDir, "audit.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	return &Logger{path: logPath, f: f}, nil
}

// Close closes the log file. Must be called when the logger is no longer needed.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		return l.f.Close()
	}
	return nil
}

// LogSwitch logs a provider switch event to the audit log.
//
// This method creates an audit entry recording when a user switches to a
// different provider. The entry includes the provider name and timestamp.
//
// Parameters:
//   - provider: Name of the provider being switched to
//
// Returns:
//   - error: Returns error if unable to write to audit log
//
// Error conditions:
//   - Returns error when audit log file cannot be written (e.g., permissions, disk full)
//
// Thread Safety: Thread-safe (uses mutex in writeEntry)
func (l *Logger) LogSwitch(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "switch",
		Provider:  provider,
		Status:    "success",
	}
	return l.writeEntry(entry)
}

// LogConfig logs a configuration change event to the audit log.
//
// This method creates an audit entry recording when a provider's configuration
// is modified (added, updated, or changed). The entry includes provider
// name, action type (add/update), and list of changed fields with
// old and new values.
//
// Parameters:
//   - provider: Name of the provider being configured
//   - action: Type of configuration action (e.g., "add", "update", "change")
//   - changes: List of fields that were changed with old/new values
//
// Returns:
//   - error: Returns error if unable to write to audit log
//
// Error conditions:
//   - Returns error when audit log file cannot be written (e.g., permissions, disk full)
//
// Thread Safety: Thread-safe (uses mutex in writeEntry)
func (l *Logger) LogConfig(provider, action string, changes []Change) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "config",
		Provider:  provider,
		Action:    action,
		Status:    "success",
		Changes:   changes,
	}
	return l.writeEntry(entry)
}

func (l *Logger) LogRotate(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "rotate",
		Provider:  provider,
		Status:    "success",
	}
	return l.writeEntry(entry)
}

func (l *Logger) LogDefault(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "default",
		Provider:  provider,
		Status:    "success",
	}
	return l.writeEntry(entry)
}

func (l *Logger) LogReset(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "reset",
		Provider:  provider,
		Status:    "success",
	}
	return l.writeEntry(entry)
}

func (l *Logger) LogSetup(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "setup",
		Provider:  provider,
		Status:    "success",
	}
	return l.writeEntry(entry)
}

// LogSuccess logs a successful operation with optional details
func (l *Logger) LogSuccess(event, provider string, details map[string]interface{}) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     event,
		Provider:  provider,
		Status:    "success",
		Details:   details,
	}
	return l.writeEntry(entry)
}

// LogMigration logs a configuration file migration event
func (l *Logger) LogMigration(details map[string]interface{}) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "migration",
		Status:    "success",
		Details:   details,
	}
	return l.writeEntry(entry)
}

// LogFailure logs a failed operation with error details
func (l *Logger) LogFailure(event, provider, errMsg string, details map[string]interface{}) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     event,
		Provider:  provider,
		Status:    "failure",
		Error:     errMsg,
		Details:   details,
	}
	return l.writeEntry(entry)
}

// writeEntry writes an audit entry to the log file.
// The entry is serialized to JSON, written to the file with a newline,
// and then synced to disk to ensure durability even in crash scenarios.
func (l *Logger) writeEntry(entry AuditEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	// Reopen file if it was closed
	if l.f == nil {
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		l.f = f
	}

	_, err = l.f.Write(data)
	if err != nil {
		return err
	}
	_, err = l.f.WriteString("\n")
	if err != nil {
		return err
	}
	// Sync to ensure data is written to disk
	if err := l.f.Sync(); err != nil {
		return err
	}
	return nil
}

// LoadEntries reads and parses all audit entries from the log file.
//
// This method reads the entire audit log, parses each JSON line, and returns
// all entries as a slice. Empty lines are skipped. The log file is
// opened in read-only mode.
//
// Parameters:
//   - none (method receiver only)
//
// Returns:
//   - []AuditEntry: Slice of all audit entries in chronological order
//   - error: Returns error if unable to read or parse audit log
//
// Error conditions:
//   - Returns error when audit log file cannot be read (e.g., permissions, file not found)
//   - Returns error if any JSON line cannot be parsed (e.g., corrupted log file)
//
// Thread Safety: Not thread-safe (log file may be modified concurrently by writes)
// Security Notes: Returns all audit entries including potentially sensitive data
func (l *Logger) LoadEntries() ([]AuditEntry, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}
	var entries []AuditEntry
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func splitLines(s string) []string {
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
