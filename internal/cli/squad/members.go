package squad

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newAddMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-member <squad-id> <profile-id>",
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
		Use:   "remove-member <squad-id> <profile-id>",
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
