package cmd

// testCLI is a shared *CLIContext used by tests that need to set up a config
// directory. Tests should use this instead of relying on CLIContextFromCmd
// when no cobra.Command is available.
var testCLI = NewCLIContext()
