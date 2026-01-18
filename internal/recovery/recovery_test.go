package recovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset"), true},
		{"connection timeout", errors.New("connection timeout"), true},
		{"network unreachable", errors.New("network is unreachable"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"temporary failure", errors.New("temporary failure"), true},
		{"service unavailable", errors.New("service unavailable"), true},
		{"too many requests", errors.New("too many requests"), true},
		{"503 error", errors.New("503 Service Unavailable"), true},
		{"504 error", errors.New("504 Gateway Timeout"), true},
		{"429 error", errors.New("429 Too Many Requests"), true},
		{"gateway timeout", errors.New("gateway timeout"), true},
		{"bad gateway", errors.New("bad gateway"), true},
		{"permanent error", errors.New("permission denied"), false},
		{"validation error", errors.New("invalid input"), false},
		{"not found", errors.New("not found"), false},
		{"HTTP 400", errors.New("400 Bad Request"), false},
		{"HTTP 401", errors.New("401 Unauthorized"), false},
		{"HTTP 403", errors.New("403 Forbidden"), false},
		{"HTTP 404", errors.New("404 Not Found"), false},
		{"HTTP 500", errors.New("500 Internal Server Error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransient(tt.err)
			if result != tt.expected {
				t.Errorf("IsTransient(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsTransient_NetError(t *testing.T) {
	// Test with actual net.Error implementation
	t.Run("net.OpError - connection refused", func(t *testing.T) {
		err := &net.OpError{
			Op:   "dial",
			Net:  "tcp",
			Err:  fmt.Errorf("connection refused"),
			Addr: nil,
		}
		if !IsTransient(err) {
			t.Error("IsTransient should detect net.OpError (connection refused) as transient")
		}
	})

	t.Run("net.OpError - timeout", func(t *testing.T) {
		err := &net.OpError{
			Op:   "read",
			Net:  "tcp",
			Err:  fmt.Errorf("i/o timeout"),
			Addr: nil,
		}
		if !IsTransient(err) {
			t.Error("IsTransient should detect net.OpError (timeout) as transient")
		}
	})

	t.Run("net.OpError - permanent error", func(t *testing.T) {
		err := &net.OpError{
			Op:   "connect",
			Net:  "tcp",
			Err:  fmt.Errorf("permission denied"),
			Addr: nil,
		}
		if IsTransient(err) {
			t.Error("IsTransient should not detect net.OpError (permission denied) as transient")
		}
	})
}

func TestNewRetryConfig(t *testing.T) {
	cfg := NewRetryConfig()

	if cfg.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %v, want %v", cfg.MaxRetries, DefaultMaxRetries)
	}
	if cfg.BaseDelay != DefaultBaseDelay {
		t.Errorf("BaseDelay = %v, want %v", cfg.BaseDelay, DefaultBaseDelay)
	}
	if cfg.MaxDelay != DefaultMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", cfg.MaxDelay, DefaultMaxDelay)
	}
	if cfg.JitterFactor != DefaultJitterFactor {
		t.Errorf("JitterFactor = %v, want %v", cfg.JitterFactor, DefaultJitterFactor)
	}
}

func TestRetryOptions(t *testing.T) {
	t.Run("WithMaxRetries", func(t *testing.T) {
		cfg := NewRetryConfig(WithMaxRetries(5))
		if cfg.MaxRetries != 5 {
			t.Errorf("MaxRetries = %v, want 5", cfg.MaxRetries)
		}
	})

	t.Run("WithBaseDelay", func(t *testing.T) {
		delay := 200 * time.Millisecond
		cfg := NewRetryConfig(WithBaseDelay(delay))
		if cfg.BaseDelay != delay {
			t.Errorf("BaseDelay = %v, want %v", cfg.BaseDelay, delay)
		}
	})

	t.Run("WithMaxDelay", func(t *testing.T) {
		delay := 10 * time.Second
		cfg := NewRetryConfig(WithMaxDelay(delay))
		if cfg.MaxDelay != delay {
			t.Errorf("MaxDelay = %v, want %v", cfg.MaxDelay, delay)
		}
	})

	t.Run("WithJitterFactor", func(t *testing.T) {
		cfg := NewRetryConfig(WithJitterFactor(0.5))
		if cfg.JitterFactor != 0.5 {
			t.Errorf("JitterFactor = %v, want 0.5", cfg.JitterFactor)
		}
	})

	t.Run("WithRetryableFunc", func(t *testing.T) {
		customRetryable := func(err error) bool {
			return err != nil && err.Error() == "custom retry"
		}
		cfg := NewRetryConfig(WithRetryableFunc(customRetryable))
		if cfg.Retryable == nil {
			t.Error("Retryable should be set")
		}
	})
}

func TestRetry_Success(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(3))
	callCount := 0

	result, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		return "success", nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("Retry() result = %v, want 'success'", result)
	}
	if callCount != 1 {
		t.Errorf("Retry() called %v times, want 1", callCount)
	}
}

func TestRetry_TransientError(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(3), WithBaseDelay(10*time.Millisecond))
	callCount := 0

	result, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("connection refused")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("Retry() result = %v, want 'success'", result)
	}
	if callCount != 3 {
		t.Errorf("Retry() called %v times, want 3", callCount)
	}
}

func TestRetry_PermanentError(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(3), WithBaseDelay(10*time.Millisecond))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		return "", errors.New("permission denied")
	})

	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	if err.Error() != "permission denied" {
		t.Errorf("Retry() error = %v, want permission denied", err)
	}
	if callCount != 1 {
		t.Errorf("Retry() called %v times, want 1 (should not retry permanent error)", callCount)
	}
}

func TestRetry_ExhaustedRetries(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(2), WithBaseDelay(10*time.Millisecond))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		return "", errors.New("connection timeout")
	})

	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	if callCount != 3 { // 1 initial + 2 retries = 3 attempts
		t.Errorf("Retry() called %v times, want 3", callCount)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := NewRetryConfig(WithMaxRetries(5), WithBaseDelay(100*time.Millisecond))

	// Cancel after first attempt
	callCount := 0
	var firstAttemptDone bool
	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		if !firstAttemptDone {
			firstAttemptDone = true
			cancel()
		}
		// Sleep to allow context cancellation to propagate before retry loop checks it
		// This ensures deterministic test behavior for timing-sensitive operations
		time.Sleep(200 * time.Millisecond)
		return "", errors.New("connection refused")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Retry() error = %v, want context.Canceled", err)
	}
	// Should have made at least one attempt
	if callCount < 1 {
		t.Errorf("Retry() called %v times, want >= 1", callCount)
	}
}

func TestRetryWithTimeout(t *testing.T) {
	// Use longer timeout with immediate failures to test timeout behavior.
	// Zero jitter and small base delay make timing predictable.
	// Exponential backoff: 0ms, 20ms, 40ms, 80ms, 160ms...
	ctx, cancel := WithTimeout(200 * time.Millisecond)
	defer cancel()
	cfg := NewRetryConfig(WithMaxRetries(100), WithBaseDelay(10*time.Millisecond), WithJitterFactor(0))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		// Immediate failure to test retry loop behavior without timing complexity
		return "", errors.New("connection refused")
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Retry() error = %v, want context.DeadlineExceeded", err)
	}
	// Timeline with exponential backoff (baseDelay=10ms):
	// Attempt 1: 0ms, Attempt 2: 0ms+20ms=20ms, Attempt 3: 20ms+40ms=60ms,
	// Attempt 4: 60ms+80ms=140ms, Attempt 5: 140ms+160ms=300ms (timeout)
	// So we get exactly 5 attempts with exponential backoff
	if callCount != 5 {
		t.Errorf("Retry() called %v times, want 5 (timeout with exponential backoff)", callCount)
	}
}

func TestRetry_ContextCancelledBeforeStart(t *testing.T) {
	// Test fast path: context cancelled before first attempt
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := NewRetryConfig(WithMaxRetries(5), WithBaseDelay(100*time.Millisecond))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		return "result", nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Retry() error = %v, want context.Canceled", err)
	}
	// Should not have made any attempts (fast path)
	if callCount != 0 {
		t.Errorf("Retry() called %v times, want 0 (fast path)", callCount)
	}
}

func TestRetryWithoutResult(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(3))
	callCount := 0

	err := RetryWithoutResult(ctx, cfg, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection refused")
		}
		return nil
	})

	if err != nil {
		t.Errorf("RetryWithoutResult() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("RetryWithoutResult() called %v times, want 3", callCount)
	}
}

func TestCalculateDelay(t *testing.T) {
	cfg := RetryConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		JitterFactor: 0.0, // No jitter for predictability
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"attempt 0", 0, 0},
		{"attempt 1", 1, 200 * time.Millisecond},  // 100 * 2^1
		{"attempt 2", 2, 400 * time.Millisecond},  // 100 * 2^2
		{"attempt 3", 3, 800 * time.Millisecond},  // 100 * 2^3
		{"attempt 4", 4, 1600 * time.Millisecond}, // 100 * 2^4
		{"attempt 5", 5, 3200 * time.Millisecond}, // 100 * 2^5
		{"attempt 6", 6, 5 * time.Second},         // 100 * 2^6 = 6400ms, capped at 5s
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateDelay(tt.attempt, cfg)
			if delay != tt.expected {
				t.Errorf("calculateDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestCalculateDelay_MaxDelay(t *testing.T) {
	cfg := RetryConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond, // Low max for testing
		JitterFactor: 0.0,
	}

	// Should cap at max delay
	delay := calculateDelay(10, cfg)
	if delay != 500*time.Millisecond {
		t.Errorf("calculateDelay(10) = %v, want 500ms (capped)", delay)
	}
}

func TestCalculateDelay_MinDelayBound(t *testing.T) {
	// Test that jitter cannot reduce delay below 1ms minimum bound
	cfg := RetryConfig{
		BaseDelay:    1 * time.Millisecond, // Very small base delay
		MaxDelay:     5 * time.Second,
		JitterFactor: 0.5, // 50% jitter (could produce 0ms)
	}

	// Run many iterations to check minimum bound
	minObserved := time.Hour
	for i := 0; i < 1000; i++ {
		delay := calculateDelay(1, cfg)
		if delay < minObserved {
			minObserved = delay
		}
	}

	// Minimum delay should be at least 1ms
	const minDelay = 1 * time.Millisecond
	if minObserved < minDelay {
		t.Errorf("Minimum delay %v below expected %v", minObserved, minDelay)
	}

	// Verify jitter is still working (should have variance)
	if minObserved == time.Hour {
		t.Error("calculateDelay() produced no values")
	}
}

func TestCalculateDelay_WithJitter(t *testing.T) {
	cfg := RetryConfig{
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		JitterFactor: 0.5, // 50% jitter
	}

	// Test that jitter adds randomness by running multiple iterations
	// Using 100 iterations to check for statistical variance
	// This eliminates flakiness from just checking 3 values
	uniqueDelays := make(map[time.Duration]bool)
	minObserved := time.Hour // Start with large value
	maxObserved := time.Duration(0)

	for i := 0; i < 100; i++ {
		delay := calculateDelay(1, cfg)
		uniqueDelays[delay] = true

		if delay < minObserved {
			minObserved = delay
		}
		if delay > maxObserved {
			maxObserved = delay
		}
	}

	// Should have at least some variance (expect >10 unique values with 50% jitter)
	if len(uniqueDelays) < 10 {
		t.Errorf("calculateDelay() should produce variance, got %v unique values", len(uniqueDelays))
	}

	// All delays should be in reasonable range (200ms ± 50% jitter)
	minExpected := 150 * time.Millisecond // 200ms - 50%
	maxExpected := 250 * time.Millisecond // 200ms + 50%

	if minObserved < minExpected {
		t.Errorf("Minimum observed delay %v below expected %v", minObserved, minExpected)
	}
	if maxObserved > maxExpected {
		t.Errorf("Maximum observed delay %v above expected %v", maxObserved, maxExpected)
	}
}

func TestIsTransient_EdgeCases(t *testing.T) {
	t.Run("wrapped error is transient", func(t *testing.T) {
		baseErr := errors.New("connection refused")
		wrappedErr := fmt.Errorf("wrapper: %w", baseErr)
		result := IsTransient(wrappedErr)
		if !result {
			t.Errorf("IsTransient(wrapped error) = false, want true")
		}
	})

	t.Run("wrapped error is permanent", func(t *testing.T) {
		baseErr := errors.New("permission denied")
		wrappedErr := fmt.Errorf("wrapper: %w", baseErr)
		result := IsTransient(wrappedErr)
		if result {
			t.Errorf("IsTransient(wrapped permanent error) = true, want false")
		}
	})
}

func TestRetry_NoRetries(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(0))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		return "", errors.New("connection refused")
	})

	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	if callCount != 1 {
		t.Errorf("Retry() called %v times, want 1 (no retries)", callCount)
	}
}

func TestRetry_ZeroBaseDelay(t *testing.T) {
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(3), WithBaseDelay(0))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("connection refused")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("Retry() called %v times, want 3", callCount)
	}
}

func TestRecoverFromPanic_NilPanic(t *testing.T) {
	err := RecoverFromPanic(func() {
		panic(nil)
	})

	if err == nil {
		t.Error("RecoverFromPanic() error = nil, want error for nil panic")
	}
	// Verify error message contains nil panic information
	// Go's panic with nil produces "panic called with nil argument"
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("RecoverFromPanic() error = %v, should contain panic information", err)
	}
}

func TestRecoverFromPanic_InterfaceNilPanic(t *testing.T) {
	// Test panic with typed nil: interface{}(nil) vs panic(nil)
	// These behave differently in Go and should both be handled
	err := RecoverFromPanic(func() {
		var nilInterface interface{} = nil
		panic(nilInterface)
	})

	if err == nil {
		t.Error("RecoverFromPanic() error = nil, want error for interface nil panic")
	}
	// Verify error message contains panic information
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("RecoverFromPanic() error = %v, should contain panic information", err)
	}
}

func TestRecoverFromPanic(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		err := RecoverFromPanic(func() {})
		if err != nil {
			t.Errorf("RecoverFromPanic() error = %v, want nil", err)
		}
	})

	t.Run("with panic", func(t *testing.T) {
		err := RecoverFromPanic(func() {
			panic("test panic")
		})
		if err == nil {
			t.Error("RecoverFromPanic() error = nil, want error")
		}
		// Check that panic value is preserved in error message
		if err.Error() == "panic recovered" {
			t.Error("RecoverFromPanic() lost panic value in error message")
		}
	})

	t.Run("with panic value", func(t *testing.T) {
		err := RecoverFromPanic(func() {
			panic(errors.New("panic with error"))
		})
		if err == nil {
			t.Error("RecoverFromPanic() error = nil, want error")
		}
		// Check that error message contains original panic message
		if !strings.Contains(err.Error(), "panic with error") {
			t.Errorf("RecoverFromPanic() error = %v, should contain panic value", err)
		}
	})
}

func TestRecoverFromPanicSanitized(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		err := RecoverFromPanicSanitized(func() {})
		if err != nil {
			t.Errorf("RecoverFromPanicSanitized() error = %v, want nil", err)
		}
	})

	t.Run("with panic value", func(t *testing.T) {
		// Panic with sensitive value (password)
		err := RecoverFromPanicSanitized(func() {
			panic("secret-password-123")
		})
		if err == nil {
			t.Error("RecoverFromPanicSanitized() error = nil, want error")
		}
		// Check that panic value is NOT in error message
		if strings.Contains(err.Error(), "secret-password-123") {
			t.Error("RecoverFromPanicSanitized() leaked panic value in error message")
		}
		// Check that redaction message is present
		if !strings.Contains(err.Error(), "redacted") {
			t.Error("RecoverFromPanicSanitized() missing redaction message")
		}
	})

	t.Run("with sensitive data", func(t *testing.T) {
		// Panic with token (highly sensitive)
		sensitiveToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
		err := RecoverFromPanicSanitized(func() {
			panic(sensitiveToken)
		})
		if err == nil {
			t.Error("RecoverFromPanicSanitized() error = nil, want error")
		}
		// Check that token is NOT in error message
		if strings.Contains(err.Error(), sensitiveToken) {
			t.Error("RecoverFromPanicSanitized() leaked sensitive token in error message")
		}
	})
}

func TestRecoverFromPanicWithContext(t *testing.T) {
	t.Run("no panic, active context", func(t *testing.T) {
		ctx := context.Background()
		err := RecoverFromPanicWithContext(ctx, func() {})
		if err != nil {
			t.Errorf("RecoverFromPanicWithContext() error = %v, want nil", err)
		}
	})

	t.Run("with panic, active context", func(t *testing.T) {
		ctx := context.Background()
		err := RecoverFromPanicWithContext(ctx, func() {
			panic("test panic")
		})
		if err == nil {
			t.Error("RecoverFromPanicWithContext() error = nil, want error")
		}
		// Check that panic value is preserved
		if !strings.Contains(err.Error(), "test panic") {
			t.Errorf("RecoverFromPanicWithContext() error = %v, should contain panic value", err)
		}
	})

	t.Run("no panic, cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := RecoverFromPanicWithContext(ctx, func() {})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("RecoverFromPanicWithContext() error = %v, want context.Canceled", err)
		}
	})

	t.Run("with panic, cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := RecoverFromPanicWithContext(ctx, func() {
			panic("test panic")
		})
		if err == nil {
			t.Error("RecoverFromPanicWithContext() error = nil, want error")
		}
		// Should have both panic and context cancellation
		if !strings.Contains(err.Error(), "panic") {
			t.Error("RecoverFromPanicWithContext() should contain panic information")
		}
		// Check that context cancellation is mentioned
		errLower := strings.ToLower(err.Error())
		t.Logf("Error message: %s", err.Error())
		if !strings.Contains(errLower, "context") && !strings.Contains(errLower, "cancel") {
			t.Errorf("RecoverFromPanicWithContext() should mention context cancellation, got: %s", err.Error())
		}
		// Verify no double-wrapping of KairoError (nested KairoError within KairoError)
		var kairoErr *kairoerrors.KairoError
		if errors.As(err, &kairoErr) {
			// Check if the unwrapped cause is also a KairoError (double-wrapping)
			if kairoErr.Unwrap() != nil {
				var nestedKairoErr *kairoerrors.KairoError
				if errors.As(kairoErr.Unwrap(), &nestedKairoErr) {
					t.Errorf("RecoverFromPanicWithContext() should not double-wrap KairoError, got nested KairoError: %v -> %v", kairoErr.Type, nestedKairoErr.Type)
				}
			}
		}
	})
}

func TestRecoverFromPanicSanitizedWithContext(t *testing.T) {
	t.Run("no panic, active context", func(t *testing.T) {
		ctx := context.Background()
		err := RecoverFromPanicSanitizedWithContext(ctx, func() {})
		if err != nil {
			t.Errorf("RecoverFromPanicSanitizedWithContext() error = %v, want nil", err)
		}
	})

	t.Run("with panic, active context", func(t *testing.T) {
		ctx := context.Background()
		err := RecoverFromPanicSanitizedWithContext(ctx, func() {
			panic("secret-password")
		})
		if err == nil {
			t.Error("RecoverFromPanicSanitizedWithContext() error = nil, want error")
		}
		// Check that panic value is NOT in error message
		if strings.Contains(err.Error(), "secret-password") {
			t.Error("RecoverFromPanicSanitizedWithContext() leaked panic value in error message")
		}
	})

	t.Run("no panic, cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := RecoverFromPanicSanitizedWithContext(ctx, func() {})
		if !errors.Is(err, context.Canceled) {
			t.Errorf("RecoverFromPanicSanitizedWithContext() error = %v, want context.Canceled", err)
		}
	})

	t.Run("with panic, cancelled context (sanitized)", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := RecoverFromPanicSanitizedWithContext(ctx, func() {
			panic("api-key-123456")
		})
		if err == nil {
			t.Error("RecoverFromPanicSanitizedWithContext() error = nil, want error")
		}
		// Should have both panic and context cancellation
		if !strings.Contains(err.Error(), "redacted") {
			t.Error("RecoverFromPanicSanitizedWithContext() should have redaction message")
		}
		// Check that context cancellation is mentioned (not necessarily using errors.Is)
		if !strings.Contains(strings.ToLower(err.Error()), "context") && !strings.Contains(strings.ToLower(err.Error()), "cancel") {
			t.Error("RecoverFromPanicSanitizedWithContext() should mention context cancellation")
		}
		// Should NOT leak API key
		if strings.Contains(err.Error(), "api-key-123456") {
			t.Error("RecoverFromPanicSanitizedWithContext() leaked sensitive data")
		}
	})
}

func TestCustomRetryableFunc(t *testing.T) {
	ctx := context.Background()
	customRetryable := func(err error) bool {
		return err != nil && err.Error() == "custom error"
	}
	cfg := NewRetryConfig(WithMaxRetries(3), WithRetryableFunc(customRetryable))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("custom error")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if callCount != 3 {
		t.Errorf("Retry() called %v times, want 3", callCount)
	}
}

func TestCustomRetryableFunc_Permanent(t *testing.T) {
	ctx := context.Background()
	customRetryable := func(err error) bool {
		return err != nil && err.Error() == "custom error"
	}
	cfg := NewRetryConfig(WithMaxRetries(3), WithRetryableFunc(customRetryable))
	callCount := 0

	_, err := Retry(ctx, cfg, func() (string, error) {
		callCount++
		return "", errors.New("different error")
	})

	if err == nil {
		t.Error("Retry() error = nil, want error")
	}
	// Should not retry because custom function returns false
	if callCount != 1 {
		t.Errorf("Retry() called %v times, want 1 (custom retryable returns false)", callCount)
	}
}

func TestRetry_ConcurrentSafety(t *testing.T) {
	// This test validates thread-safety of concurrent retry operations.
	// Run with race detector to ensure no data races: go test -race
	ctx := context.Background()
	cfg := NewRetryConfig(WithMaxRetries(3), WithBaseDelay(1*time.Millisecond))
	const goroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Each goroutine runs independent retry operations
			_, _ = Retry(ctx, cfg, func() (string, error) {
				return "", errors.New("connection refused")
			})
		}(i)
	}
	wg.Wait()
	// Test passes if no data race detected by -race flag
}
