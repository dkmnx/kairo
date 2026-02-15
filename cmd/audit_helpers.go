package cmd

import (
	"github.com/dkmnx/kairo/internal/audit"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
)

// logAuditEvent logs an audit event using the provided logging function.
//
// This function creates an audit logger, executes the provided logging function,
// and ensures the logger is properly closed. It wraps all errors with
// descriptive context for debugging.
//
// Parameters:
//   - configDir: Directory containing the audit log file
//   - logFunc: Function that performs the actual logging operation
//
// Returns:
//   - error: Returns error if logger creation or logging fails
//
// Error conditions:
//   - Returns error when unable to create audit logger (e.g., permissions, invalid directory)
//   - Returns error when logFunc returns an error
//
// Thread Safety: Not thread-safe due to file I/O operations
func logAuditEvent(configDir string, logFunc func(*audit.Logger) error) error {
	logger, err := audit.NewLogger(configDir)
	if err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to create audit logger", err)
	}
	defer logger.Close()

	if err := logFunc(logger); err != nil {
		return kairoerrors.WrapError(kairoerrors.ConfigError,
			"failed to log audit event", err)
	}
	return nil
}
