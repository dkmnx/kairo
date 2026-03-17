package config

import (
	"log"
	"sort"
	"strings"
)

func ParseSecrets(secrets string) map[string]string {
	result := make(map[string]string)
	for lineNum, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			if key == "" {
				// SECURITY: Do not log the value (which may contain a secret)
				log.Printf("Warning: skipping malformed secret entry at line %d: empty key", lineNum+1)

				continue
			}
			if value == "" {
				// SECURITY: Do not log the key (which may be a secret identifier)
				log.Printf("Warning: skipping malformed secret entry at line %d: empty value", lineNum+1)

				continue
			}
			if strings.Contains(key, "\n") || strings.Contains(value, "\n") {
				// SECURITY: Do not log the key or value
				log.Printf("Warning: skipping malformed secret entry at line %d: contains newline", lineNum+1)

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
