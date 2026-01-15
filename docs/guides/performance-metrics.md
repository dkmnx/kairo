# Performance Metrics

Kairo includes optional performance metrics collection for tracking API call durations, configuration operations, and other performance-related data.

## Privacy-First Design

Metrics collection is **opt-in and disabled by default**. You must explicitly enable metrics to begin recording. This ensures privacy by default - no performance data is collected unless you choose to enable it.

## Enabling Metrics

There are two ways to enable metrics collection:

### 1. Environment Variable (Recommended)

Set the `KAIRO_METRICS_ENABLED` environment variable:

```bash
export KAIRO_METRICS_ENABLED=true
kairo switch zai
```

### 2. Command Flag

Enable metrics for the current session:

```bash
kairo metrics enable
kairo switch zai
```

## Using Metrics

### View Metrics

Display collected metrics in a human-readable format:

```bash
kairo metrics
```

Example output:

```
Performance Metrics

  api_call
    anthropic
      Calls:      15
      Avg:        125ms
      Min:        80ms
      Max:        250ms
    zai
      Calls:      8
      Avg:        180ms
      Min:        120ms
      Max:        350ms

  config_save
    anthropic
      Calls:      3
      Avg:        45ms
      Min:        30ms
      Max:        60ms

Summary
api_call: 23 ops, avg 145ms, 0 failures
config_save: 3 ops, avg 45ms, 0 failures
```

### Export Metrics

Export metrics to JSON for external analysis:

```bash
kairo metrics -o metrics.json
```

### Reset Metrics

Clear all recorded metrics:

```bash
kairo metrics reset
```

### Disable Metrics

Stop collecting metrics:

```bash
kairo metrics disable
```

## Metrics Tracked

Kairo tracks the following performance data:

- **API call durations** per provider (anthropic, zai, minimax, etc.)
- **Configuration operation times** (save, load, switch)
- **Success/failure rates** for all operations
- **Min/Max/Average** durations for aggregated analysis

## Example Workflow

```bash
# Enable metrics
export KAIRO_METRICS_ENABLED=true

# Use kairo normally - metrics are collected automatically
kairo switch zai
kairo switch anthropic
kairo config zai

# View performance data
kairo metrics

# Export for analysis
kairo metrics -o /tmp/metrics.json

# Reset when done
kairo metrics reset
```

## JSON Export Format

When exporting metrics, the output includes:

```json
{
  "api_call": {
    "anthropic": {
      "count": 15,
      "total_duration_ms": 1875000000,
      "avg_duration_ms": 125000000,
      "min_duration_ms": 80000000,
      "max_duration_ms": 250000000,
      "failure_count": 0
    }
  },
  "config_save": {
    "anthropic": {
      "count": 3,
      "total_duration_ms": 135000000,
      "avg_duration_ms": 45000000,
      "min_duration_ms": 30000000,
      "max_duration_ms": 60000000,
      "failure_count": 0
    }
  }
}
```

## Programmatic Usage

If you're building tools that integrate with Kairo, you can enable metrics programmatically:

```go
import "github.com/dkmnx/kairo/internal/performance"

registry := performance.NewRegistry()
registry.Enable()

// Record an operation
registry.RecordOperation("api_call", "anthropic", 125*time.Millisecond, true)

// Get stats
stats := registry.GetStats()

// Export to JSON
jsonData, err := registry.ToJSON()
```

## Best Practices

1. **Enable only when needed**: Metrics add minimal overhead, but disable when not in use
2. **Export regularly**: Export metrics before resetting to preserve historical data
3. **Analyze trends**: Look for patterns in API call durations to identify performance issues
4. **Check failure rates**: High failure rates may indicate API key issues or network problems

## Limitations

- Metrics are **not persisted** across sessions - each run starts fresh
- Metrics are **in-memory only** - they're lost when the process exits
- No automatic aggregation or historical tracking - export before resetting if needed
