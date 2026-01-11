package config

import (
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
			// Skip entries with empty keys or newlines in key or value (malformed input)
			if key == "" || strings.Contains(key, "\n") || strings.Contains(value, "\n") {
				continue
			}
			result[key] = value
		}
	}
	return result
}
