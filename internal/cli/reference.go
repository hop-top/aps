package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RootCommand returns the root cobra command in the state the binary
// dispatches it: aps-side setup (decorateTree) plus everything kit
// installs before parsing argv (Root.Prepare: kit-managed flags such as
// --confirm-token and --idempotency-key, the completion command, help
// addenda, the RunE middleware). Documentation tooling
// (internal/tools/climd, invoked from the cog markers via `make
// docs-gen`) renders from it. Both steps are idempotent. Callers must
// treat the returned tree as read-only outside doc generation and tests.
func RootCommand() (*cobra.Command, error) {
	decorateTree()
	if err := root.Prepare(); err != nil {
		return nil, fmt.Errorf("preparing command tree: %w", err)
	}
	return rootCmd, nil
}
