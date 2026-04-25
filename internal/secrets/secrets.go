package secrets

import (
	"fmt"
	"sort"
	"strings"
)

type Result struct {
	Secrets      map[string]string
	SkippedCount int
	Warnings     []string
	RawLines     []string
}

func Parse(content string) map[string]string {
	return ParseWithStats(content).Secrets
}

func ParseWithStats(content string) Result {
	result := make(map[string]string)
	var warnings []string
	var rawLines []string
	var skippedCount int
	for lineNum, line := range strings.Split(content, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			skippedCount++
			rawLines = append(rawLines, line)

			continue
		}
		key, value := parts[0], parts[1]
		if key == "" {
			warnings = append(warnings, fmt.Sprintf("skipping malformed secret entry at line %d: empty key", lineNum+1))
			skippedCount++
			rawLines = append(rawLines, line)

			continue
		}
		if value == "" {
			warnings = append(warnings, fmt.Sprintf("skipping malformed secret entry at line %d: empty value", lineNum+1))
			skippedCount++
			rawLines = append(rawLines, line)

			continue
		}
		if strings.Contains(key, "\n") || strings.Contains(value, "\n") {
			warnings = append(warnings, fmt.Sprintf("skipping malformed secret entry at line %d: contains newline", lineNum+1))
			skippedCount++
			rawLines = append(rawLines, line)

			continue
		}
		result[key] = value
	}

	return Result{Secrets: result, SkippedCount: skippedCount, Warnings: warnings, RawLines: rawLines}
}

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
