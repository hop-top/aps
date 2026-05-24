package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/clinote"
	kitcli "hop.top/kit/go/console/cli"
)

// NewJoinCmd creates the "collab join" command.
func NewJoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join <workspace>",
		Short: "Join a collaboration workspace",
		Long: `Join an existing collaboration workspace as a contributor.

You must specify your profile to identify which agent is joining.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID := args[0]

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			ctx := clinote.WithContext(cmd.Context(), clinote.FromCmd(cmd)) // T-1291
			agent, err := mgr.Join(ctx, wsID, profile)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(agent)
			}

			fmt.Printf("Joined '%s' as contributor\n", wsID)
			fmt.Println()
			fmt.Println("  Next steps:")
			fmt.Printf("    aps workspace use %s\n", wsID)
			fmt.Printf("    aps workspace members %s\n", wsID)

			return nil
		},
	}

	addProfileFlag(cmd)
	_ = cmd.MarkFlagRequired("profile")
	addJSONFlag(cmd)
	clinote.AddFlag(cmd) // T-1291

	// T-0648 — registers an agent in a local workspace; re-joining as
	// the same profile yields the same membership (idempotent).
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — join appends a single membership entry; the result is
	// fully determined by --workspace and --profile.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "join appends a single membership entry to the workspace; the resulting record is fully determined by --workspace and --profile."); err != nil {
		panic(err)
	}

	return cmd
}
