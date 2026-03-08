package config

import (
	"log"
	"sort"
	"strings"
)

func ParseSecrets(secrets string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			if key == "" || value == "" || strings.Contains(key, "\n") || strings.Contains(value, "\n") {
				log.Printf("Warning: skipping malformed secret entry: %q", line)

				continue
			}
			result[key] = value
		}
	}

	return result
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
