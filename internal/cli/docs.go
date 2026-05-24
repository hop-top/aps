package cli

import (
	"fmt"
	"path/filepath"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentsDir, err := core.GetAgentsDir()
		if err != nil {
			return fmt.Errorf("getting agents dir: %w", err)
		}

		docsDest := filepath.Join(agentsDir, "docs")
		if err := core.GenerateDocs(docsDest); err != nil {
			return fmt.Errorf("generating docs: %w", err)
		}

		fmt.Printf("Documentation generated at: %s\n", docsDest)
		return nil
	},
}

func init() {
	// T-0648 — kit signature annotations. `docs` writes the generated
	// documentation tree under the resolved agents dir; rerunning
	// produces the same artefacts (idempotent overwrite).
	kitcli.SetSideEffect(docsCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(docsCmd, kitcli.IdempotencyYes)
	// T-0656 — docs walks the live cobra tree and writes a deterministic
	// markdown tree; preview would have to render the same output and
	// then throw it away.
	kitcli.OptOutDryRun(docsCmd)
	if err := kitcli.SetDryRunRationale(docsCmd, "docs renders a deterministic markdown tree from the live cobra command tree; previewing would have to compute the same output and discard it instead of writing it."); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(docsCmd)
}
