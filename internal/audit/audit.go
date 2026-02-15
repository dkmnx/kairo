package audit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
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
	// Context fields for traceability
	Hostname  string `json:"hostname,omitempty"`
	Username  string `json:"username,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type Change struct {
	Field string `json:"field"`
	Old   string `json:"old,omitempty"`
	New   string `json:"new,omitempty"`
}

type Logger struct {
	path      string
	f         *os.File
	mu        sync.RWMutex
	hostname  string
	username  string
	sessionID string
}

func NewLogger(configDir string) (*Logger, error) {
	logPath := filepath.Join(configDir, "audit.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	// Capture hostname
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// Capture username
	username := "unknown"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	// Generate unique session ID
	sessionID := generateSessionID()

	return &Logger{
		path:      logPath,
		f:         f,
		hostname:  hostname,
		username:  username,
		sessionID: sessionID,
	}, nil
}

// generateSessionID generates a unique session identifier.
func generateSessionID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}

// Close closes the log file. Must be called when the logger is no longer needed.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.f != nil {
		err := l.f.Close()
		l.f = nil
		return err
	}
	return nil
}

// RotateOptions contains configuration options for log rotation.
type RotateOptions struct {
	// MaxSize is the maximum size in bytes before rotating (default: 10MB)
	MaxSize int64
	// MaxAge is the maximum age in days before rotating (default: 30 days)
	MaxAge int
	// MaxBackups is the number of old log files to keep (default: 5)
	MaxBackups int
}

// DefaultRotateOptions returns sensible defaults for log rotation.
func DefaultRotateOptions() RotateOptions {
	return RotateOptions{
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxAge:     30,               // 30 days
		MaxBackups: 5,
	}
}

// RotateLog rotates the audit log if it exceeds size or age limits.
// Old log files are renamed with a timestamp suffix.
// Returns true if rotation occurred, false otherwise.
func (l *Logger) RotateLog(opts ...RotateOptions) (bool, error) {
	options := DefaultRotateOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check file stats
	info, err := os.Stat(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// Check size limit
	shouldRotate := info.Size() > options.MaxSize

	// Check age limit
	if !shouldRotate {
		age := time.Since(info.ModTime())
		shouldRotate = age > time.Duration(options.MaxAge)*24*time.Hour
	}

	if !shouldRotate {
		return false, nil
	}

	// Close current file if open
	if l.f != nil {
		l.f.Close()
		l.f = nil
	}

	// Generate timestamped backup name
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupPath := filepath.Join(filepath.Dir(l.path),
		fmt.Sprintf("audit.%s.log", timestamp))

	// Rename current log to backup
	if err := os.Rename(l.path, backupPath); err != nil {
		return false, err
	}

	// Open new log file
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return false, err
	}
	l.f = f

	// Clean up old backups
	l.cleanupOldBackups(options.MaxBackups)

	return true, nil
}

// cleanupOldBackups removes old audit log backups beyond the limit.
func (l *Logger) cleanupOldBackups(maxBackups int) {
	if maxBackups <= 0 {
		return
	}

	dir := filepath.Dir(l.path)
	pattern := filepath.Join(dir, "audit.*.log")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	if len(matches) <= maxBackups {
		return
	}

	// Sort by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil {
			files = append(files, fileInfo{path: m, modTime: info.ModTime()})
		}
	}

	// Sort by mod time
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.After(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Remove oldest files beyond limit
	for i := 0; i < len(files)-maxBackups; i++ {
		os.Remove(files[i].path)
	}
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
	l.enrichWithContext(&entry)
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
	l.enrichWithContext(&entry)
	return l.writeEntry(entry)
}

func (l *Logger) LogRotate(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "rotate",
		Provider:  provider,
		Status:    "success",
	}
	l.enrichWithContext(&entry)
	return l.writeEntry(entry)
}

func (l *Logger) LogDefault(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "default",
		Provider:  provider,
		Status:    "success",
	}
	l.enrichWithContext(&entry)
	return l.writeEntry(entry)
}

func (l *Logger) LogReset(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "reset",
		Provider:  provider,
		Status:    "success",
	}
	l.enrichWithContext(&entry)
	return l.writeEntry(entry)
}

func (l *Logger) LogSetup(provider string) error {
	entry := AuditEntry{
		Timestamp: time.Now().UTC(),
		Event:     "setup",
		Provider:  provider,
		Status:    "success",
	}
	l.enrichWithContext(&entry)
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
	l.enrichWithContext(&entry)
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
	l.enrichWithContext(&entry)
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
	l.enrichWithContext(&entry)
	return l.writeEntry(entry)
}

// enrichWithContext adds hostname, username, and session ID to an audit entry.
func (l *Logger) enrichWithContext(entry *AuditEntry) {
	entry.Hostname = l.hostname
	entry.Username = l.username
	entry.SessionID = l.sessionID
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
// Thread Safety: Thread-safe (uses RWMutex to allow concurrent reads)
// Security Notes: Returns all audit entries including potentially sensitive data
func (l *Logger) LoadEntries() ([]AuditEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

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

// splitLines splits a string by newline character into a slice of strings.
//
// This function scans the input string character by character, splitting on
// newline characters ('\n'). Each line (including empty lines) is
// added to the result slice. Used for parsing audit log files
// where each JSON line represents a separate audit entry.
//
// Parameters:
//   - s: String to split by newlines
//
// Returns:
//   - []string: Slice of strings, one per line in original order
//
// Error conditions: None
//
// Thread Safety: Thread-safe (pure function, no shared state)
// Performance Notes: O(n) where n is string length, creates one slice with capacity
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
