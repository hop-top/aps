package collab

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	collab "oss-aps-cli/internal/core/collaboration"
	"oss-aps-cli/internal/storage"

	"github.com/spf13/cobra"
)

// resolveWorkspace determines the workspace ID from flag or active context.
func resolveWorkspace(cmd *cobra.Command, args []string) (string, error) {
	ws, _ := cmd.Flags().GetString("workspace")
	if ws != "" {
		return ws, nil
	}

	if len(args) > 0 {
		return args[0], nil
	}

	// Try active workspace
	store, err := getStorage()
	if err != nil {
		return "", err
	}
	mgr := collab.NewManager(store)
	active, err := mgr.GetActiveWorkspace(cmd.Context())
	if err != nil || active == "" {
		return "", fmt.Errorf("no workspace specified and no active workspace set\n\n  Set one: aps collab use <workspace>")
	}
	return active, nil
}

// resolveProfile determines the profile ID from flag or environment.
func resolveProfile(cmd *cobra.Command) (string, error) {
	p, _ := cmd.Flags().GetString("profile")
	if p != "" {
		return p, nil
	}
	p = os.Getenv("APS_PROFILE")
	if p != "" {
		return p, nil
	}
	return "", fmt.Errorf("no profile specified\n\n  Use: --profile <name> or set APS_PROFILE")
}

// getStorage creates a CollaborationStorage with the default root.
func getStorage() (*storage.CollaborationStorage, error) {
	return storage.NewCollaborationStorage("")
}

// getManager creates a Manager with default storage.
func getManager() (*collab.Manager, error) {
	store, err := getStorage()
	if err != nil {
		return nil, err
	}
	return collab.NewManager(store), nil
}

// outputJSON prints v as indented JSON.
func outputJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// isJSON returns true if --json flag is set.
func isJSON(cmd *cobra.Command) bool {
	j, _ := cmd.Flags().GetBool("json")
	return j
}

// newTabWriter creates a tabwriter for aligned table output.
func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// addWorkspaceFlag adds the -w/--workspace flag to a command.
func addWorkspaceFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("workspace", "w", "", "Workspace ID (uses active workspace if not set)")
}

// addProfileFlag adds the -p/--profile flag to a command.
func addProfileFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("profile", "p", "", "Profile ID (uses APS_PROFILE if not set)")
}

// addJSONFlag adds the --json flag to a command.
func addJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "Output as JSON")
}

// addForceFlag adds the --force flag to a command.
func addForceFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation")
}

// addLimitFlag adds the --limit flag.
func addLimitFlag(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 25, "Maximum results")
}
