# PowerShell completion script for kairo
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
