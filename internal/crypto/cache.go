package crypto

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dkmnx/kairo/internal/errors"
)

const secretsCacheTTL = 60 * time.Second

type cachedSecrets struct {
	secrets  map[string]string
	loadedAt time.Time
	modTime  time.Time
}

// SecretsCache provides a short-lived cache for decrypted secrets, keyed by
// config directory. Entries auto-invalidate when the TTL expires or when the
// secrets.age file modification time changes.
type SecretsCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedSecrets
	ttl     time.Duration
}

// NewSecretsCache creates a SecretsCache with the default TTL.
func NewSecretsCache() *SecretsCache {
	return &SecretsCache{
		entries: make(map[string]*cachedSecrets),
		ttl:     secretsCacheTTL,
	}
}

// secretsFilePath returns the expected secrets.age path for a config directory.
func secretsFilePath(configDir string) string {
	return filepath.Join(configDir, "secrets.age")
}

// Get returns the cached secrets for configDir, loading them fresh if the
// entry is missing, expired, or the secrets.age file has changed.
func (sc *SecretsCache) Get(ctx context.Context, configDir, keyPath string) (map[string]string, error) {
	secretsPath := secretsFilePath(configDir)
	currentModTime := getFileModTime(secretsPath)

	sc.mu.RLock()
	entry, exists := sc.entries[configDir]
	if exists && time.Since(entry.loadedAt) < sc.ttl && !currentModTime.After(entry.modTime) {
		secrets := copySecrets(entry.secrets)
		sc.mu.RUnlock()

		return secrets, nil
	}
	sc.mu.RUnlock()

	// Cache miss: decrypt fresh.
	decrypted, err := DecryptSecrets(ctx, secretsPath, keyPath)
	if err != nil {
		return nil, errors.WrapError(errors.CryptoError,
			"failed to decrypt secrets", err).
			WithContext("path", secretsPath).
			WithContext("key_path", keyPath)
	}

	parsed := parseSecrets(decrypted)

	sc.mu.Lock()
	sc.entries[configDir] = &cachedSecrets{
		secrets:  parsed,
		loadedAt: time.Now(),
		modTime:  currentModTime,
	}
	sc.mu.Unlock()

	return copySecrets(parsed), nil
}

// Invalidate removes the cached entry for configDir, forcing a fresh decrypt
// on the next Get call.
func (sc *SecretsCache) Invalidate(configDir string) {
	sc.mu.Lock()
	delete(sc.entries, configDir)
	sc.mu.Unlock()
}

// getFileModTime returns the modification time of the file at path, or a
// zero time if the file does not exist or cannot be read.
func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}

	return info.ModTime()
}

// parseSecrets parses a decrypted secrets string into a map.
// Each line is "KEY=VALUE" format; blank lines and comments are skipped.
func parseSecrets(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range splitLines(content) {
		line = trimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if idx := indexOf(line, '='); idx > 0 {
			key := line[:idx]
			value := line[idx+1:]
			result[key] = value
		}
	}

	return result
}

// copySecrets returns a shallow copy of the secrets map.
func copySecrets(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}

// splitLines splits a string into lines. Avoids importing strings to keep
// the crypto package dependency-free of stdlib text utilities.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

// trimSpace returns s with leading and trailing whitespace removed.
func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && isSpace(s[start]) {
		start++
	}
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// indexOf returns the index of the first occurrence of b in s, or -1.
func indexOf(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}

	return -1
}

// isSpace reports whether b is a whitespace character.
func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}
