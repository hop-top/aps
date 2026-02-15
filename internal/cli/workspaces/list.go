package workspaces

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewListCmd() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var all []workspaceRow

			// Collect from global scope
			if scope == "all" || scope == "global" {
				rows, err := listFromScope(ctx, "global", "")
				if err != nil {
					return err
				}
				all = append(all, rows...)
			}

			// Collect from profile scope (all profiles)
			if scope == "all" || scope == "profile" {
				profiles, err := core.ListProfiles()
				if err != nil {
					return fmt.Errorf("failed to list profiles: %w", err)
				}
				for _, pid := range profiles {
					rows, err := listFromScope(ctx, "profile", pid)
					if err != nil {
						continue // skip inaccessible profile workspaces
					}
					all = append(all, rows...)
				}
			}

			if len(all) == 0 {
				fmt.Println(styles.Dim.Render("No workspaces yet."))
				fmt.Println()
				fmt.Println("  Create your first workspace:")
				fmt.Printf("    aps workspaces new my-project\n")
				return nil
			}

			// Count linked profiles per workspace
			profileCounts := countLinkedProfiles()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, styles.Bold.Render("Workspaces"))
			fmt.Fprintln(w)
			fmt.Fprintln(w, "NAME\tSCOPE\tPROFILES\tSTATUS")
			globalCount, profileCount := 0, 0
			for _, row := range all {
				count := profileCounts[row.name]
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
					row.name, scopeBadge(row.scope), count, row.status)
				if row.scope == "global" {
					globalCount++
				} else {
					profileCount++
				}
			}
			w.Flush()

			fmt.Printf("\n%d workspaces", len(all))
			if globalCount > 0 && profileCount > 0 {
				fmt.Printf(" (%d global, %d profile-scoped)", globalCount, profileCount)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "all", "Scope filter: all, global, or profile")

	return cmd
}

type workspaceRow struct {
	name   string
	scope  string
	status string
}

func listFromScope(ctx context.Context, scope, profileID string) ([]workspaceRow, error) {
	dataDir, err := workspaceDataDir(scope, profileID)
	if err != nil {
		return nil, err
	}

	// Check if directory exists before trying to open DB
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, nil
	}

	adapter, err := ws.NewAdapter(dataDir)
	if err != nil {
		return nil, err
	}
	defer adapter.Close()

	wsList, err := adapter.List(ctx, ws.ListOptions{})
	if err != nil {
		return nil, err
	}

	rows := make([]workspaceRow, len(wsList))
	for i, w := range wsList {
		rows[i] = workspaceRow{
			name:   w.Name,
			scope:  scope,
			status: w.Status,
		}
	}
	return rows, nil
}

func countLinkedProfiles() map[string]int {
	counts := make(map[string]int)
	profiles, err := core.ListProfiles()
	if err != nil {
		return counts
	}
	for _, pid := range profiles {
		profile, err := core.LoadProfile(pid)
		if err != nil {
			continue
		}
		if profile.Workspace != nil {
			counts[profile.Workspace.Name]++
		}
	}
	return counts
}

func scopeBadge(scope string) string {
	switch scope {
	case "global":
		return styles.KindBadge("builtin") // blue
	case "profile":
		return styles.TypeBadge("managed") // teal
	default:
		return scope
	}
}
