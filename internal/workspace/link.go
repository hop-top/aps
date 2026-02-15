package workspace

import (
	"fmt"

	"oss-aps-cli/internal/core"
)

// LinkProfile associates a profile with a workspace.
func LinkProfile(profileID, workspaceName, scope string) error {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	profile.Workspace = &core.WorkspaceLink{
		Name:  workspaceName,
		Scope: scope,
	}

	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile %s: %w", profileID, err)
	}

	return nil
}

// UnlinkProfile removes workspace association from a profile.
func UnlinkProfile(profileID string) error {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	profile.Workspace = nil

	if err := core.SaveProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile %s: %w", profileID, err)
	}

	return nil
}

// GetLinkedWorkspace returns the workspace link for a profile, or nil.
func GetLinkedWorkspace(profileID string) (*core.WorkspaceLink, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	return profile.Workspace, nil
}
