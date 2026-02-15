package workspace

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewLinkCmd() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "link <profile> <workspace>",
		Short: "Link a profile to a workspace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID, workspaceName := args[0], args[1]

			// Validate profile exists
			_, err := core.LoadProfile(profileID)
			if err != nil {
				fmt.Printf("Error: profile '%s' not found\n", profileID)
				fmt.Println()
				profiles, _ := core.ListProfiles()
				if len(profiles) > 0 {
					fmt.Println("  Available profiles:")
					for _, p := range profiles {
						prof, _ := core.LoadProfile(p)
						name := p
						if prof != nil && prof.DisplayName != "" {
							name = prof.DisplayName
						}
						fmt.Printf("    %-16s %s\n", p, styles.Dim.Render(name))
					}
					fmt.Println()
				}
				fmt.Printf("  To create a new profile:\n    aps profile new %s\n", profileID)
				return err
			}

			// Validate workspace exists
			ctx := context.Background()
			_, _, err = resolveWorkspace(ctx, workspaceName)
			if err != nil {
				fmt.Printf("Error: workspace '%s' not found\n", workspaceName)
				fmt.Println()
				fmt.Println("  To create a new workspace:")
				fmt.Printf("    aps workspace new %s\n", workspaceName)
				return err
			}

			// Link
			if err := ws.LinkProfile(profileID, workspaceName, scope); err != nil {
				return fmt.Errorf("failed to link: %w", err)
			}

			fmt.Printf("%s Linked '%s' to '%s' (%s)\n",
				styles.Success.Render("Linked"), profileID, workspaceName, scope)
			fmt.Printf("\n  View workspace:\n    aps workspace show %s\n", workspaceName)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				profiles, _ := core.ListProfiles()
				return profiles, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				return completeWorkspaceNames(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "global", "Scope of the link: global or profile")

	return cmd
}

func completeWorkspaceNames() []string {
	ctx := context.Background()
	var names []string

	dataDir, _ := workspaceDataDir("global", "")
	adapter, err := ws.NewAdapter(dataDir)
	if err == nil {
		wsList, _ := adapter.List(ctx, ws.ListOptions{})
		for _, w := range wsList {
			names = append(names, w.Name)
		}
		adapter.Close()
	}

	return names
}
