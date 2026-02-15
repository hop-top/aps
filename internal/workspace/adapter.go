package workspace

import (
	"context"
	"fmt"
	"path/filepath"

	"hop.top/wsm/pkg/backend"
	"hop.top/wsm/pkg/backend/sqlite"
	"hop.top/wsm/pkg/model"
	"hop.top/wsm/pkg/workspace"

	"oss-aps-cli/internal/core"
)

// Workspace is the APS view of a wsm workspace.
type Workspace struct {
	ID     string
	Name   string
	Status string
}

// CreateOptions for creating a workspace.
type CreateOptions struct {
	Description string
	Tags        []string
}

// ListOptions for listing workspaces.
type ListOptions struct {
	Status string // "active", "archived", or "" for all
}

// Adapter wraps wsm Manager for APS use.
type Adapter struct {
	manager *workspace.Manager
	backend backend.Backend
}

// NewAdapter creates a workspace adapter backed by SQLite at the given directory.
func NewAdapter(dataDir string) (*Adapter, error) {
	dbPath := filepath.Join(dataDir, "wsm.db")
	be, err := sqlite.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open workspace database: %w", err)
	}

	if err := be.Migrate(context.Background()); err != nil {
		be.Close()
		return nil, fmt.Errorf("failed to migrate workspace database: %w", err)
	}

	mgr, err := workspace.NewManager(be)
	if err != nil {
		be.Close()
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	return &Adapter{manager: mgr, backend: be}, nil
}

// Create creates a new workspace.
func (a *Adapter) Create(ctx context.Context, name string, opts CreateOptions) (*Workspace, error) {
	ws, err := a.manager.Create(ctx, name, workspace.CreateOptions{
		Description: opts.Description,
		Tags:        opts.Tags,
	})
	if err != nil {
		return nil, translateError(err)
	}
	return toWorkspace(ws), nil
}

// Get retrieves a workspace by name or ID.
func (a *Adapter) Get(ctx context.Context, ref string) (*Workspace, error) {
	ws, err := a.manager.ResolveWorkspaceRef(ctx, ref)
	if err != nil {
		return nil, translateError(err)
	}
	return toWorkspace(ws), nil
}

// List returns workspaces matching the given options.
func (a *Adapter) List(ctx context.Context, opts ListOptions) ([]Workspace, error) {
	listOpts := backend.ListOptions{}
	if opts.Status != "" {
		s := model.Status(opts.Status)
		listOpts.Status = &s
	}
	wsList, err := a.manager.List(ctx, listOpts)
	if err != nil {
		return nil, translateError(err)
	}
	result := make([]Workspace, len(wsList))
	for i, ws := range wsList {
		result[i] = *toWorkspace(&ws)
	}
	return result, nil
}

// Archive archives a workspace.
func (a *Adapter) Archive(ctx context.Context, ref string) error {
	return translateError(a.manager.Archive(ctx, ref))
}

// Delete permanently deletes a workspace.
func (a *Adapter) Delete(ctx context.Context, ref string) error {
	return translateError(a.manager.Delete(ctx, ref))
}

// Close releases adapter resources.
func (a *Adapter) Close() error {
	return a.backend.Close()
}

func toWorkspace(ws *workspace.Workspace) *Workspace {
	if ws == nil {
		return nil
	}
	return &Workspace{
		ID:     ws.ID,
		Name:   ws.Name,
		Status: string(ws.Status),
	}
}

// translateError converts wsm errors to APS error types.
func translateError(err error) error {
	if err == nil {
		return nil
	}
	if workspace.IsKind(err, workspace.ErrorKindNotFound) {
		return core.NewNotFoundError(err.Error())
	}
	if workspace.IsKind(err, workspace.ErrorKindConflict) {
		return core.NewInvalidInputError("workspace", err.Error())
	}
	if workspace.IsKind(err, workspace.ErrorKindUsage) {
		return core.NewInvalidInputError("workspace", err.Error())
	}
	return err
}
