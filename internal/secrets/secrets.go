// Package secrets parses and formats key-value secret entries.
package secrets

import (
	"fmt"
	"sort"
	"strings"
)

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
	for lineNum, line := range strings.Split(content, "\n") {
		if line == "" {
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
