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
		Long: `Append a profile to a squad's member list. Both arguments are
positional: <squad-id> is the squad slug (see aps squad list) and
<profile-id> is the profile ID. The squad must exist; the profile
ID is recorded verbatim and is not cross-validated against the
profile store at this layer.

Local write to the squad record only — pair with aps squad members
remove to undo. Not idempotent at the manager level (re-running
returns an error on duplicate membership). --dry-run is opted out
because the result is fully determined by the two positional
arguments. Use --note to attach an audit reason that flows to the
event bus alongside the mutation.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddMember(args[0], args[1])
		},
	}
	clinote.AddFlag(cmd) // T-1291
	// T-0648 — kit 0.4 signature annotations.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — add appends a single member entry to the squad; the
	// result is fully determined by the two positional arguments.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "add appends a member entry to the squad; the resulting record is fully determined by the two positional arguments."); err != nil {
		panic(err)
	}
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
		Long: `Drop a profile from a squad's member list. Both arguments are
positional: <squad-id> is the squad slug (see aps squad list) and
<profile-id> is the profile ID to remove. The profile record
itself is untouched — only the squad's membership entry is
dropped.

Destructive at the squad level: the membership entry is removed
irreversibly from the squad store. The destructive-token
confirmation flow gates the apply path, and --dry-run is opted out
because preview would only restate the two positional arguments.
Idempotent on already-absent membership.`,
		Args: cobra.ExactArgs(2),
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
	// T-0656 — destructive-token confirm already gates the apply path;
	// preview would only restate the squad/profile IDs.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "remove drops a member entry from the squad; the destructive-token confirm flow already gates the apply path, and preview would only restate the two positional arguments."); err != nil {
		panic(err)
	}
	return cmd
}

func runRemoveMember(squadID, profileID string) error {
	if err := defaultManager.RemoveMember(squadID, profileID); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Removed %q from squad %q\n", profileID, squadID)
	return nil
}
