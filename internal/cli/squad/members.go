package squad

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newMembersCmd returns the `squad members` mid-level command grouping
// membership operations (add, remove).
func newMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Manage squad membership",
	}
	cmd.AddCommand(newAddMemberCmd())
	cmd.AddCommand(newRemoveMemberCmd())
	return cmd
}

func newAddMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <squad-id> <profile-id>",
		Short: "Add a member to a squad",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddMember(args[0], args[1])
		},
	}
}

func runAddMember(squadID, profileID string) error {
	if err := defaultManager.AddMember(squadID, profileID); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Added %q to squad %q\n", profileID, squadID)
	return nil
}

func newRemoveMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <squad-id> <profile-id>",
		Short: "Remove a member from a squad",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveMember(args[0], args[1])
		},
	}
}

func runRemoveMember(squadID, profileID string) error {
	if err := defaultManager.RemoveMember(squadID, profileID); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Removed %q from squad %q\n", profileID, squadID)
	return nil
}
