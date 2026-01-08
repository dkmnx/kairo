package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
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
}

func NewLogger(configDir string) (*Logger, error) {
	logPath := filepath.Join(configDir, "audit.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &Logger{path: logPath}, nil
}

func (l *Logger) LogSwitch(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "switch",
		Provider:  provider,
		Status:    "success",
	}
	return l.writeEntry(entry)
}

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

func (l *Logger) writeEntry(entry AuditEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	_, err = f.WriteString("\n")
	return err
}

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
