# Research & Decisions

## Shell Completion
- **Decision**: Use `cobra.Command.GenBashCompletion`, `GenZshCompletion`, etc.
- **Rationale**: Built-in Cobra feature, robust and standard.
- **Dynamic Completion**: Use `ValidArgsFunction` on the root command to dynamically fetch profile IDs for completion.

## Profile Shorthand (`aps <profile>`)
- **Decision**: Implement `Args: cobra.ArbitraryArgs` on `rootCmd` and handle dispatch in `Run`.
- **Logic**:
  1. Check if `args[0]` matches a valid profile ID.
  2. If yes, treat as `run <profile> -- <rest>`.
  3. If no, show Help (or TUI if no args).
- **Conflict**: Subcommands take precedence over profiles (Cobra handles this by matching subcommands first).

## Alias Generation
- **Decision**: Generate simple shell functions or aliases depending on shell.
- **Format**: `alias <name>='aps run <name> --'`? No, spec says `alias agent-a='aps agent-a'`.
- **Conflict Detection**: Use `exec.LookPath(name)` to warn if `name` exists in PATH.

## Default Shell
- **Decision**:
  1. Store `DefaultShell` in `Profile.Preferences`.
  2. `aps profile new` detects current shell via `$SHELL` env var (or `os.Getenv("SHELL")`).
  3. `aps <profile>` (no args) launches this shell.
