package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/dkmnx/kairo/internal/performance"
	"github.com/dkmnx/kairo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	metricsOutputPath string
	metricsFormat     string
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Display performance metrics",
	Long: `Display performance metrics for API calls and configuration operations.

Metrics collection is opt-in for privacy. To enable metrics collection:
  • Set KAIRO_METRICS_ENABLED=true environment variable
  • Use --enable flag with this command

Metrics track:
  • API call durations per provider
  • Configuration operation times
  • Success/failure rates`,
	Run: func(cmd *cobra.Command, args []string) {
		registry := getMetricsRegistry()
		if !registry.Enabled {
			ui.PrintWarn("Metrics are disabled")
			ui.PrintInfo("Enable with: KAIRO_METRICS_ENABLED=true or --enable flag")
			return
		}

		stats := registry.GetStats()
		if len(stats) == 0 {
			ui.PrintInfo("No metrics recorded yet")
			return
		}

		// Handle export to file
		if metricsOutputPath != "" {
			if err := exportMetrics(registry); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to export metrics: %v", err))
				return
			}
			ui.PrintSuccess(fmt.Sprintf("Metrics exported to %s", metricsOutputPath))
			return
		}

		// Display metrics
		displayMetrics(stats)
	},
}

var metricsResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset all recorded metrics",
	Long:  "Clear all recorded performance metrics",
	Run: func(cmd *cobra.Command, args []string) {
		registry := getMetricsRegistry()
		if !registry.Enabled {
			ui.PrintWarn("Metrics are disabled")
			return
		}

		registry.Reset()
		ui.PrintSuccess("Metrics reset successfully")
	},
}

var metricsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable metrics collection",
	Long:  "Enable performance metrics collection for this session",
	Run: func(cmd *cobra.Command, args []string) {
		registry := getMetricsRegistry()
		registry.Enable()
		ui.PrintSuccess("Metrics collection enabled")
		ui.PrintInfo("Metrics will be collected until you disable them or exit")
	},
}

var metricsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable metrics collection",
	Long:  "Disable performance metrics collection",
	Run: func(cmd *cobra.Command, args []string) {
		registry := getMetricsRegistry()
		if !registry.Enabled {
			ui.PrintInfo("Metrics are already disabled")
			return
		}

		registry.Disable()
		ui.PrintSuccess("Metrics collection disabled")
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)

	// Add subcommands
	metricsCmd.AddCommand(metricsResetCmd)
	metricsCmd.AddCommand(metricsEnableCmd)
	metricsCmd.AddCommand(metricsDisableCmd)

	// Flags for metrics command
	metricsCmd.Flags().StringVarP(&metricsOutputPath, "output", "o", "", "Export metrics to file")
	metricsCmd.Flags().StringVarP(&metricsFormat, "format", "f", "json", "Output format (json, text)")
	metricsCmd.Flags().Bool("enable", false, "Enable metrics for display")
}

// Global metrics registry
var globalMetricsRegistry *performance.Registry

// getMetricsRegistry returns the global metrics registry.
func getMetricsRegistry() *performance.Registry {
	if globalMetricsRegistry == nil {
		globalMetricsRegistry = performance.NewRegistry()

		// Check environment variable
		if os.Getenv("KAIRO_METRICS_ENABLED") == "true" {
			globalMetricsRegistry.Enable()
		}
	}
	return globalMetricsRegistry
}

// displayMetrics shows metrics in human-readable format.
func displayMetrics(stats performance.Stats) {
	ui.PrintHeader("Performance Metrics")
	fmt.Println()

	for operation, providers := range stats {
		ui.PrintWhite(fmt.Sprintf("  %s", operation))
		fmt.Println()

		for provider, stat := range providers {
			if stat.Count > 0 {
				fmt.Printf("    %s%s\n", ui.Gray, provider)
				fmt.Printf("      Calls:      %d\n", stat.Count)
				fmt.Printf("      Avg:        %v\n", stat.AvgDuration.Round(time.Millisecond))
				fmt.Printf("      Min:        %v\n", stat.MinDuration.Round(time.Millisecond))
				fmt.Printf("      Max:        %v\n", stat.MaxDuration.Round(time.Millisecond))
				if stat.FailureCount > 0 {
					fmt.Printf("      Failures:   %d\n", stat.FailureCount)
				}
				fmt.Printf("    %s\n", ui.Reset)
			}
		}
	}

	// Print summary
	if summary := stats.Summary(); summary != "" {
		fmt.Println()
		ui.PrintHeader("Summary")
		fmt.Print(summary)
	}
}

// exportMetrics saves metrics to a file in the specified format.
func exportMetrics(registry *performance.Registry) error {
	var data []byte
	var err error

	switch metricsFormat {
	case "json":
		data, err = registry.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to convert to JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s (supported: json)", metricsFormat)
	}

	return os.WriteFile(metricsOutputPath, data, 0600)
}
