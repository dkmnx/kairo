package config

import (
	"log"
	"sort"
	"strings"
)

type SecretsMap struct {
	data map[string][]byte
}

func NewSecretsMap() *SecretsMap {
	return &SecretsMap{data: make(map[string][]byte)}
}

func (s *SecretsMap) Get(key string) (string, bool) {
	val, ok := s.data[key]
	if !ok {
		return "", false
	}

	return string(val), true
}

func (s *SecretsMap) Set(key, value string) {
	s.data[key] = []byte(value)
}

func (s *SecretsMap) Delete(key string) {
	if val, ok := s.data[key]; ok {
		for i := range val {
			val[i] = 0
		}
		delete(s.data, key)
	}
}

func (s *SecretsMap) Range(fn func(key, value string) bool) {
	for k, v := range s.data {
		if !fn(k, string(v)) {
			break
		}
	}
}

func (s *SecretsMap) Len() int {
	return len(s.data)
}

func (s *SecretsMap) Clear() {
	for k := range s.data {
		for i := range s.data[k] {
			s.data[k][i] = 0
		}
		delete(s.data, k)
	}
}

func (s *SecretsMap) Close() error {
	s.Clear()

	return nil
}

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

func ParseSecretsToSecureMap(secrets string) *SecretsMap {
	result := NewSecretsMap()
	for _, line := range strings.Split(secrets, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			if key == "" || value == "" || strings.Contains(key, "\n") || strings.Contains(value, "\n") {
				continue
			}
			result.Set(key, value)
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

func FormatSecretsMap(secrets *SecretsMap) string {
	var keys []string
	secrets.Range(func(key, _ string) bool {
		keys = append(keys, key)

		return true
	})
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		if value, ok := secrets.Get(key); ok && key != "" && value != "" {
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(value)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}
