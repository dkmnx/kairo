package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	completionOutput string
	completionSave   bool
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:
  $ source <(kairo completion bash)

  # To load completions for every new session:
  $ kairo completion bash --save

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for every new session:
  $ kairo completion zsh --save

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ kairo completion fish | source

  # To load completions for every new session:
  $ kairo completion fish --save

PowerShell:
  PS> kairo completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session:
  PS> kairo completion powershell --save
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		var out io.Writer
		var closeOut bool

		// Determine output destination.
		if completionOutput != "" {
			// Write to specified file.
			f, err := os.Create(completionOutput)
			if err != nil {
				cmd.Printf("Error creating output file: %v\n", err)
				return
			}
			out = f
			closeOut = true
		} else if completionSave {
			// Auto-save to default location.
			defaultPath := getDefaultCompletionPath(args[0])
			if err := os.MkdirAll(filepath.Dir(defaultPath), 0755); err != nil {
				cmd.Printf("Error creating directory: %v\n", err)
				return
			}
			f, err := os.Create(defaultPath)
			if err != nil {
				cmd.Printf("Error creating output file: %v\n", err)
				return
			}
			cmd.Printf("Completion saved to: %s\n", defaultPath)
			out = f
			closeOut = true
		} else {
			// Write to stdout (use cmd's output to respect SetOut in tests).
			out = cmd.OutOrStdout()
		}

		// Generate completion for the specified shell.
		switch args[0] {
		case "bash":
			if err := rootCmd.GenBashCompletion(out); err != nil {
				cmd.Printf("Error generating bash completion: %v\n", err)
			}
		case "zsh":
			if err := rootCmd.GenZshCompletion(out); err != nil {
				cmd.Printf("Error generating zsh completion: %v\n", err)
			}
		case "fish":
			if err := rootCmd.GenFishCompletion(out, true); err != nil {
				cmd.Printf("Error generating fish completion: %v\n", err)
			}
		case "powershell":
			if err := rootCmd.GenPowerShellCompletionWithDesc(out); err != nil {
				cmd.Printf("Error generating PowerShell completion: %v\n", err)
			}
		}

		if closeOut {
			if f, ok := out.(*os.File); ok {
				f.Close()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.Flags().StringVarP(&completionOutput, "output", "o", "", "Output file path")
	completionCmd.Flags().BoolVar(&completionSave, "save", false, "Auto-save to default shell completion directory")
}

// getDefaultCompletionPath returns the default completion file path for a shell.
func getDefaultCompletionPath(shell string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return "kairo-completion.sh"
	}

	switch shell {
	case "bash":
		// Use user's home directory for writable location.
		return filepath.Join(home, ".bash_completion.d", "kairo")
	case "zsh":
		// Try to find fpath from zsh.
		return filepath.Join(home, ".zsh", "completion", "_kairo")
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "kairo.fish")
	case "powershell":
		// Use PowerShell Modules directory for auto-loading.
		// PowerShell auto-loads .psm1 files from $env:USERPROFILE\Documents\PowerShell\Modules.
		return filepath.Join(home, "Documents", "PowerShell", "Modules", "kairo-completion", "kairo-completion.psm1")
	default:
		return "kairo-completion.sh"
	}
}
