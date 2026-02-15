package cmd

import (
	"strings"
)

// mergeEnvVars merges environment variable slices, deduplicating by key.
// If duplicate keys are found, the last value wins (preserves order of precedence).
// Env vars should be in "KEY=VALUE" format.
func mergeEnvVars(envs ...[]string) []string {
	// Use map to track key -> index in result for O(1) lookup and removal
	seen := make(map[string]int)
	var result []string

	for _, envSlice := range envs {
		for _, env := range envSlice {
			// Extract the key (everything before the first '=')
			idx := strings.IndexByte(env, '=')
			// Check for invalid format: no '=' or '=' is first or last char
			if idx <= 0 || idx == len(env)-1 {
				// Invalid format (no key or no value), skip
				continue
			}
			key := env[:idx]

			// Remove any previous occurrence of this key
			if prevIdx, exists := seen[key]; exists {
				// Remove the previous entry by swapping with last and truncating
				lastIdx := len(result) - 1
				result[prevIdx] = result[lastIdx]
				// Update the index for the moved entry
				if prevIdx != lastIdx {
					seen[result[prevIdx][:strings.IndexByte(result[prevIdx], '=')]] = prevIdx
				}
				result = result[:lastIdx]
			}

			// Add the new entry and record its index
			result = append(result, env)
			seen[key] = len(result) - 1
		}
	}

	return result
}
