package config

import (
	"log"
	"sort"
	"strings"
)

type SecretsResult struct {
	Secrets      map[string]string
	SkippedCount int
}

func ParseSecrets(secrets string) map[string]string {
	result := ParseSecretsWithStats(secrets)

	return result.Secrets
}

func ParseSecretsWithStats(secrets string) SecretsResult {
	result := make(map[string]string)
	skippedCount := 0
	for _, entry := range parseSecretsEntries(secrets, &skippedCount) {
		result[entry.key] = entry.value
	}

	return SecretsResult{Secrets: result, SkippedCount: skippedCount}
}

type secretEntry struct {
	key   string
	value string
}

func parseSecretsEntries(secrets string, skippedCount *int) []secretEntry {
	var entries []secretEntry
	for lineNum, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			*skippedCount++

			continue
		}
		key, value := parts[0], parts[1]
		if key == "" {
			log.Printf("Warning: skipping malformed secret entry at line %d: empty key", lineNum+1)
			*skippedCount++

			continue
		}
		if value == "" {
			log.Printf("Warning: skipping malformed secret entry at line %d: empty value", lineNum+1)
			*skippedCount++

			continue
		}
		if strings.Contains(key, "\n") || strings.Contains(value, "\n") {
			log.Printf("Warning: skipping malformed secret entry at line %d: contains newline", lineNum+1)
			*skippedCount++

			continue
		}
		entries = append(entries, secretEntry{key: key, value: value})
	}

	return entries
}

func FormatSecrets(secrets map[string]string) string {
	keys := make([]string, 0, len(secrets))
	for key := range secrets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		value := secrets[key]
		if key != "" && value != "" {
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(value)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
