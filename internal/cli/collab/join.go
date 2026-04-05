package collab

import (
	"fmt"

	"github.com/spf13/cobra"
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

			agent, err := mgr.Join(cmd.Context(), wsID, profile)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(agent)
			}

			fmt.Printf("Joined '%s' as contributor\n", wsID)
			fmt.Println()
			fmt.Println("  Next steps:")
			fmt.Printf("    aps collab use %s\n", wsID)
			fmt.Printf("    aps collab members %s\n", wsID)

			return nil
		},
	}

	addProfileFlag(cmd)
	_ = cmd.MarkFlagRequired("profile")
	addJSONFlag(cmd)

	return cmd
}
