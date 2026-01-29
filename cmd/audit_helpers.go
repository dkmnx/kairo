package cmd

import (
	"fmt"

	"github.com/dkmnx/kairo/internal/audit"
)

func logAuditEvent(configDir string, logFunc func(*audit.Logger) error) error {
	logger, err := audit.NewLogger(configDir)
	if err != nil {
		return fmt.Errorf("failed to create audit logger: %w", err)
	}
	defer logger.Close()

	if err := logFunc(logger); err != nil {
		return fmt.Errorf("failed to log audit event: %w", err)
	}
	return nil
}
