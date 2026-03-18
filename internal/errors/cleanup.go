package errors

import "fmt"

// CleanupFunc represents a function that performs cleanup and may return an error.
type CleanupFunc func() error

// CleanupOnError runs all cleanup functions if err is not nil.
// Cleanup errors are ignored to preserve the original error.
// Returns the original error unchanged.
//
// Usage:
//
//	file, err := os.Create(tempPath)
//	if err != nil {
//	    return CleanupOnError(err, file.Close, func() error { return os.Remove(tempPath) })
//	}
func CleanupOnError(err error, cleanups ...CleanupFunc) error {
	if err == nil {
		return nil
	}
	for _, cleanup := range cleanups {
		if cleanup != nil {
			// Ignore cleanup errors to preserve original error
			_ = cleanup()
		}
	}

	return err
}

// CleanupOnErrorWith wraps the error after running cleanup functions.
// Use this when you want to wrap the error with additional context after cleanup.
//
// Usage:
//
//	file, err := os.Create(tempPath)
//	if err != nil {
//	    return CleanupOnErrorWith(err, func() error { return os.Remove(tempPath) },
//	        WrapError(FileSystemError, "failed to create file", err))
//	}
func CleanupOnErrorWith(err error, cleanup CleanupFunc, wrapErr *KairoError) *KairoError {
	if err == nil {
		return nil
	}
	if cleanup != nil {
		_ = cleanup()
	}

	return wrapErr
}

// CleanupAll runs all cleanup functions regardless of errors.
// Collects all cleanup errors and returns them as a combined error.
// Returns nil if all cleanups succeed.
//
// Usage:
//
//	defer func() {
//	    if err := CleanupAll(file.Close, func() error { return os.Remove(tempPath) }); err != nil {
//	        log.Printf("cleanup warnings: %v", err)
//	    }
//	}()
func CleanupAll(cleanups ...CleanupFunc) error {
	var errs []error
	for _, cleanup := range cleanups {
		if cleanup != nil {
			if err := cleanup(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	// Combine multiple errors
	msg := "multiple cleanup errors:"
	for _, e := range errs {
		msg += fmt.Sprintf("\n  - %v", e)
	}

	return NewError(RuntimeError, msg)
}
