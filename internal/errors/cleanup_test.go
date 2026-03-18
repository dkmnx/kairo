package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestCleanupOnError(t *testing.T) {
	t.Run("returns nil when err is nil", func(t *testing.T) {
		cleanupCalled := false
		cleanup := func() error {
			cleanupCalled = true
			return nil
		}

		result := CleanupOnError(nil, cleanup)
		if result != nil {
			t.Error("CleanupOnError should return nil when err is nil")
		}
		if cleanupCalled {
			t.Error("cleanup should not be called when err is nil")
		}
	})

	t.Run("runs cleanup and returns original error", func(t *testing.T) {
		originalErr := errors.New("original error")
		cleanupCalled := false
		cleanup := func() error {
			cleanupCalled = true
			return nil
		}

		result := CleanupOnError(originalErr, cleanup)
		if result != originalErr {
			t.Error("CleanupOnError should return the original error")
		}
		if !cleanupCalled {
			t.Error("cleanup should be called when err is not nil")
		}
	})

	t.Run("runs multiple cleanups in order", func(t *testing.T) {
		originalErr := errors.New("original error")
		var order []int
		cleanup1 := func() error {
			order = append(order, 1)
			return nil
		}
		cleanup2 := func() error {
			order = append(order, 2)
			return nil
		}
		cleanup3 := func() error {
			order = append(order, 3)
			return nil
		}

		result := CleanupOnError(originalErr, cleanup1, cleanup2, cleanup3)
		if result != originalErr {
			t.Error("CleanupOnError should return the original error")
		}
		if len(order) != 3 {
			t.Fatalf("expected 3 cleanups, got %d", len(order))
		}
		for i, expected := range []int{1, 2, 3} {
			if order[i] != expected {
				t.Errorf("cleanup %d called out of order: got %d, want %d", i, order[i], expected)
			}
		}
	})

	t.Run("ignores cleanup errors", func(t *testing.T) {
		originalErr := errors.New("original error")
		cleanupErr := errors.New("cleanup error")
		cleanup := func() error {
			return cleanupErr
		}

		result := CleanupOnError(originalErr, cleanup)
		if result != originalErr {
			t.Error("CleanupOnError should return the original error, not cleanup error")
		}
	})

	t.Run("handles nil cleanup functions", func(t *testing.T) {
		originalErr := errors.New("original error")
		cleanupCalled := false
		cleanup := func() error {
			cleanupCalled = true
			return nil
		}

		result := CleanupOnError(originalErr, nil, cleanup, nil)
		if result != originalErr {
			t.Error("CleanupOnError should return the original error")
		}
		if !cleanupCalled {
			t.Error("non-nil cleanup should still be called")
		}
	})
}

func TestCleanupOnErrorWith(t *testing.T) {
	t.Run("returns nil when err is nil", func(t *testing.T) {
		wrapErr := NewError(FileSystemError, "wrapped error")
		cleanupCalled := false
		cleanup := func() error {
			cleanupCalled = true
			return nil
		}

		result := CleanupOnErrorWith(nil, cleanup, wrapErr)
		if result != nil {
			t.Error("CleanupOnErrorWith should return nil when err is nil")
		}
		if cleanupCalled {
			t.Error("cleanup should not be called when err is nil")
		}
	})

	t.Run("runs cleanup and returns wrapped error", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrapErr := WrapError(FileSystemError, "wrapped error", originalErr)
		cleanupCalled := false
		cleanup := func() error {
			cleanupCalled = true
			return nil
		}

		result := CleanupOnErrorWith(originalErr, cleanup, wrapErr)
		if result != wrapErr {
			t.Error("CleanupOnErrorWith should return the wrapped error")
		}
		if !cleanupCalled {
			t.Error("cleanup should be called when err is not nil")
		}
	})

	t.Run("handles nil cleanup function", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrapErr := WrapError(FileSystemError, "wrapped error", originalErr)

		result := CleanupOnErrorWith(originalErr, nil, wrapErr)
		if result != wrapErr {
			t.Error("CleanupOnErrorWith should return the wrapped error even with nil cleanup")
		}
	})
}

func TestCleanupAll(t *testing.T) {
	t.Run("returns nil when all cleanups succeed", func(t *testing.T) {
		cleanup1 := func() error { return nil }
		cleanup2 := func() error { return nil }

		result := CleanupAll(cleanup1, cleanup2)
		if result != nil {
			t.Errorf("CleanupAll should return nil when all cleanups succeed, got: %v", result)
		}
	})

	t.Run("returns single error when one cleanup fails", func(t *testing.T) {
		cleanupErr := errors.New("cleanup error")
		cleanup1 := func() error { return cleanupErr }
		cleanup2 := func() error { return nil }

		result := CleanupAll(cleanup1, cleanup2)
		if result != cleanupErr {
			t.Errorf("CleanupAll should return the single error, got: %v", result)
		}
	})

	t.Run("combines multiple errors", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		cleanup1 := func() error { return err1 }
		cleanup2 := func() error { return err2 }

		result := CleanupAll(cleanup1, cleanup2)
		if result == nil {
			t.Fatal("CleanupAll should return an error")
		}
		// Should contain both error messages
		if !strings.Contains(result.Error(), "error 1") {
			t.Error("combined error should contain 'error 1'")
		}
		if !strings.Contains(result.Error(), "error 2") {
			t.Error("combined error should contain 'error 2'")
		}
	})

	t.Run("handles nil cleanup functions", func(t *testing.T) {
		cleanupCalled := false
		cleanup := func() error {
			cleanupCalled = true
			return nil
		}

		result := CleanupAll(nil, cleanup, nil)
		if result != nil {
			t.Errorf("CleanupAll should return nil when all non-nil cleanups succeed, got: %v", result)
		}
		if !cleanupCalled {
			t.Error("non-nil cleanup should still be called")
		}
	})

	t.Run("handles no cleanup functions", func(t *testing.T) {
		result := CleanupAll()
		if result != nil {
			t.Errorf("CleanupAll with no functions should return nil, got: %v", result)
		}
	})

	t.Run("runs all cleanups even when some fail", func(t *testing.T) {
		var callOrder []int
		cleanup1 := func() error {
			callOrder = append(callOrder, 1)
			return errors.New("error 1")
		}
		cleanup2 := func() error {
			callOrder = append(callOrder, 2)
			return nil
		}
		cleanup3 := func() error {
			callOrder = append(callOrder, 3)
			return errors.New("error 3")
		}

		_ = CleanupAll(cleanup1, cleanup2, cleanup3)
		if len(callOrder) != 3 {
			t.Fatalf("expected 3 cleanups, got %d", len(callOrder))
		}
		for i, expected := range []int{1, 2, 3} {
			if callOrder[i] != expected {
				t.Errorf("cleanup %d called out of order: got %d, want %d", i, callOrder[i], expected)
			}
		}
	})
}
