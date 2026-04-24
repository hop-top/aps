# Extending the APS CLI

APS uses the [Cobra](https://github.com/spf13/cobra) library for command routing.

## Command Registration

All top-level commands are registered via Go's `init()` mechanism in `internal/cli/`.

```go
// internal/cli/my_command.go
func init() {
    rootCmd.AddCommand(newMyCommand())
}
```

## Types of Extensions

1. **Adapter Promotion**: Taking a modular script adapter and giving it a dedicated CLI surface (e.g., `contact.go`).
2. **Aliases**: Providing context-specific shorthands (e.g., `messenger_alias.go`).
3. **Core Features**: Adding new systemic capabilities (e.g., `capability.go`, `squad.go`).

## Profile Dispatch

The root command in `internal/cli/root.go` handles the "Profile Dispatch" logic. If the first argument matches a profile ID, APS automatically assumes the rest of the arguments are a command to run within that profile's isolated environment.
