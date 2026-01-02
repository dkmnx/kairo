package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dkmnx/kairo/internal/audit"
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
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			ui.PrintInfo("No audit log entries yet")
			return
		}

		logger, err := audit.NewLogger(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to open audit log: %v", err))
			return
		}

		entries, err := logger.LoadEntries()
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintInfo("No audit log entries yet")
				return
			}
			ui.PrintError(fmt.Sprintf("Failed to read audit log: %v", err))
			return
		}

		if len(entries) == 0 {
			ui.PrintInfo("No audit log entries")
			return
		}

		if exportOutput != "" {
			if err := exportAuditLog(entries, exportOutput, exportFormat); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to export audit log: %v", err))
				return
			}
			ui.PrintSuccess(fmt.Sprintf("Audit log exported to %s", exportOutput))
			return
		}

		printAuditList(entries, cmd)
	},
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit entries",
	Long:  "Display audit entries in human-readable format",
	Run: func(cmd *cobra.Command, args []string) {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			ui.PrintInfo("No audit log entries yet")
			return
		}

		logger, err := audit.NewLogger(dir)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to open audit log: %v", err))
			return
		}

		entries, err := logger.LoadEntries()
		if err != nil {
			if os.IsNotExist(err) {
				ui.PrintInfo("No audit log entries yet")
				return
			}
			ui.PrintError(fmt.Sprintf("Failed to read audit log: %v", err))
			return
		}

		if len(entries) == 0 {
			ui.PrintInfo("No audit log entries")
			return
		}

		printAuditList(entries, cmd)
	},
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit log",
	Long:  "Export audit log to CSV or JSON format",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := getConfigDir()
		if dir == "" {
			ui.PrintError("Config directory not found")
			return nil
		}

		if exportOutput == "" {
			return fmt.Errorf("--output is required for export")
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			entries := []audit.AuditEntry{}
			if err := exportAuditLog(entries, exportOutput, exportFormat); err != nil {
				return err
			}
			ui.PrintSuccess(fmt.Sprintf("Audit log exported to %s", exportOutput))
			return nil
		}

		logger, err := audit.NewLogger(dir)
		if err != nil {
			return err
		}

		entries, err := logger.LoadEntries()
		if err != nil {
			if os.IsNotExist(err) {
				entries = []audit.AuditEntry{}
			} else {
				return err
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
	auditListCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
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

		switch entry.Event {
		case "switch":
			fmt.Fprintf(out, "  %s  [%s] Switched to %s\n", ui.Gray, timestamp, ui.Green+provider+ui.Reset)
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
			fmt.Fprintf(out, "  %s  [%s] Configured %s (%s)%s\n", ui.Gray, timestamp, ui.Blue+provider+ui.Reset, entry.Action, changes)
		case "rotate":
			fmt.Fprintf(out, "  %s  [%s] Rotated keys for %s\n", ui.Gray, timestamp, ui.Yellow+provider+ui.Reset)
		case "default":
			fmt.Fprintf(out, "  %s  [%s] Set default provider to %s\n", ui.Gray, timestamp, ui.Green+provider+ui.Reset)
		case "reset":
			fmt.Fprintf(out, "  %s  [%s] Reset configuration for %s\n", ui.Gray, timestamp, ui.Red+provider+ui.Reset)
		case "setup":
			fmt.Fprintf(out, "  %s  [%s] Set up %s\n", ui.Gray, timestamp, ui.Blue+provider+ui.Reset)
		default:
			fmt.Fprintf(out, "  %s  [%s] %s %s\n", ui.Gray, timestamp, event, provider)
		}
	}
	fmt.Fprintln(out)
}

func exportAuditLog(entries []audit.AuditEntry, outputPath, format string) error {
	format = strings.ToLower(format)

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

		if err := writer.Write([]string{"timestamp", "event", "provider", "action", "changes"}); err != nil {
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
			record := []string{
				entry.Timestamp.Format("2006-01-02T15:04:05Z"),
				entry.Event,
				entry.Provider,
				entry.Action,
				changes,
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}

		return nil
	}

	return fmt.Errorf("unsupported format: %s (supported: csv, json)", format)
}
