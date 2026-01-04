package cmd

import (
	"fmt"
	"os"

	"github.com/dkmnx/kairo/internal/audit"
)

func logAuditEvent(configDir string, logFunc func(*audit.Logger) error) {
	logger, err := audit.NewLogger(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create audit logger: %v\n", err)
		return
	}

	if err := logFunc(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log audit event: %v\n", err)
	}
}
