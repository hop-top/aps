package collaboration

import (
	"fmt"
	"strings"
)

// Registry provides workspace search and listing capabilities.
type Registry struct {
	storage Storage
}

// NewRegistry creates a Registry backed by the given storage.
func NewRegistry(storage Storage) *Registry {
	return &Registry{storage: storage}
}

// ListWorkspaces returns all workspaces, optionally filtered by name (exact match)
// and status via opts.Filters.
func (r *Registry) ListWorkspaces(opts ListOptions) ([]*Workspace, error) {
	ids, err := r.storage.ListWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	var workspaces []*Workspace
	for _, id := range ids {
		ws, err := r.storage.LoadWorkspace(id)
		if err != nil {
			continue
		}
		if !matchesFilters(ws, opts) {
			continue
		}
		workspaces = append(workspaces, ws)
	}

	// Apply pagination
	if opts.Offset > 0 && opts.Offset < len(workspaces) {
		workspaces = workspaces[opts.Offset:]
	} else if opts.Offset >= len(workspaces) {
		return []*Workspace{}, nil
	}
	if opts.Limit > 0 && opts.Limit < len(workspaces) {
		workspaces = workspaces[:opts.Limit]
	}

	return workspaces, nil
}

// SearchWorkspaces returns workspaces whose name contains the query substring
// (case-insensitive).
func (r *Registry) SearchWorkspaces(query string) ([]*Workspace, error) {
	ids, err := r.storage.ListWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	lower := strings.ToLower(query)
	var results []*Workspace
	for _, id := range ids {
		ws, err := r.storage.LoadWorkspace(id)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(ws.Config.Name), lower) {
			results = append(results, ws)
		}
	}

	return results, nil
}
