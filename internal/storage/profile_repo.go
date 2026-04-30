// Package storage provides repository implementations backed by the
// existing aps file-based stores. ProfileRepo wraps internal/core's
// per-profile YAML directory layout in a hop.top/kit/go/runtime/domain
// Repository.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"hop.top/aps/internal/core"
	"hop.top/kit/go/runtime/domain"
)

// ProfileRepo is a domain.Repository[core.Profile] backed by the
// existing per-profile YAML files under APS_DATA_PATH/profiles/<id>/profile.yaml.
//
// It is a thin adapter over package-level functions in internal/core
// (LoadProfile, SaveProfile, etc.) — no behaviour change to the
// underlying store. The value of this wrapper is interface conformance
// so callers (and downstream Service[T] composition) can speak the
// generic Repository vocabulary against profiles.
type ProfileRepo struct{}

// NewProfileRepo constructs a ProfileRepo. The repo is stateless; all
// state lives on disk under APS_DATA_PATH.
func NewProfileRepo() *ProfileRepo { return &ProfileRepo{} }

// Create persists a new profile. Returns domain.ErrConflict if a
// profile with the given ID already exists.
func (r *ProfileRepo) Create(_ context.Context, p *core.Profile) error {
	if p == nil {
		return fmt.Errorf("profile is nil")
	}
	if p.ID == "" {
		return fmt.Errorf("profile.ID is empty")
	}
	if _, err := core.LoadProfile(p.ID); err == nil {
		return fmt.Errorf("%w: profile %q already exists", domain.ErrConflict, p.ID)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("preflight load: %w", err)
	}
	if err := core.CreateProfile(p.ID, *p); err != nil {
		// CreateProfile bubbles "profile X already exists" too — translate
		// to ErrConflict for callers using errors.Is.
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("%w: %v", domain.ErrConflict, err)
		}
		return err
	}
	return nil
}

// Get retrieves a profile by ID. Returns domain.ErrNotFound if absent.
func (r *ProfileRepo) Get(_ context.Context, id string) (*core.Profile, error) {
	p, err := core.LoadProfile(id)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: profile %q", domain.ErrNotFound, id)
		}
		return nil, err
	}
	return p, nil
}

// List returns profiles matching the query. Supports Limit, Offset,
// Sort (by id, default), and Search (substring match against ID and
// DisplayName).
func (r *ProfileRepo) List(ctx context.Context, q domain.Query) ([]core.Profile, error) {
	ids, err := core.ListProfiles()
	if err != nil {
		return nil, err
	}

	out := make([]core.Profile, 0, len(ids))
	for _, id := range ids {
		p, err := r.Get(ctx, id)
		if err != nil {
			// Skip profiles that fail to load — they are surfaced via
			// the dedicated profile-doctor command, not List.
			continue
		}
		if q.Search != "" && !matchesProfileSearch(p, q.Search) {
			continue
		}
		out = append(out, *p)
	}

	switch q.Sort {
	case "", "id":
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	case "display_name":
		sort.Slice(out, func(i, j int) bool { return out[i].DisplayName < out[j].DisplayName })
	}

	if q.Offset > 0 {
		if q.Offset >= len(out) {
			return nil, nil
		}
		out = out[q.Offset:]
	}
	if q.Limit > 0 && q.Limit < len(out) {
		out = out[:q.Limit]
	}
	return out, nil
}

// Update replaces an existing profile. Returns domain.ErrNotFound if
// the profile does not exist.
func (r *ProfileRepo) Update(ctx context.Context, p *core.Profile) error {
	if p == nil {
		return fmt.Errorf("profile is nil")
	}
	if _, err := r.Get(ctx, p.ID); err != nil {
		return err
	}
	return core.SaveProfile(p)
}

// Delete removes a profile. Returns domain.ErrNotFound if absent.
// Forces deletion (active sessions are not a blocker at the repo
// layer; CLI callers wanting the safety check should call
// core.DeleteProfile directly).
func (r *ProfileRepo) Delete(ctx context.Context, id string) error {
	if _, err := r.Get(ctx, id); err != nil {
		return err
	}
	return core.DeleteProfile(id, true)
}

// matchesProfileSearch returns true if the search string is a
// case-insensitive substring of the profile ID or DisplayName.
func matchesProfileSearch(p *core.Profile, search string) bool {
	s := strings.ToLower(search)
	return strings.Contains(strings.ToLower(p.ID), s) ||
		strings.Contains(strings.ToLower(p.DisplayName), s)
}
