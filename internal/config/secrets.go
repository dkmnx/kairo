package config

import (
	"log"
	"sort"
	"strings"
)

// ParseSecrets parses a newline-separated list of key=value pairs into a map.
// Empty lines and lines without '=' are silently ignored.
// Lines without '=' are treated as malformed and skipped to handle edge cases gracefully.
// This allows the function to continue processing other valid entries even if some lines are malformed.
// Keys or values containing newlines are skipped as malformed input.
func ParseSecrets(secrets string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			// Skip entries with empty keys or values, or newlines in key or value (malformed input)
			if key == "" || value == "" || strings.Contains(key, "\n") || strings.Contains(value, "\n") {
				log.Printf("Warning: skipping malformed secret entry: %q", line)
				continue
			}
			result[key] = value
		}
	}
	return result
}

// FormatSecrets formats a secrets map into a string suitable for file storage.
// Keys are sorted for deterministic output. Empty keys or values are skipped.
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
