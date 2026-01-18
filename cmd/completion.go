package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// powerShellCompletionScript is the Register-ArgumentCompleter script for PowerShell
const powerShellCompletionScript = `# PowerShell completion script for kairo
# Usage: . ./kairo-completion.ps1 (or add to your PowerShell profile)

# Register the completer for the native kairo command
Register-ArgumentCompleter -Native -CommandName kairo -ScriptBlock {
    param(
        $wordToComplete,
        $commandAst,
        $cursorPosition
    )

    # Call kairo __complete to get completion results from cobra
    $commandElements = $commandAst.CommandElements
    $commandString = $commandElements.ToString()

    # Build arguments for __complete command
    $completerArgs = @("__complete") + $commandElements[1..($commandElements.Count - 1)]
    $completerArgs += @($wordToComplete, $cursorPosition.ToString())

    # Run kairo __complete and capture output
    $completionOutput = & kairo @completerArgs 2>&1

    # Parse JSON output from cobra's __complete command
    try {
        $completions = $completionOutput | ConvertFrom-Json

        # Must unroll results using pipeline (ForEach-Object)
        $completions | ForEach-Object {
            # Create CompletionResult with description if available
            if ($_.Description) {
                New-Object -Type System.Management.Automation.CompletionResult -ArgumentList @(
                    $_.CompletionText,  # completionText
                    $_.CompletionText,  # listItemText
                    'ParameterValue',      # resultType
                    $_.Description         # toolTip
                )
            } else {
                New-Object -Type System.Management.Automation.CompletionResult -ArgumentList @(
                    $_.CompletionText,
                    $_.CompletionText,
                    'ParameterValue',
                    $_.CompletionText
                )
            }
        }
    }
    catch {
        # If JSON parsing fails, return empty array
        @()
    }
}
`

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
  # Then add this to your PowerShell profile ($PROFILE):
  #     Register-ArgumentCompleter -Native -CommandName kairo -ScriptBlock {
  #         param($wordToComplete, $commandAst, $cursorPosition)
  #         kairo __complete $commandAst.ToString().Split()[1..$commandAst.Count] $wordToComplete $cursorPosition
  #     }
  #   }
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

			// For PowerShell, copy the prepared script with Register-ArgumentCompleter
			if args[0] == "powershell" {
				if err := os.WriteFile(defaultPath, []byte(powerShellCompletionScript), 0644); err != nil {
					cmd.Printf("Error writing completion file: %v\n", err)
					return
				}
				cmd.Printf("Completion saved to: %s\n", defaultPath)
				cmd.Printf("\nTo load completions, add this line to your PowerShell profile:\n")
				cmd.Printf("  . %s\n", defaultPath)
				cmd.Printf("\nTo edit your profile, run: notepad $PROFILE\n")
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
		// Save to home directory as a .ps1 script
		// User needs to source this in their PowerShell profile
		return filepath.Join(home, "kairo-completion.ps1")
	default:
		return "kairo-completion.sh"
	}
}
