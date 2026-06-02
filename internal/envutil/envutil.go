// Package envutil provides small utilities for composing process environment
// variable slices. It is internal to kairo; no stability guarantees.
package envutil

import "strings"

// Merge combines the given env-var slices into a single slice. When the
// same key appears in multiple slices, the value from the *later* slice
// wins and the entry sits at the position of its *last* occurrence. This
// matches the historical kairo behavior used by the harness environment
// pipeline. Malformed entries (without '=' or with an empty key) are
// skipped.
func Merge(envs ...[]string) []string {
	type entry struct {
		key string
		val string
	}
	entries := make([]entry, 0)
	indexByKey := make(map[string]int)

	for _, envSlice := range envs {
		for _, env := range envSlice {
			idx := strings.IndexByte(env, '=')
			if idx <= 0 {
				continue
			}
			key := env[:idx]
			val := env[idx+1:]
			indexByKey[key] = len(entries)
			entries = append(entries, entry{key: key, val: val})
		}
	}

	seen := make(map[string]struct{}, len(indexByKey))
	out := make([]string, 0, len(indexByKey))
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if _, ok := seen[e.key]; ok {
			continue
		}
		seen[e.key] = struct{}{}
		out = append(out, e.key+"="+e.val)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}
