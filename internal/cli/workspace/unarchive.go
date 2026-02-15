package workspace

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewUnarchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive <name>",
		Short: "Restore an archived workspace",
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

			if err := adapter.Unarchive(ctx, name); err != nil {
				return fmt.Errorf("failed to unarchive workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' restored\n",
				styles.Success.Render("Unarchived"), name)
			return nil
		},
	}

	return cmd
}
