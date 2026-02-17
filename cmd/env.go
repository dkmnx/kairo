package cmd

import (
	"strings"
)

// mergeEnvVars merges environment variable slices, deduplicating by key.
// If duplicate keys are found, last value wins (preserves order of precedence).
// Env vars should be in "KEY=VALUE" format.
func mergeEnvVars(envs ...[]string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, envSlice := range envs {
		for _, env := range envSlice {
			// Extract key (everything before the first '=')
			idx := strings.IndexByte(env, '=')
			// Check for invalid format: no '=' or '=' is first or last char
			if idx <= 0 || idx == len(env)-1 {
				// Invalid format (no key or no value), skip
				continue
			}
			key := env[:idx]

			// Remove previous occurrence if it exists
			if seen[key] {
				// Find and remove the previous entry by filtering
				var filtered []string
				for _, e := range result {
					eIdx := strings.IndexByte(e, '=')
					if eIdx > 0 {
						eKey := e[:eIdx]
						if eKey == key {
							// Skip this one (it's the duplicate)
							continue
						}
					}
					filtered = append(filtered, e)
				}
				result = filtered
			}

			// Add the new entry and mark key as seen
			result = append(result, env)
			seen[key] = true
		}
	}

	return result
}
