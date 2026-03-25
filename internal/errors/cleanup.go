package errors

import "fmt"

type CleanupFunc func() error

func CleanupOnError(err error, cleanups ...CleanupFunc) error {
	if err == nil {
		return nil
	}
	for _, cleanup := range cleanups {
		if cleanup != nil {
			_ = cleanup()
		}
	}

	return err
}

func CleanupOnErrorWith(err error, cleanup CleanupFunc, wrapErr *KairoError) *KairoError {
	if err == nil {
		return nil
	}
	if cleanup != nil {
		_ = cleanup()
	}

	return wrapErr
}

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
	msg := "multiple cleanup errors:"
	for _, e := range errs {
		msg += fmt.Sprintf("\n  - %v", e)
	}

	return NewError(RuntimeError, msg)
}
