package config

import (
	"strings"
)

// ParseSecrets parses a newline-separated list of key=value pairs into a map.
// Empty lines and lines without '=' are ignored.
func ParseSecrets(secrets string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
