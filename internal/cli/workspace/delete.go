package workspace

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewDeleteCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a workspace",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			// Find linked profiles
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

			if len(linked) > 0 && !force {
				fmt.Printf("%s '%s' is linked to %d profiles:\n",
					styles.Warn.Render("Warning:"), name, len(linked))
				for _, pid := range linked {
					fmt.Printf("  %s\n", pid)
				}
				fmt.Println()
				fmt.Println("  Use --force to delete and unlink all profiles.")
				return fmt.Errorf("workspace has linked profiles")
			}

			if dryRun {
				fmt.Printf("Would delete workspace '%s'\n", name)
				if len(linked) > 0 {
					fmt.Printf("Would unlink %d profiles: %v\n", len(linked), linked)
				}
				return nil
			}

			// Unlink all profiles
			for _, pid := range linked {
				if err := ws.UnlinkProfile(pid); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to unlink %s: %v\n", pid, err)
				}
			}

			// Delete workspace
			dataDir, _ := workspaceDataDir("global", "")
			adapter, err := ws.NewAdapter(dataDir)
			if err != nil {
				return fmt.Errorf("failed to initialize workspace backend: %w", err)
			}
			defer adapter.Close()

			if err := adapter.Delete(ctx, name); err != nil {
				return fmt.Errorf("failed to delete workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' deleted\n",
				styles.Success.Render("Deleted"), name)
			if len(linked) > 0 {
				fmt.Printf("Unlinked %d profiles\n", len(linked))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force delete even if profiles are linked")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without executing")

	return cmd
}
