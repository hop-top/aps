package squad

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

// newMembersCmd returns the `squad members` mid-level command grouping
// membership operations (add, remove).
func newMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Manage squad membership",
	}
	// T-0648 — intermediate ancestor for depth-3 leaves
	// (`aps squad members add|remove`). Signature validator requires
	// kit/hierarchical on every intermediate ancestor of a depth-3
	// leaf.
	kitcli.SetHierarchical(cmd)
	cmd.AddCommand(newAddMemberCmd())
	cmd.AddCommand(newRemoveMemberCmd())
	return cmd
}

func newAddMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <squad-id> <profile-id>",
		Short: "Add a member to a squad",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddMember(args[0], args[1])
		},
	}
	clinote.AddFlag(cmd) // T-1291
	// T-0648 — kit 0.4 signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	return cmd
}

func runAddMember(squadID, profileID string) error {
	if err := defaultManager.AddMember(squadID, profileID); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Added %q to squad %q\n", profileID, squadID)
	return nil
}

func newRemoveMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <squad-id> <profile-id>",
		Short: "Remove a member from a squad",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveMember(args[0], args[1])
		},
	}
	clinote.AddFlag(cmd) // T-1291
	// T-0654 — removing a squad member is an irreversible mutation on
	// the local squad store. Remove-by-id is naturally idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectDestructiveLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	kitcli.SetDestructiveToken(cmd)
	return cmd
}

func runRemoveMember(squadID, profileID string) error {
	if err := defaultManager.RemoveMember(squadID, profileID); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Removed %q from squad %q\n", profileID, squadID)
	return nil
}
