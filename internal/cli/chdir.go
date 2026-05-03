// Package cli — chdir.go resolves -C/--chdir targets that aren't literal
// directories. Wired into kit's PersistentPreRunE chain via
// kitcli.Config.ChdirResolver in root.go.
//
// Resolution order (first match wins):
//  1. Workspace directory (multidevice.GetWorkspaceDir)
//  2. Profile directory  (core.GetProfileDir)
//
// Returning ("", err) lets kit's resolveChdir fall through with the
// "not a directory" error citing the original target. T-0392.
package cli

import (
	"fmt"
	"os"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/multidevice"
)

// resolveAPSContext maps a -C/--chdir target to a concrete directory by
// trying the aps-specific contexts (workspace, profile) before letting
// kit fall back to literal-path semantics.
func resolveAPSContext(target string) (string, error) {
	if dir, err := multidevice.GetWorkspaceDir(target); err == nil {
		if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
			return dir, nil
		}
	}
	if dir, err := core.GetProfileDir(target); err == nil {
		if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
			return dir, nil
		}
	}
	return "", fmt.Errorf("aps: %q is not a workspace, profile, or directory", target)
}
