package collab

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewMembersCmd creates the "collab members" command.
func NewMembersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members [workspace]",
		Short: "List workspace members",
		Long:  `List all agents that are members of a collaboration workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			members, err := mgr.Members(cmd.Context(), wsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(members)
			}

			if len(members) == 0 {
				fmt.Println("No members in workspace.")
				return nil
			}

			w := newTabWriter()
			fmt.Fprintf(w, "AGENT\tROLE\tSTATUS\tLAST SEEN\n")
			for _, m := range members {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					m.ProfileID,
					m.Role,
					m.Status,
					m.LastSeen.Format("2006-01-02 15:04:05"),
				)
			}
			w.Flush()

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
