package workspaces

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <name>",
		Aliases: []string{"inspect"},
		Short:   "Show workspace details",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			// Try global first, then profile scopes
			workspace, scope, err := resolveWorkspace(ctx, name)
			if err != nil {
				fmt.Printf("Error: workspace '%s' not found\n", name)
				fmt.Println()
				fmt.Println("  Run 'aps workspaces list' to see available workspaces.")
				return err
			}

			fmt.Println(styles.Title.Render(workspace.Name))
			fmt.Println()
			fmt.Printf("%-15s %s\n", styles.Bold.Render("Scope:"), scope)
			fmt.Printf("%-15s %s\n", styles.Bold.Render("Status:"), workspace.Status)

			// Linked profiles
			profiles, _ := core.ListProfiles()
			var linked []string
			for _, pid := range profiles {
				p, err := core.LoadProfile(pid)
				if err != nil {
					continue
				}
				if p.Workspace != nil && p.Workspace.Name == name {
					linked = append(linked, pid)
				}
			}

			if len(linked) > 0 {
				fmt.Println()
				fmt.Println(styles.Bold.Render("Linked Profiles:"))
				for _, pid := range linked {
					p, _ := core.LoadProfile(pid)
					displayName := pid
					if p != nil && p.DisplayName != "" {
						displayName = p.DisplayName
					}
					fmt.Printf("  %-16s %s\n", pid, styles.Dim.Render(displayName))
				}
			}

			fmt.Printf("\n%d profiles linked\n", len(linked))

			return nil
		},
	}

	return cmd
}

// resolveWorkspace tries to find a workspace across scopes.
func resolveWorkspace(ctx context.Context, name string) (*ws.Workspace, string, error) {
	// Try global
	dataDir, _ := workspaceDataDir("global", "")
	adapter, err := ws.NewAdapter(dataDir)
	if err == nil {
		w, err := adapter.Get(ctx, name)
		adapter.Close()
		if err == nil {
			return w, "global", nil
		}
	}

	// Try each profile scope
	profiles, _ := core.ListProfiles()
	for _, pid := range profiles {
		dataDir, _ = workspaceDataDir("profile", pid)
		adapter, err := ws.NewAdapter(dataDir)
		if err != nil {
			continue
		}
		w, err := adapter.Get(ctx, name)
		adapter.Close()
		if err == nil {
			return w, "profile", nil
		}
	}

	return nil, "", core.NewNotFoundError(fmt.Sprintf("workspace '%s'", name))
}
