package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// createConfigBackup creates a backup of the current configuration file.
// Returns the path to the backup file or an error if the backup fails.
// The backup file is named with a timestamp to allow for multiple backups.
func createConfigBackup(configDir string) (string, error) {
	configPath := getConfigPath(configDir)

	// Read the current config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	// Create backup filename with timestamp
	backupPath := getBackupPath(configDir)

	// Write the backup
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// rollbackConfig restores the configuration from a backup file.
// If successful, the current config is replaced with the backup.
// The backup file is preserved after rollback for safety.
func rollbackConfig(configDir, backupPath string) error {
	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Read backup data
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Write to config file
	configPath := getConfigPath(configDir)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to restore config from backup: %w", err)
	}

	return nil
}

// withConfigTransaction executes a function within a transaction-like context.
//
// This function creates a backup of the configuration before executing the
// provided function. If the function returns an error, the configuration
// is automatically rolled back to the backup. This provides atomic-like
// behavior for configuration updates.
//
// Parameters:
//   - configDir: Directory containing the configuration file
//   - fn: Function to execute within the transaction context
//
// Returns:
//   - error: Returns error if transaction fails or rollback fails
//
// Error conditions:
//   - Returns error when unable to create configuration backup
//   - Returns error when fn returns an error (after attempting rollback)
//   - Returns error if rollback fails after transaction failure (critical error)
//
// Thread Safety: Not thread-safe due to file I/O operations
// Security Notes: Backup files retain same permissions as original config (0600)
func withConfigTransaction(configDir string, fn func(txDir string) error) error {
	// Create backup before transaction
	backupPath, err := createConfigBackup(configDir)
	if err != nil {
		return fmt.Errorf("failed to create transaction backup: %w", err)
	}

	// Execute the transaction function
	err = fn(configDir)

	// If transaction failed, rollback
	if err != nil {
		if rbErr := rollbackConfig(configDir, backupPath); rbErr != nil {
			// Rollback failed - this is a critical situation
			return fmt.Errorf("transaction failed and rollback also failed: tx_err=%w, rollback_err=%w", err, rbErr)
		}
		return fmt.Errorf("transaction failed, changes rolled back: %w", err)
	}

	// Transaction succeeded - clean up the backup file
	// Best-effort cleanup, ignore errors
	_ = os.Remove(backupPath)

	return nil
}

// getConfigPath returns the full path to the config file.
func getConfigPath(configDir string) string {
	return filepath.Join(configDir, "config.yaml")
}

// getBackupPath returns a backup file path with timestamp.
func getBackupPath(configDir string) string {
	// Use nanosecond precision to avoid filename conflicts with rapid successive operations
	timestamp := time.Now().Format("20060102-150405.000000000")
	return filepath.Join(configDir, fmt.Sprintf("config.yaml.backup.%s", timestamp))
}
