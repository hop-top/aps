package collab

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewShowCmd creates the "collab show" command.
func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [workspace]",
		Short: "Show workspace details",
		Long:  `Display detailed information about a collaboration workspace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			ws, err := mgr.Get(cmd.Context(), wsID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(ws)
			}

			fmt.Printf("Name:     %s\n", ws.Config.Name)
			fmt.Printf("State:    %s\n", ws.State)
			fmt.Printf("Owner:    %s\n", ws.Config.OwnerProfileID)
			fmt.Printf("Agents:   %d\n", len(ws.Agents))
			fmt.Printf("Policy:   %s\n", ws.Policy.Default)
			fmt.Printf("Created:  %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
