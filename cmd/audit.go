package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
	kairoerrors "github.com/dkmnx/kairo/internal/errors"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log",
	Long:  "View and export the audit log of configuration changes and provider switches",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getConfigDir()
		if dir == "" {
			return kairoerrors.NewError(kairoerrors.ConfigError,
				"config directory not found")
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			ui.PrintInfo("No audit log entries yet")
			return nil
		}

		logger, err := audit.NewLogger(dir)
		if err != nil {
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				"failed to open audit log", err).
				WithContext("config_dir", dir)
		}
		defer logger.Close()

		entries, err := logger.LoadEntries()
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintInfo("No audit log entries yet")
				return nil
			}
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				"failed to read audit log", err).
				WithContext("config_dir", dir)
		}

		if len(entries) == 0 {
			ui.PrintInfo("No audit log entries")
			return nil
		}

		if exportOutput != "" {
			if err := exportAuditLog(entries, exportOutput, exportFormat); err != nil {
				return kairoerrors.WrapError(kairoerrors.ConfigError,
					"failed to export audit log", err).
					WithContext("output_path", exportOutput).
					WithContext("format", exportFormat)
			}
			ui.PrintSuccess(fmt.Sprintf("Audit log exported to %s", exportOutput))
			return nil
		}

		printAuditList(entries, cmd)
		return nil
	},
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit entries",
	Long:  "Display audit entries in human-readable format",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getConfigDir()
		if dir == "" {
			return kairoerrors.NewError(kairoerrors.ConfigError,
				"config directory not found")
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			ui.PrintInfo("No audit log entries yet")
			return nil
		}

		logger, err := audit.NewLogger(dir)
		if err != nil {
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				"failed to open audit log", err).
				WithContext("config_dir", dir)
		}
		defer logger.Close()

		entries, err := logger.LoadEntries()
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintInfo("No audit log entries yet")
				return nil
			}
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				"failed to read audit log", err).
				WithContext("config_dir", dir)
		}

		if len(entries) == 0 {
			ui.PrintInfo("No audit log entries")
			return nil
		}

		printAuditList(entries, cmd)
		return nil
	},
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit log",
	Long:  "Export audit log to CSV or JSON format",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getConfigDir()
		if dir == "" {
			return kairoerrors.NewError(kairoerrors.ConfigError,
				"config directory not found")
		}

		if exportOutput == "" {
			return kairoerrors.NewError(kairoerrors.ConfigError,
				"--output is required for export")
		}

		// Audit directory must exist for export - don't create empty files
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return kairoerrors.NewError(kairoerrors.ConfigError,
				"no audit log directory exists")
		}

		logger, err := audit.NewLogger(dir)
		if err != nil {
			return kairoerrors.WrapError(kairoerrors.FileSystemError,
				"failed to open audit log", err).
				WithContext("config_dir", dir)
		}
		defer logger.Close()

		entries, err := logger.LoadEntries()
		if err != nil {
			if os.IsNotExist(err) {
				entries = []audit.AuditEntry{}
			} else {
				return kairoerrors.WrapError(kairoerrors.FileSystemError,
					"failed to read audit log", err).
					WithContext("config_dir", dir)
			}
		}

		if err := exportAuditLog(entries, exportOutput, exportFormat); err != nil {
			return err
		}

		ui.PrintSuccess(fmt.Sprintf("Audit log exported to %s", exportOutput))
		return nil
	},
}

func init() {
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditExportCmd)
	rootCmd.AddCommand(auditCmd)

	auditExportCmd.Flags().StringVarP(&exportFormat, "format", "f", "csv", "Export format (csv, json)")
	auditExportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (required for export)")
}

func printAuditList(entries []audit.AuditEntry, cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%sAudit Log:%s\n", ui.White, ui.Reset)
	fmt.Fprintln(out)

	for _, entry := range entries {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		event := entry.Event
		provider := entry.Provider

		// Add status indicator
		statusIcon := ""
		if entry.Status == "failure" {
			statusIcon = ui.Red + "[FAILED]" + ui.Reset + " "
		}

		switch entry.Event {
		case "switch":
			fmt.Fprintf(out, "  %s  [%s] %sSwitched to %s\n", ui.Gray, timestamp, statusIcon, ui.Green+provider+ui.Reset)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		case "config":
			changes := ""
			if len(entry.Changes) > 0 {
				var changeParts []string
				for _, c := range entry.Changes {
					if c.Old != "" {
						changeParts = append(changeParts, fmt.Sprintf("%s: %s -> %s", c.Field, c.Old, c.New))
					} else {
						changeParts = append(changeParts, fmt.Sprintf("%s: %s", c.Field, c.New))
					}
				}
				changes = " - " + strings.Join(changeParts, ", ")
			}
			fmt.Fprintf(out, "  %s  [%s] %sConfigured %s (%s)%s\n", ui.Gray, timestamp, statusIcon, ui.Blue+provider+ui.Reset, entry.Action, changes)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		case "rotate":
			fmt.Fprintf(out, "  %s  [%s] %sRotated keys for %s\n", ui.Gray, timestamp, statusIcon, ui.Yellow+provider+ui.Reset)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		case "default":
			fmt.Fprintf(out, "  %s  [%s] %sSet default provider to %s\n", ui.Gray, timestamp, statusIcon, ui.Green+provider+ui.Reset)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		case "reset":
			fmt.Fprintf(out, "  %s  [%s] %sReset configuration for %s\n", ui.Gray, timestamp, statusIcon, ui.Red+provider+ui.Reset)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		case "setup":
			// Display name from details if available
			displayName := provider
			if entry.Details["display_name"] != nil {
				if dn, ok := entry.Details["display_name"].(string); ok {
					displayName = dn
				}
			}
			fmt.Fprintf(out, "  %s  [%s] %sSet up %s\n", ui.Gray, timestamp, statusIcon, ui.Blue+displayName+ui.Reset)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		default:
			fmt.Fprintf(out, "  %s  [%s] %s%s %s\n", ui.Gray, timestamp, statusIcon, event, provider)
			if entry.Error != "" {
				fmt.Fprintf(out, "      %sError: %s%s\n", ui.Red, entry.Error, ui.Reset)
			}
		}

		// Show details if present
		if len(entry.Details) > 0 {
			for key, value := range entry.Details {
				fmt.Fprintf(out, "      %s%s: %v%s\n", ui.Gray, key, value, ui.Reset)
			}
		}
	}
	fmt.Fprintln(out)
}

func exportAuditLog(entries []audit.AuditEntry, outputPath, format string) error {
	format = strings.ToLower(format)

	// Validate format early before any file operations to fail fast and prevent creating empty files
	if format != "json" && format != "csv" {
		return kairoerrors.NewError(kairoerrors.ConfigError,
			fmt.Sprintf("unsupported format: %s (supported: csv, json)", format))
	}

	if format == "json" {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(outputPath, data, 0600)
	}

	if format == "csv" {
		f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer f.Close()

		writer := csv.NewWriter(f)
		defer writer.Flush()

		if err := writer.Write([]string{"timestamp", "event", "provider", "action", "status", "error", "details", "changes"}); err != nil {
			return err
		}

		for _, entry := range entries {
			changes := ""
			if len(entry.Changes) > 0 {
				var changeParts []string
				for _, c := range entry.Changes {
					if c.Old != "" {
						changeParts = append(changeParts, fmt.Sprintf("%s: %s -> %s", c.Field, c.Old, c.New))
					} else {
						changeParts = append(changeParts, fmt.Sprintf("%s: %s", c.Field, c.New))
					}
				}
				changes = strings.Join(changeParts, ", ")
			}

			// Format details as key:value pairs for readability
			details := ""
			if len(entry.Details) > 0 {
				var detailParts []string
				for key, value := range entry.Details {
					detailParts = append(detailParts, fmt.Sprintf("%s:%v", key, value))
				}
				details = strings.Join(detailParts, ", ")
			}

			record := []string{
				entry.Timestamp.Format("2006-01-02T15:04:05Z"),
				entry.Event,
				entry.Provider,
				entry.Action,
				entry.Status,
				entry.Error,
				details,
				changes,
			}
			if err := writer.Write(record); err != nil {
				return kairoerrors.WrapError(kairoerrors.FileSystemError,
					"failed to write CSV record", err).
					WithContext("output_path", outputPath)
			}
		}

		return nil
	}

	return kairoerrors.NewError(kairoerrors.ConfigError,
		fmt.Sprintf("unsupported format: %s (supported: csv, json)", format)).
		WithContext("format", format).
		WithContext("supported", "csv, json")
}
