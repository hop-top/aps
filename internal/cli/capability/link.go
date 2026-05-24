package capability

import (
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newLinkCmd() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "link <name> [--target <path>]",
		Short: "Symlink a capability to a target path",
		Long: `Create a symlink at --target pointing back at the capability
stored under $APS_DATA_PATH/capabilities/<name>/. When --target is
omitted, the command consults the smart-pattern registry (see aps
capability patterns list) and, if <name> matches a known tool,
links into that tool's conventional location.

Local write of a single filesystem symlink. Idempotent on the
source/target pair. --dry-run is opted out because the source and
target paths are fully determined by --name and --target; preview
would only echo the inputs. Use --note to attach an audit reason
that flows to the event bus alongside the mutation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if target == "" {
				if pattern, err := capability.GetSmartPattern(name); err == nil {
					target = pattern.ToolName
					fmt.Println(dimStyle.Render(fmt.Sprintf(
						"Smart link: %s -> %s", name, pattern.DefaultPath)))
				} else {
					return fmt.Errorf(
						"--target required unless using a Smart Pattern name")
				}
			}

			if err := capability.Link(name, target); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Linked") + " " +
				boldStyle.Render(name) + " -> " + target)
			return nil
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Target path for symlink")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — link creates a symlink to the --target path; preview
	// would only restate the source and target paths from the flags.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "link creates one filesystem symlink whose source and target are fully determined by --name and --target; previewing would only echo the inputs."); err != nil {
		panic(err)
	}
	return cmd
}
