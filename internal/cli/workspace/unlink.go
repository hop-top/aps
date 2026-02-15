package workspace

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewUnlinkCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "unlink <profile>",
		Short: "Unlink a profile from its workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID := args[0]

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("profile '%s' not found: %w", profileID, err)
			}

			if profile.Workspace == nil {
				fmt.Printf("Profile '%s' is not linked to any workspace.\n", profileID)
				return nil
			}

			wsName := profile.Workspace.Name

			if !force {
				fmt.Printf("Unlinking profile '%s' from workspace '%s'...\n", profileID, wsName)
				fmt.Println()
				fmt.Println("  This will:")
				fmt.Println("    - Remove workspace context from profile")
				fmt.Println("    - Active sessions will lose workspace access")
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

			if err := ws.UnlinkProfile(profileID); err != nil {
				return fmt.Errorf("failed to unlink: %w", err)
			}

			fmt.Printf("%s Unlinked '%s' from '%s'\n",
				styles.Success.Render("Unlinked"), profileID, wsName)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				profiles, _ := core.ListProfiles()
				return profiles, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}
