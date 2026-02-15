package collab

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewArchiveCmd creates the "collab archive" command.
func NewArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive [workspace]",
		Short: "Archive a collaboration workspace",
		Long: `Archive a collaboration workspace. Archived workspaces are read-only
and cannot accept new agents or tasks. Use --force to skip confirmation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			wsID, err := resolveWorkspace(cmd, args)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			if !force {
				fmt.Printf("Archiving workspace '%s'...\n", wsID)
				fmt.Println()
				fmt.Println("  This will:")
				fmt.Println("    - Set workspace to read-only")
				fmt.Println("    - Prevent new agents from joining")
				fmt.Println("    - Cancel any pending tasks")
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

			if err := mgr.Archive(cmd.Context(), wsID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				return err
			}

			if isJSON(cmd) {
				return outputJSON(map[string]string{
					"workspace": wsID,
					"status":    "archived",
				})
			}

			fmt.Printf("Archived workspace '%s'\n", wsID)

			return nil
		},
	}

	addWorkspaceFlag(cmd)
	addForceFlag(cmd)
	addJSONFlag(cmd)

	return cmd
}
