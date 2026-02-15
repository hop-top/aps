package collab

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewLeaveCmd creates the "collab leave" command.
func NewLeaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leave [workspace]",
		Short: "Leave a collaboration workspace",
		Long: `Leave a collaboration workspace. Use --force to skip confirmation.

If no workspace is specified, the active workspace is used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				fmt.Printf("Leaving workspace '%s'...\n", wsID)
				fmt.Println()
				fmt.Println("  This will:")
				fmt.Println("    - Remove you from the workspace")
				fmt.Println("    - Revoke access to shared context")
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

			if err := mgr.Leave(cmd.Context(), wsID, profile); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"profile":   profile,
					"status":    "left",
				})
			}

			fmt.Printf("Left workspace '%s'\n", wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addProfileFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
