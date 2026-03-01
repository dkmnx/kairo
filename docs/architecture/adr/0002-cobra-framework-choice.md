# ADR 0002: Cobra Framework Choice

## Context

Kairo is a CLI tool that needs:

- Subcommand structure (setup, list, delete, switch, etc.)
- Flag management (config path, verbose output, etc.)
- Help generation
- Shell completion
- Error handling

## Decision

We chose **Cobra** (`github.com/spf13/cobra`) for the CLI framework.

### Why Cobra?

| Feature         | Cobra        | uvloop/urcli   | kingpin   | docopt   |
| --------------- | ------------ | -------------- | --------- | -------- |
| Subcommands     | Full support | Limited        | Yes       | Limited  |
| Flag management | Built-in     | Custom         | Yes       | Custom   |
| Help generation | Automatic    | Manual         | Yes       | Manual   |
| Completion      | Built-in     | Limited        | Yes       | Manual   |
| Stack depth     | Low          | Low            | Low       | Low      |
| Ecosystem       | Large        | Small          | Medium    | Small    |
| Learning curve  | Moderate     | Low            | Low       | High     |

### Why not alternatives?

- **kingpin**: Simple but limited subcommand nesting, less "Go-like"
- **docopt**: Requires learning docopt syntax, runtime parsing overhead
- **Built-in flag**: Too limited for complex CLI with subcommands

## Consequences

### Positive

- **Rich feature set**: Automatic help, completion, flag groups
- **Large ecosystem**: Subcommands, Viper integration, etc.
- **Popular**: Well-documented, common pattern for Go CLIs
- **Testability**: Cobra provides testing helpers

### Negative

- **Dependency**: One more external dependency (minimal impact)
- **Learning curve**: Team needs to learn Cobra patterns

## Implementation

```go
// cmd/root.go - Root command
var rootCmd = &cobra.Command{
    Use:   "kairo",
    Short: "Kairo - Manage Claude Code API providers",
    Long:  `...`,
    Run:   executeRootCommand,
}
```

```go
// cmd/setup.go - Subcommand
var setupCmd = &cobra.Command{
    Use:   "setup",
    Short: "Configure providers",
    Run:   executeSetup,
}
```

## Status

**Accepted** - Implemented and in use since v0.1.0
