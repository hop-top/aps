package cli

import "github.com/spf13/cobra"

// RootCommand returns the fully assembled root cobra command so
// documentation tooling (internal/tools/climd, invoked from the cog
// marker in docs/cli/commands.md via `make docs-gen`) can render the
// live command tree. Callers must treat the returned tree as
// read-only outside doc generation and tests.
func RootCommand() *cobra.Command { return rootCmd }
