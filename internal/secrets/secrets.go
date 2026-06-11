// Package secrets parses and formats key-value secret entries.
package secrets

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// WarnFunc is called when malformed entries are encountered during parsing.
// Defaults to printing to stderr. Tests should save and restore this value
// (e.g., via t.Cleanup) to avoid leaking between test cases.
var WarnFunc = defaultWarn

func defaultWarn(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// SetWarnFunc replaces WarnFunc and returns a function that restores the
// previous value. Intended for use in tests: defer secrets.SetWarnFunc(old).
func SetWarnFunc(fn func(string)) func() {
	old := WarnFunc
	WarnFunc = fn

	return func() { WarnFunc = old }
}

// Result holds parsed secrets along with parsing metadata.
type Result struct {
	Secrets      map[string]string
	SkippedCount int
	Warnings     []string
}

// Parse extracts key-value pairs from the given content string.
func Parse(content string) map[string]string {
	return ParseWithStats(content).Secrets
}

// ParseWithStats extracts key-value pairs and returns detailed parsing statistics.
// Malformed entries are counted and reported via warnings; they are not
// preserved on the Result, since the secrets file is regenerated on every
// write and would otherwise carry stale unparseable content.
func ParseWithStats(content string) Result {
	result := make(map[string]string)
	var warnings []string
	var skippedCount int
	for lineNum, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			skippedCount++

			continue
		}
		key, value := parts[0], parts[1]
		if key == "" {
			warnings = append(warnings, fmt.Sprintf("skipping malformed secret entry at line %d: empty key", lineNum+1))
			skippedCount++

			continue
		}
		if value == "" {
			warnings = append(warnings, fmt.Sprintf("skipping malformed secret entry at line %d: empty value", lineNum+1))
			skippedCount++

			continue
		}
		if strings.Contains(key, "\n") || strings.Contains(value, "\n") {
			warnings = append(warnings, fmt.Sprintf("skipping malformed secret entry at line %d: contains newline", lineNum+1))
			skippedCount++

			continue
		}
		result[key] = value
	}

	for _, w := range warnings {
		WarnFunc(w)
	}

	return Result{Secrets: result, SkippedCount: skippedCount, Warnings: warnings}
}

// Format serializes a secrets map into sorted key=value lines.
func Format(secrets map[string]string) string {
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		value := secrets[key]
		if key != "" && value != "" {
			b.WriteString(key)
			b.WriteString("=")
			b.WriteString(value)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// EnvVars converts a secrets map into environment variable strings.
func EnvVars(secrets map[string]string) []string {
	envVars := make([]string, 0, len(secrets))
	for k, v := range secrets {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	return envVars
}
