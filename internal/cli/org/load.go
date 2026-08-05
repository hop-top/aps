package org

import (
	"fmt"

	"hop.top/aps/internal/core"
)

// loadFailure records a profile directory whose profile.yaml exists
// but failed to load (bad YAML, id mismatch, invalid isolation, …).
type loadFailure struct {
	ID  string
	Err error
}

// loadAllProfiles loads every profile directory that ListProfiles
// reports, surfacing per-profile load errors instead of silently
// skipping them the way core.ListProfilesFull does. `aps org check`
// turns each failure into a finding; the other org leaves ignore
// failures (an unloadable profile cannot participate in the graph).
func loadAllProfiles() ([]core.Profile, []loadFailure, error) {
	ids, err := core.ListProfiles()
	if err != nil {
		return nil, nil, fmt.Errorf("listing profiles: %w", err)
	}
	profiles := make([]core.Profile, 0, len(ids))
	var failures []loadFailure
	for _, id := range ids {
		p, loadErr := core.LoadProfile(id)
		if loadErr != nil {
			failures = append(failures, loadFailure{ID: id, Err: loadErr})
			continue
		}
		profiles = append(profiles, *p)
	}
	return profiles, failures, nil
}
