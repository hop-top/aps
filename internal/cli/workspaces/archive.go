package workspaces

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <name>",
		Short: "Archive a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			dataDir, _ := workspaceDataDir("global", "")
			adapter, err := ws.NewAdapter(dataDir)
			if err != nil {
				return fmt.Errorf("failed to initialize workspace backend: %w", err)
			}
			defer adapter.Close()

			if err := adapter.Archive(ctx, name); err != nil {
				return fmt.Errorf("failed to archive workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' archived\n",
				styles.Success.Render("Archived"), name)
			return nil
		},
	}

	return cmd
}
