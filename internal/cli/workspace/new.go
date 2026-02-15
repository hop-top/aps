package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewNewCmd() *cobra.Command {
	var (
		scope  string
		noLink bool
	)

	cmd := &cobra.Command{
		Use:     "new <name>",
		Aliases: []string{"create"},
		Short:   "Create a new workspace",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			dataDir, err := workspaceDataDir(scope, "")
			if err != nil {
				return err
			}

			adapter, err := ws.NewAdapter(dataDir)
			if err != nil {
				return fmt.Errorf("failed to initialize workspace backend: %w", err)
			}
			defer adapter.Close()

			ctx := context.Background()
			_, err = adapter.Create(ctx, name, ws.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' created (scope: %s)\n",
				styles.Success.Render("Created"), name, scope)

			// Auto-link if APS_PROFILE is set and --no-link not passed
			if !noLink {
				profileID := os.Getenv("APS_PROFILE")
				if profileID != "" {
					if err := ws.LinkProfile(profileID, name, scope); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to auto-link profile: %v\n", err)
					} else {
						fmt.Printf("Linked to active profile '%s'\n", profileID)
					}
				}
			}

			fmt.Printf("\n  View workspace:\n    aps workspace show %s\n", name)

			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "global", "Scope: global or profile")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Skip auto-linking to active profile")

	return cmd
}

// workspaceDataDir returns the data directory for workspace storage.
func workspaceDataDir(scope, profileID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if scope == "profile" && profileID != "" {
		return filepath.Join(home, ".aps", "profiles", profileID, "workspaces"), nil
	}

	return filepath.Join(home, ".aps", "workspaces"), nil
}
