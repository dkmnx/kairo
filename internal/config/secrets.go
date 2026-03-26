package config

import (
	"log"
	"sort"
	"strings"
)

func ParseSecrets(secrets string) map[string]string {
	result := make(map[string]string)
	for _, entry := range parseSecretsEntries(secrets) {
		result[entry.key] = entry.value
	}

	return result
}

type secretEntry struct {
	key   string
	value string
}

func parseSecretsEntries(secrets string) []secretEntry {
	var entries []secretEntry
	for lineNum, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if key == "" {
			log.Printf("Warning: skipping malformed secret entry at line %d: empty key", lineNum+1)

			continue
		}
		if value == "" {
			log.Printf("Warning: skipping malformed secret entry at line %d: empty value", lineNum+1)

			continue
		}
		if strings.Contains(key, "\n") || strings.Contains(value, "\n") {
			log.Printf("Warning: skipping malformed secret entry at line %d: contains newline", lineNum+1)

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
