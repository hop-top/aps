package workspace

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/storage"
	"hop.top/aps/internal/styles"
)

var (
	// Shared styles used by merged conflict/audit subcommands.
	// T-0456 — collabTableHeader / tableHeader / newTabWriter were
	// removed when the workspace tables migrated to listing.RenderList
	// (kit-themed styled output via output.WithTableStyle on TTY).
	headerStyle  = styles.Title
	dimStyle     = styles.Dim
	boldStyle    = styles.Bold
	successStyle = styles.Success
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
		return "", fmt.Errorf("no workspace specified and no active workspace set\n\n  Set one: aps workspace use <workspace>")
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

// addWorkspaceFlag is a no-op. --workspace is a persistent global flag
// declared in cli.Config.Globals (T-0376). Subcommands read it via
// cmd.Flags().GetString("workspace") which falls through to the
// persistent flag set.
func addWorkspaceFlag(cmd *cobra.Command) {}

// addProfileFlag is a no-op. --profile is a persistent global flag
// declared in cli.Config.Globals (T-0376). Subcommands read it via
// cmd.Flags().GetString("profile") which falls through to the
// persistent flag set.
func addProfileFlag(cmd *cobra.Command) {}

// addJSONFlag adds the --json flag to a command.
func addJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "Output as JSON")
}

// addForceFlag adds the --force flag to a command.
func addForceFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("force", false, "Skip confirmation")
}

// addLimitFlag adds the --limit flag.
func addLimitFlag(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 25, "Maximum results")
}
