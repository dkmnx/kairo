// Package recovery provides utilities for error recovery and retry mechanisms.
//
// Thread Safety:
//   - IsTransient(): Thread-safe (no shared state)
//   - Retry(): Thread-safe (context-safe, no shared state)
//   - RetryWithoutResult(): Thread-safe (context-safe, no shared state)
//   - RecoverFromPanic(): Thread-safe (no shared state)
//   - calculateDelay(): Thread-safe (uses mutex-protected local RNG)
//
// Security:
//   - Uses math/rand for jitter (non-cryptographic, acceptable for timing)
//   - No secrets in transient error patterns
//   - Panic values preserved in error messages (use RecoverFromPanicSanitized for sensitive data)
//
// Performance:
//   - IsTransient() uses bytes package for zero-allocation case-insensitive matching
//   - Transient patterns pre-allocated as byte slices at package level
//   - Local RNG with mutex avoids contention on global rand
//   - Jitter has minimum delay bound of 1ms to prevent immediate retries
package recovery

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	mathrand "math/rand"
	"sync"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// Default retry configuration
const (
	DefaultMaxRetries   = 3
	DefaultBaseDelay    = 100 * time.Millisecond
	DefaultMaxDelay     = 5 * time.Second
	DefaultJitterFactor = 0.1
)

// MaxSafeBackoffFactor prevents integer overflow in time.Duration calculations.
// With float64 to int64 conversion, factors > 2^60 could overflow.
// Since time.Duration is int64 nanoseconds, we cap at a safe limit.
const MaxSafeBackoffFactor = 1.0e18

// Thread-safe random number generator for jitter calculations.
// Uses a local RNG with mutex to avoid race conditions and contention.
// Seeded with crypto/rand for unpredictability to avoid race condition risk
// from time.Now().UnixNano() being predictable during concurrent init or process restarts.
var (
	rngMu sync.Mutex
	rng   *mathrand.Rand
)

func init() {
	// Use crypto/rand to get an unpredictable seed
	var seed int64
	if err := binary.Read(cryptorand.Reader, binary.BigEndian, &seed); err != nil {
		// Fallback to time if crypto/rand fails (shouldn't happen)
		seed = time.Now().UnixNano()
	}
	rng = mathrand.New(mathrand.NewSource(seed))
}

// Helper functions for zero-allocation case-insensitive substring matching.

// toLowerByte converts an uppercase ASCII byte to lowercase without allocation.
//
// This function performs a simple ASCII-only lowercase conversion for single
// bytes. It only converts 'A'-'Z' to 'a'-'z', leaving
// other characters unchanged. This is used by equalIgnoreCaseBytes
// for case-insensitive comparison without allocating strings.
//
// Parameters:
//   - c: ASCII byte to convert
//
// Returns:
//   - byte: Lowercase byte if c is uppercase 'A'-'Z', otherwise returns c unchanged
//
// Error conditions: None
//
// Thread Safety: Thread-safe (pure function, no shared state)
// Performance Notes: Zero-allocation, used for fast case-insensitive matching
func toLowerByte(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// containsIgnoreCaseBytes checks if pattern exists in data using case-insensitive matching.
//
// This function performs a case-insensitive substring search on byte slices.
// It returns true immediately if pattern is empty (empty pattern matches
// everything). Uses toLowerByte for ASCII-only conversion without string
// allocations. Optimized for use in hot paths like error message
// matching.
//
// Parameters:
//   - data: Byte slice to search within
//   - pattern: Byte slice pattern to search for
//
// Returns:
//   - bool: true if pattern is found in data (case-insensitive), false otherwise
//
// Error conditions: None
//
// Thread Safety: Thread-safe (pure function, no shared state)
// Performance Notes: Zero-allocation matching using byte slices, O(n*m) where n=len(data), m=len(pattern)
func containsIgnoreCaseBytes(data []byte, pattern []byte) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(data) < len(pattern) {
		return false
	}
	for i := 0; i <= len(data)-len(pattern); i++ {
		if equalIgnoreCaseBytes(data[i:i+len(pattern)], pattern) {
			return true
		}
	}
	return false
}

// equalIgnoreCaseBytes compares two byte slices for case-insensitive equality.
//
// This function checks if two byte slices are equal ignoring ASCII case
// differences. It uses toLowerByte for zero-allocation ASCII-only
// case conversion. This is used by containsIgnoreCaseBytes for substring
// matching.
//
// Parameters:
//   - a: First byte slice to compare
//   - b: Second byte slice to compare
//
// Returns:
//   - bool: true if slices have equal length and byte values (ignoring ASCII case)
//
// Error conditions: None
//
// Thread Safety: Thread-safe (pure function, no shared state)
// Performance Notes: Zero-allocation, O(n) where n is slice length
func equalIgnoreCaseBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if toLowerByte(a[i]) != toLowerByte(b[i]) {
			return false
		}
	}
	return true
}

// Pre-allocated transient error patterns as byte slices for zero-allocation matching.
// Using byte slices avoids string allocations during error matching.
// Note: HTTP status codes are intentionally more specific to avoid false positives
// (e.g., "500" could match non-transient IDs, so we use full error messages).
// Patterns are alphabetized for maintainability.
var transientPatternsBytes = [][]byte{
	// HTTP status codes (specific full messages to avoid false positives)
	[]byte("429 too many requests"),
	[]byte("500 internal server error"),
	[]byte("503 service unavailable"),
	[]byte("504 gateway timeout"),
	// General error patterns (alphabetized, lowercase)
	[]byte("bad gateway"),
	[]byte("connection refused"),
	[]byte("connection reset"),
	[]byte("connection timeout"),
	[]byte("econnrefused"),
	[]byte("ehostunreach"),
	[]byte("enetunreach"),
	[]byte("etimedout"),
	[]byte("gateway timeout"),
	[]byte("i/o timeout"),
	[]byte("network is unreachable"),
	[]byte("no such host"),
	[]byte("server misconfigured"),
	[]byte("service unavailable"),
	[]byte("temporary"),
	[]byte("temporary failure"),
	[]byte("timeout"),
	[]byte("too many requests"),
}

// IsTransient checks if an error is likely transient and retryable.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	// Check for common transient error patterns (case-insensitive)
	// Using byte slice for zero-allocation matching
	errMsg := []byte(err.Error())
	for _, pattern := range transientPatternsBytes {
		if containsIgnoreCaseBytes(errMsg, pattern) {
			return true
		}
	}

	// Check wrapped errors
	var netErr interface {
		Timeout() bool
		Temporary() bool
	}
	if errors.As(err, &netErr) {
		return netErr.Temporary() || netErr.Timeout()
	}

	return false
}

// RetryableChecker defines an interface for determining if an error should trigger a retry.
// This enables dependency injection for custom retry logic in tests and adapters.
type RetryableChecker interface {
	IsRetryable(err error) bool
}

// funcRetryableChecker adapts a function to the RetryableChecker interface.
type funcRetryableChecker struct {
	check func(error) bool
}

func (f *funcRetryableChecker) IsRetryable(err error) bool {
	return f.check(err)
}

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	MaxRetries   int              // Maximum number of retry attempts (0 = no retries)
	BaseDelay    time.Duration    // Initial delay between retries
	MaxDelay     time.Duration    // Maximum delay between retries
	JitterFactor float64          // Random jitter factor (0.0 = no jitter)
	Retryable    RetryableChecker // Custom checker for retryable errors
}

// RetryOption applies a configuration option.
type RetryOption func(*RetryConfig)

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) RetryOption {
	return func(c *RetryConfig) {
		if n >= 0 {
			c.MaxRetries = n
		}
	}
}

// WithBaseDelay sets the base delay between retries.
func WithBaseDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) {
		if d > 0 {
			c.BaseDelay = d
		}
	}
}

// WithMaxDelay sets the maximum delay between retries.
func WithMaxDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) {
		if d > 0 {
			c.MaxDelay = d
		}
	}
}

// WithJitterFactor sets the jitter factor for randomizing delays.
func WithJitterFactor(f float64) RetryOption {
	return func(c *RetryConfig) {
		if f >= 0 && f <= 1 {
			c.JitterFactor = f
		}
	}
}

// WithRetryableFunc sets a custom function to determine if an error is retryable.
func WithRetryableFunc(f func(error) bool) RetryOption {
	return func(c *RetryConfig) {
		if f != nil {
			c.Retryable = &funcRetryableChecker{check: f}
		}
	}
}

// NewRetryConfig creates a RetryConfig with default values.
func NewRetryConfig(opts ...RetryOption) RetryConfig {
	cfg := RetryConfig{
		MaxRetries:   DefaultMaxRetries,
		BaseDelay:    DefaultBaseDelay,
		MaxDelay:     DefaultMaxDelay,
		JitterFactor: DefaultJitterFactor,
		// Wrap the global IsTransient function in an interface for consistency
		Retryable: &funcRetryableChecker{check: IsTransient},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Retry executes fn with retry logic according to cfg.
// Returns the result of the last attempt, or the last error if all retries fail.
func Retry[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error) {
	var result T

	// Use retryLoop for common retry logic, capturing result from fn
	_, err := retryLoop(ctx, cfg, func() error {
		var loopErr error
		result, loopErr = fn()
		return loopErr
	})

	return result, err
}

// retryLoop handles the common retry logic for both Retry and RetryWithoutResult.
// It returns the number of attempts made and whether the operation succeeded.
func retryLoop(ctx context.Context, cfg RetryConfig, fn func() error) (int, error) {
	var lastErr error
	attemptCount := 0

	// Fast path: check context cancellation before first attempt
	// This allows callers to fail fast if context is already cancelled
	select {
	case <-ctx.Done():
		return attemptCount, ctx.Err()
	default:
	}

	for attempt := 0; ; attempt++ {
		attemptCount = attempt + 1
		lastErr = fn()
		if lastErr == nil {
			return attemptCount, nil
		}

		// Check if we should retry
		if attempt >= cfg.MaxRetries {
			break
		}

		// Use custom retryable if provided, otherwise use default IsTransient
		shouldRetry := false
		if cfg.Retryable != nil {
			shouldRetry = cfg.Retryable.IsRetryable(lastErr)
		} else {
			shouldRetry = IsTransient(lastErr)
		}

		if !shouldRetry {
			break
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return attemptCount, ctx.Err()
		default:
		}

		// Calculate delay with exponential backoff and jitter
		delay := calculateDelay(attempt, cfg)
		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop() // Prevent goroutine leak
			select {
			case <-ctx.Done():
				// If context is cancelled, stop timer immediately
				// Note: Stop() is safe to call even if timer already fired
				return attemptCount, ctx.Err()
			case <-timer.C:
			}
		}
	}

	return attemptCount, lastErr
}

// RetryFunc is a function that can be retried.
type RetryFunc func() error

// RetryWithoutResult executes fn (that returns only error) with retry logic.
func RetryWithoutResult(ctx context.Context, cfg RetryConfig, fn RetryFunc) error {
	_, err := retryLoop(ctx, cfg, fn)
	return err
}

// calculateDelay computes the delay for a given attempt using exponential backoff with jitter.
func calculateDelay(attempt int, cfg RetryConfig) time.Duration {
	if attempt == 0 || cfg.BaseDelay == 0 {
		return 0
	}

	// Exponential backoff: base * 2^attempt
	// Note: Using loop instead of 1 << attempt to avoid float64 type issues
	backoffFactor := 1.0
	for i := 0; i < int(attempt); i++ {
		backoffFactor *= 2
		// Prevent integer overflow in time.Duration calculations
		// MaxSafeBackoffFactor is 10^18, safe for int64 nanoseconds
		if backoffFactor > MaxSafeBackoffFactor {
			backoffFactor = MaxSafeBackoffFactor
			break
		}
	}
	delay := time.Duration(float64(cfg.BaseDelay) * backoffFactor)

	// Cap at max delay
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}

	// Add jitter
	// Note: Using math/rand instead of crypto/rand because jitter values don't need cryptographic security
	// math/rand is significantly faster and appropriate for random timing variations
	// Local RNG with mutex ensures thread-safety and avoids race conditions
	// Note: Jitter can reduce delay below base backoff (intentional - distributes retries evenly around base delay)
	if cfg.JitterFactor > 0 {
		jitterRangeMs := float64(delay) * cfg.JitterFactor
		jitterRange := time.Duration(jitterRangeMs)

		// Skip jitter if range is zero or negative (prevents Int63n(0) panic)
		// This can happen when delay is very small (e.g., < 1ms)
		if jitterRange.Milliseconds() > 0 {
			// Use mutex-protected RNG for thread-safety
			rngMu.Lock()
			jitterMs := rng.Int63n(int64(jitterRange.Milliseconds()))
			rngMu.Unlock()
			// Subtract half range to center jitter around base delay (can result in negative/short delays)
			jitter := time.Duration(jitterMs)*time.Millisecond - jitterRange/2
			delay = delay + jitter
		}

		// Ensure minimum delay bound to prevent immediate retries
		// Minimum delay is 1ms to avoid zero/negative delays after jitter
		const minDelay = 1 * time.Millisecond
		if delay < minDelay {
			delay = minDelay
		}
	}

	return delay
}

// WithTimeout creates a context with timeout for retry operations.
func WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// RecoverFromPanic recovers from a panic and returns an error.
// The panic value is preserved in the error message for debugging purposes.
// For sensitive environments, use RecoverFromPanicSanitized to avoid logging panic values.
func RecoverFromPanic(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Preserve actual panic value in error message
			err = kairoerrors.RuntimeErr(
				"panic recovered",
				fmt.Errorf("panic: %v", r),
			)
		}
	}()
	fn()
	return nil
}

// RecoverFromPanicSanitized recovers from a panic and returns an error.
// The panic value is NOT included in the error message for security/privacy.
// Use this in production environments where panic values might contain sensitive data
// (e.g., passwords, tokens, personal information).
func RecoverFromPanicSanitized(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Don't include panic value in error message for security
			err = kairoerrors.RuntimeErr(
				"panic recovered (value redacted for security)",
				nil,
			)
		}
	}()
	fn()
	return nil
}

// RecoverFromPanicWithContext recovers from a panic and returns an error, respecting context cancellation.
// The panic value is preserved in error message for debugging purposes.
// If context is cancelled during panic recovery, context error is returned.
func RecoverFromPanicWithContext(ctx context.Context, fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := kairoerrors.RuntimeErr(
				"panic recovered",
				fmt.Errorf("panic: %v", r),
			)
			// Check if context is also cancelled
			if ctx.Err() != nil {
				// Both panic and context error - join both
				err = errors.Join(panicErr, ctx.Err())
			} else {
				// Only panic - no context cancellation
				err = panicErr
			}
		}
	}()

	fn()

	// Check context after function completes (no panic case)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// RecoverFromPanicSanitizedWithContext recovers from a panic and returns an error, respecting context cancellation.
// The panic value is NOT included in the error message for security/privacy.
// Use this in production environments where panic values might contain sensitive data.
// If context is cancelled during panic recovery, context error is returned.
func RecoverFromPanicSanitizedWithContext(ctx context.Context, fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr := kairoerrors.RuntimeErr(
				"panic recovered (value redacted for security)",
				nil,
			)
			// Check if context is also cancelled
			if ctx.Err() != nil {
				// Both panic and context error - join both
				err = errors.Join(panicErr, ctx.Err())
			} else {
				// Only panic - no context cancellation
				err = panicErr
			}
		}
	}()

	fn()

	// Check context after function completes (no panic case)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
