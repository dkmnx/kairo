// Package performance provides metrics collection for tracking API call
// durations, configuration operations, and other performance-related data.
//
// Metrics collection is opt-in (disabled by default) for privacy reasons.
// Users must explicitly enable metrics to begin recording.
package performance

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Registry holds all recorded metrics.
type Registry struct {
	Metrics map[string]map[string]*OperationMetrics // operation -> provider -> metrics
	Enabled bool                                    // metrics are opt-in for privacy
	mu      sync.RWMutex
}

// OperationMetrics tracks performance data for a specific operation and provider.
type OperationMetrics struct {
	Count         int           // total number of operations
	TotalDuration time.Duration // cumulative duration
	MinDuration   time.Duration // minimum duration
	MaxDuration   time.Duration // maximum duration
	FailureCount  int           // number of failed operations
}

// OperationStats is a snapshot of metrics for display.
type OperationStats struct {
	Count         int           `json:"count"`
	TotalDuration time.Duration `json:"total_duration_ms"`
	AvgDuration   time.Duration `json:"avg_duration_ms"`
	MinDuration   time.Duration `json:"min_duration_ms"`
	MaxDuration   time.Duration `json:"max_duration_ms"`
	FailureCount  int           `json:"failure_count"`
}

// Stats is a map of operation names to provider stats.
type Stats map[string]map[string]OperationStats

// NewRegistry creates a new metrics registry with metrics disabled by default.
func NewRegistry() *Registry {
	return &Registry{
		Metrics: make(map[string]map[string]*OperationMetrics),
		Enabled: false,
	}
}

// Enable turns on metrics collection.
func (r *Registry) Enable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Enabled = true
}

// Disable turns off metrics collection.
func (r *Registry) Disable() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Enabled = false
}

// RecordOperation records a single operation's performance data.
// If metrics are disabled, this is a no-op.
func (r *Registry) RecordOperation(operation, provider string, duration time.Duration, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.Enabled {
		return
	}

	if r.Metrics[operation] == nil {
		r.Metrics[operation] = make(map[string]*OperationMetrics)
	}

	if r.Metrics[operation][provider] == nil {
		r.Metrics[operation][provider] = &OperationMetrics{
			MinDuration: duration,
			MaxDuration: duration,
		}
	}

	m := r.Metrics[operation][provider]
	m.Count++
	m.TotalDuration += duration

	if duration < m.MinDuration {
		m.MinDuration = duration
	}

	if duration > m.MaxDuration {
		m.MaxDuration = duration
	}

	if !success {
		m.FailureCount++
	}
}

// GetStats returns a snapshot of current metrics.
func (r *Registry) GetStats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(Stats)
	for operation, providers := range r.Metrics {
		stats[operation] = make(map[string]OperationStats)
		for provider, m := range providers {
			stats[operation][provider] = OperationStats{
				Count:         m.Count,
				TotalDuration: m.TotalDuration,
				AvgDuration:   m.TotalDuration / time.Duration(m.Count),
				MinDuration:   m.MinDuration,
				MaxDuration:   m.MaxDuration,
				FailureCount:  m.FailureCount,
			}
		}
	}
	return stats
}

// Reset clears all recorded metrics.
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Metrics = make(map[string]map[string]*OperationMetrics)
}

// ToJSON exports metrics as JSON for external tools.
func (r *Registry) ToJSON() ([]byte, error) {
	stats := r.GetStats()
	return json.MarshalIndent(stats, "", "  ")
}

// RecordDuration is a helper that records the duration of an operation function.
// It wraps the operation with timing and records success/failure.
// The operation's error is returned after recording.
func RecordDuration(registry *Registry, operation, provider string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	registry.RecordOperation(operation, provider, duration, err == nil)

	return err
}

// Summary returns a human-readable summary of all stats.
func (s Stats) Summary() string {
	if len(s) == 0 {
		return ""
	}

	var result string
	for operation, providers := range s {
		totalOps := 0
		totalDuration := time.Duration(0)
		totalFailures := 0

		for _, stats := range providers {
			totalOps += stats.Count
			totalDuration += stats.TotalDuration
			totalFailures += stats.FailureCount
		}

		avgDuration := time.Duration(0)
		if totalOps > 0 {
			avgDuration = totalDuration / time.Duration(totalOps)
		}

		result += fmt.Sprintf("%s: %d ops, avg %v, %d failures\n",
			operation, totalOps, avgDuration.Round(time.Millisecond), totalFailures)
	}

	return result
}
