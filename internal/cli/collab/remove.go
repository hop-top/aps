package collab

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewRemoveCmd creates the "collab remove" command.
func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <agent> [workspace]",
		Short: "Remove an agent from the workspace",
		Long: `Remove an agent from a collaboration workspace. Only the workspace owner
can perform this action. Use --force to skip confirmation.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetAgent := args[0]

			wsID, err := resolveWorkspace(cmd, args[1:])
			if err != nil {
				return err
			}

			actor, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				fmt.Printf("Removing agent '%s' from workspace '%s'...\n", targetAgent, wsID)
				fmt.Println()
				fmt.Println("  This will:")
				fmt.Printf("    - Remove '%s' from the workspace\n", targetAgent)
				fmt.Println("    - Revoke their access to shared context")
				fmt.Println("    - Cancel any pending tasks assigned to them")
				fmt.Println()
				fmt.Print("  Proceed? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			if err := mgr.Remove(cmd.Context(), wsID, targetAgent, actor); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"agent":     targetAgent,
					"actor":     actor,
					"status":    "removed",
				})
			}

			fmt.Printf("Removed '%s' from workspace '%s'\n", targetAgent, wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
