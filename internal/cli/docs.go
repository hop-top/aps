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
	Long: `Write the documentation set embedded in the aps binary (README,
CLI, PROFILES, ISOLATION, SESSIONS, EXAMPLES, WEBHOOKS, SECURITY,
and SPEC, as markdown) to <data-dir>/docs/. The data dir is
$APS_DATA_PATH when set, else $XDG_DATA_HOME/aps, else
~/.local/share/aps. There is no flag or config key to choose another
destination.

The files are a fixed copy shipped with the binary, not generated
from the live command tree (aps <command> --help is the live
reference), so rerunning overwrites existing files with identical
content for the same binary. --dry-run is opted out because the
preview would have to list the same files the command writes.

Idempotent overwrite: safe to rerun. No network calls.`,
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
	// documentation tree under the resolved data dir; rerunning
	// produces the same artefacts (idempotent overwrite).
	kitcli.SetSideEffect(docsCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(docsCmd, kitcli.IdempotencyYes)
	// T-0679 — depth-1 runnable leaf under `aps`; mark intentional so the
	// shape validator (kit/top-level-verb) accepts it under EnforceValidate.
	kitcli.SetTopLevelVerb(docsCmd)
	// docs copies the embedded markdown set; preview would
	// have to list the same files the command writes.
	kitcli.OptOutDryRun(docsCmd)
	if err := kitcli.SetDryRunRationale(docsCmd, "docs copies the embedded markdown set to the data dir; previewing would only list the same files it writes."); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(docsCmd)
}
