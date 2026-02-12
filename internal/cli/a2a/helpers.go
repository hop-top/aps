package a2a

import (
	"fmt"

	"oss-aps-cli/internal/core"
)

// loadProfile loads a profile by ID
func loadProfile(profileID string) (*core.Profile, error) {
	if profileID == "" {
		return nil, fmt.Errorf("profile ID is required")
	}

	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	if !core.ProfileHasCapability(profile, "a2a") {
		return nil, fmt.Errorf("A2A is not enabled for profile %s", profileID)
	}

	return profile, nil
}
