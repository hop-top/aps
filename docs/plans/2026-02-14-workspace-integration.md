# APS-Workspace Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate workspace-cli (`hop.top/wsm`) into APS, extending profiles and sessions with workspace awareness.

**Architecture:** An adapter layer (`internal/workspace/`) wraps wsm's Manager API, translating wsm types/errors to APS conventions. CLI commands in `internal/cli/workspaces/` follow the existing Cobra command group pattern (see `internal/cli/webhook/`). Profile gets an optional `*WorkspaceLink` pointer field; SessionInfo gets an optional `WorkspaceID` string.

**Tech Stack:** Go, Cobra CLI, hop.top/wsm (SQLite-backed workspace manager), lipgloss styles, tabwriter

---

## Task 1: Add wsm dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the wsm module as a local replace directive**

```bash
# From the repo root
go mod edit -require hop.top/wsm@v0.0.0
go mod edit -replace hop.top/wsm=../wsm/hops/main
go mod tidy
```

Note: The wsm repo is at `~/.w/ideacrafterslabs/wsm/hops/main`. The replace directive uses a relative path `../wsm/hops/main` assuming sibling checkout layout.

**Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build with no errors

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add hop.top/wsm dependency for workspace integration"
```

---

## Task 2: Add WorkspaceLink to Profile and WorkspaceID to SessionInfo

**Files:**
- Modify: `internal/core/profile.go` (after line 38, before closing brace of Profile struct)
- Modify: `internal/core/session/registry.go` (after line 47, before closing brace of SessionInfo)
- Test: `internal/core/core_test.go`

**Step 1: Write the failing test**

Add to `internal/core/core_test.go`:

```go
func TestProfileWithWorkspaceLink(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	os.MkdirAll(filepath.Join(tmpDir, ".agents", "profiles", "ws-test"), 0755)

	profile := Profile{
		ID:          "ws-test",
		DisplayName: "Workspace Test",
		Workspace: &WorkspaceLink{
			Name:  "dev-project",
			Scope: "global",
		},
	}

	err := SaveProfile(&profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("ws-test")
	require.NoError(t, err)
	require.NotNil(t, loaded.Workspace)
	assert.Equal(t, "dev-project", loaded.Workspace.Name)
	assert.Equal(t, "global", loaded.Workspace.Scope)
}

func TestProfileWithoutWorkspaceLink(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	os.MkdirAll(filepath.Join(tmpDir, ".agents", "profiles", "no-ws-test"), 0755)

	profile := Profile{
		ID:          "no-ws-test",
		DisplayName: "No Workspace",
	}

	err := SaveProfile(&profile)
	require.NoError(t, err)

	loaded, err := LoadProfile("no-ws-test")
	require.NoError(t, err)
	assert.Nil(t, loaded.Workspace)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/core/ -run TestProfileWithWorkspaceLink -v`
Expected: FAIL — `WorkspaceLink` undefined

**Step 3: Add WorkspaceLink type and field to Profile**

In `internal/core/profile.go`, add the type before Profile struct:

```go
// WorkspaceLink associates a profile with a workspace
type WorkspaceLink struct {
	Name  string `yaml:"name"`
	Scope string `yaml:"scope"` // "global" or "profile"
}
```

Add the field to the `Profile` struct (after the `ACP` field):

```go
	Workspace *WorkspaceLink `yaml:"workspace,omitempty"`
```

**Step 4: Add WorkspaceID to SessionInfo**

In `internal/core/session/registry.go`, add after the `Environment` field in `SessionInfo`:

```go
	WorkspaceID string `json:"workspace_id,omitempty"`
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/core/ -run TestProfileWith -v`
Expected: PASS

Run: `go build ./...`
Expected: Clean build

**Step 6: Commit**

```bash
git add internal/core/profile.go internal/core/session/registry.go internal/core/core_test.go
git commit -m "feat(core): add WorkspaceLink to Profile and WorkspaceID to SessionInfo"
```

---

## Task 3: Create workspace adapter

**Files:**
- Create: `internal/workspace/adapter.go`
- Create: `internal/workspace/adapter_test.go`

**Step 1: Write the failing test**

Create `internal/workspace/adapter_test.go`:

```go
package workspace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	defer adapter.Close()
}

func TestAdapterCreateAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	ws, err := adapter.Create(ctx, "test-project", CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "test-project", ws.Name)
	assert.Equal(t, "active", ws.Status)

	got, err := adapter.Get(ctx, "test-project")
	require.NoError(t, err)
	assert.Equal(t, ws.Name, got.Name)
}

func TestAdapterList(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "project-a", CreateOptions{})
	require.NoError(t, err)
	_, err = adapter.Create(ctx, "project-b", CreateOptions{})
	require.NoError(t, err)

	list, err := adapter.List(ctx, ListOptions{})
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestAdapterArchive(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "archive-me", CreateOptions{})
	require.NoError(t, err)

	err = adapter.Archive(ctx, "archive-me")
	require.NoError(t, err)

	ws, err := adapter.Get(ctx, "archive-me")
	require.NoError(t, err)
	assert.Equal(t, "archived", ws.Status)
}

func TestAdapterDelete(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "delete-me", CreateOptions{})
	require.NoError(t, err)

	err = adapter.Delete(ctx, "delete-me")
	require.NoError(t, err)

	_, err = adapter.Get(ctx, "delete-me")
	assert.Error(t, err)
}

func TestAdapterGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Get(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestAdapterCreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewAdapter(tmpDir)
	require.NoError(t, err)
	defer adapter.Close()

	ctx := context.Background()
	_, err = adapter.Create(ctx, "dupe", CreateOptions{})
	require.NoError(t, err)

	_, err = adapter.Create(ctx, "dupe", CreateOptions{})
	assert.Error(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/workspace/ -v`
Expected: FAIL — package doesn't exist

**Step 3: Implement the adapter**

Create `internal/workspace/adapter.go`:

```go
package workspace

import (
	"context"
	"fmt"
	"path/filepath"

	"hop.top/wsm/pkg/backend"
	"hop.top/wsm/pkg/backend/sqlite"
	"hop.top/wsm/pkg/workspace"

	"oss-aps-cli/internal/core"
)

// Workspace is the APS view of a wsm workspace
type Workspace struct {
	ID     string
	Name   string
	Status string
}

// CreateOptions for creating a workspace
type CreateOptions struct {
	Description string
	Tags        []string
}

// ListOptions for listing workspaces
type ListOptions struct {
	Status string // "active", "archived", or "" for all
}

// Adapter wraps wsm Manager for APS use
type Adapter struct {
	manager *workspace.Manager
	backend backend.Backend
}

// NewAdapter creates a workspace adapter backed by SQLite at the given directory.
func NewAdapter(dataDir string) (*Adapter, error) {
	dbPath := filepath.Join(dataDir, "wsm.db")
	be, err := sqlite.New(dbPath)
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

// Create creates a new workspace
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

// Get retrieves a workspace by name or ID
func (a *Adapter) Get(ctx context.Context, ref string) (*Workspace, error) {
	ws, err := a.manager.ResolveWorkspaceRef(ctx, ref)
	if err != nil {
		return nil, translateError(err)
	}
	return toWorkspace(ws), nil
}

// List returns workspaces matching the given options
func (a *Adapter) List(ctx context.Context, opts ListOptions) ([]Workspace, error) {
	listOpts := backend.ListOptions{}
	if opts.Status != "" {
		listOpts.Status = opts.Status
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

// Archive archives a workspace
func (a *Adapter) Archive(ctx context.Context, ref string) error {
	return translateError(a.manager.Archive(ctx, ref))
}

// Delete permanently deletes a workspace
func (a *Adapter) Delete(ctx context.Context, ref string) error {
	return translateError(a.manager.Delete(ctx, ref))
}

// Close releases adapter resources
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

// translateError converts wsm errors to APS error types
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/workspace/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/workspace/adapter.go internal/workspace/adapter_test.go
git commit -m "feat(workspace): add adapter wrapping wsm Manager"
```

---

## Task 4: Create profile-workspace linking logic

**Files:**
- Create: `internal/workspace/link.go`
- Create: `internal/workspace/link_test.go`

**Step 1: Write the failing test**

Create `internal/workspace/link_test.go`:

```go
package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/core"
)

func TestLinkProfile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "link-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{ID: "link-test", DisplayName: "Link Test"}
	require.NoError(t, core.SaveProfile(profile))

	err := LinkProfile("link-test", "dev-project", "global")
	require.NoError(t, err)

	loaded, err := core.LoadProfile("link-test")
	require.NoError(t, err)
	require.NotNil(t, loaded.Workspace)
	assert.Equal(t, "dev-project", loaded.Workspace.Name)
	assert.Equal(t, "global", loaded.Workspace.Scope)
}

func TestUnlinkProfile(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "unlink-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{
		ID:          "unlink-test",
		DisplayName: "Unlink Test",
		Workspace: &core.WorkspaceLink{
			Name:  "dev-project",
			Scope: "global",
		},
	}
	require.NoError(t, core.SaveProfile(profile))

	err := UnlinkProfile("unlink-test")
	require.NoError(t, err)

	loaded, err := core.LoadProfile("unlink-test")
	require.NoError(t, err)
	assert.Nil(t, loaded.Workspace)
}

func TestGetLinkedWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "linked-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{
		ID:          "linked-test",
		DisplayName: "Linked Test",
		Workspace: &core.WorkspaceLink{
			Name:  "my-workspace",
			Scope: "global",
		},
	}
	require.NoError(t, core.SaveProfile(profile))

	link, err := GetLinkedWorkspace("linked-test")
	require.NoError(t, err)
	require.NotNil(t, link)
	assert.Equal(t, "my-workspace", link.Name)
}

func TestGetLinkedWorkspaceNone(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	profileDir := filepath.Join(tmpDir, ".agents", "profiles", "nolink-test")
	require.NoError(t, os.MkdirAll(profileDir, 0755))

	profile := &core.Profile{ID: "nolink-test", DisplayName: "No Link"}
	require.NoError(t, core.SaveProfile(profile))

	link, err := GetLinkedWorkspace("nolink-test")
	require.NoError(t, err)
	assert.Nil(t, link)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/workspace/ -run TestLinkProfile -v`
Expected: FAIL — `LinkProfile` undefined

**Step 3: Implement linking functions**

Create `internal/workspace/link.go`:

```go
package workspace

import (
	"fmt"

	"oss-aps-cli/internal/core"
)

// LinkProfile associates a profile with a workspace
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

// UnlinkProfile removes workspace association from a profile
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

// GetLinkedWorkspace returns the workspace link for a profile, or nil
func GetLinkedWorkspace(profileID string) (*core.WorkspaceLink, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	return profile.Workspace, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/workspace/ -run TestLink -v && go test ./internal/workspace/ -run TestUnlink -v && go test ./internal/workspace/ -run TestGetLinked -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/workspace/link.go internal/workspace/link_test.go
git commit -m "feat(workspace): add profile-workspace linking functions"
```

---

## Task 5: Create CLI command group and `workspaces new`

**Files:**
- Create: `internal/cli/workspaces/cmd.go`
- Create: `internal/cli/workspaces/new.go`
- Create: `internal/cli/workspaces.go`

**Step 1: Create the command group**

Create `internal/cli/workspaces/cmd.go`:

```go
package workspaces

import (
	"github.com/spf13/cobra"
)

// NewWorkspacesCmd creates the workspaces command group
func NewWorkspacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspaces",
		Aliases: []string{"ws"},
		Short:   "Manage workspaces",
		Long: `Manage workspaces for organizing agent work.

Workspaces provide logical groupings with configuration.
Profiles can be linked to workspaces for context awareness.`,
	}

	cmd.AddCommand(NewNewCmd())

	return cmd
}
```

**Step 2: Create the `new` command**

Create `internal/cli/workspaces/new.go`:

```go
package workspaces

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewNewCmd() *cobra.Command {
	var (
		scope  string
		noLink bool
	)

	cmd := &cobra.Command{
		Use:     "new <name>",
		Aliases: []string{"create"},
		Short:   "Create a new workspace",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			dataDir, err := workspaceDataDir(scope, "")
			if err != nil {
				return err
			}

			adapter, err := ws.NewAdapter(dataDir)
			if err != nil {
				return fmt.Errorf("failed to initialize workspace backend: %w", err)
			}
			defer adapter.Close()

			ctx := context.Background()
			_, err = adapter.Create(ctx, name, ws.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' created (scope: %s)\n",
				styles.Success.Render("Created"), name, scope)

			// Auto-link if APS_PROFILE is set and --no-link not passed
			if !noLink {
				profileID := os.Getenv("APS_PROFILE")
				if profileID != "" {
					if err := ws.LinkProfile(profileID, name, scope); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to auto-link profile: %v\n", err)
					} else {
						fmt.Printf("Linked to active profile '%s'\n", profileID)
					}
				}
			}

			fmt.Printf("\n  View workspace:\n    aps workspaces show %s\n", name)

			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "global", "Scope: global or profile")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Skip auto-linking to active profile")

	return cmd
}

// workspaceDataDir returns the data directory for workspace storage
func workspaceDataDir(scope, profileID string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if scope == "profile" && profileID != "" {
		return filepath.Join(home, ".aps", "profiles", profileID, "workspaces"), nil
	}

	return filepath.Join(home, ".aps", "workspaces"), nil
}
```

**Step 3: Register the command group**

Create `internal/cli/workspaces.go`:

```go
package cli

import (
	"oss-aps-cli/internal/cli/workspaces"
)

func init() {
	rootCmd.AddCommand(workspaces.NewWorkspacesCmd())
}
```

**Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

Run: `go run ./cmd/aps workspaces --help`
Expected: Shows workspaces command with `new` subcommand

**Step 5: Commit**

```bash
git add internal/cli/workspaces/cmd.go internal/cli/workspaces/new.go internal/cli/workspaces.go
git commit -m "feat(cli): add workspaces command group with 'new' command"
```

---

## Task 6: Add `workspaces list` command

**Files:**
- Create: `internal/cli/workspaces/list.go`
- Modify: `internal/cli/workspaces/cmd.go` (add `NewListCmd()`)

**Step 1: Create the list command**

Create `internal/cli/workspaces/list.go`:

```go
package workspaces

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewListCmd() *cobra.Command {
	var (
		scope    string
		sortBy   string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var all []workspaceRow

			// Collect from global scope
			if scope == "all" || scope == "global" {
				rows, err := listFromScope(ctx, "global", "")
				if err != nil {
					return err
				}
				all = append(all, rows...)
			}

			// Collect from profile scope (all profiles)
			if scope == "all" || scope == "profile" {
				profiles, err := core.ListProfiles()
				if err != nil {
					return fmt.Errorf("failed to list profiles: %w", err)
				}
				for _, pid := range profiles {
					rows, err := listFromScope(ctx, "profile", pid)
					if err != nil {
						continue // skip inaccessible profile workspaces
					}
					all = append(all, rows...)
				}
			}

			if len(all) == 0 {
				fmt.Println(styles.Dim.Render("No workspaces yet."))
				fmt.Println()
				fmt.Println("  Create your first workspace:")
				fmt.Printf("    aps workspaces new my-project\n")
				return nil
			}

			// Count linked profiles per workspace
			profileCounts := countLinkedProfiles()

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, styles.Bold.Render("Workspaces"))
			fmt.Fprintln(w)
			fmt.Fprintln(w, "NAME\tSCOPE\tPROFILES\tSTATUS")
			globalCount, profileCount := 0, 0
			for _, row := range all {
				count := profileCounts[row.name]
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
					row.name, scopeBadge(row.scope), count, row.status)
				if row.scope == "global" {
					globalCount++
				} else {
					profileCount++
				}
			}
			w.Flush()

			fmt.Printf("\n%d workspaces", len(all))
			if globalCount > 0 && profileCount > 0 {
				fmt.Printf(" (%d global, %d profile-scoped)", globalCount, profileCount)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "all", "Scope filter: all, global, or profile")
	cmd.Flags().StringVar(&sortBy, "sort", "activity", "Sort by: name, activity, or created")

	return cmd
}

type workspaceRow struct {
	name   string
	scope  string
	status string
}

func listFromScope(ctx context.Context, scope, profileID string) ([]workspaceRow, error) {
	dataDir, err := workspaceDataDir(scope, profileID)
	if err != nil {
		return nil, err
	}

	// Check if directory exists before trying to open DB
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, nil
	}

	adapter, err := ws.NewAdapter(dataDir)
	if err != nil {
		return nil, err
	}
	defer adapter.Close()

	wsList, err := adapter.List(ctx, ws.ListOptions{})
	if err != nil {
		return nil, err
	}

	rows := make([]workspaceRow, len(wsList))
	for i, w := range wsList {
		rows[i] = workspaceRow{
			name:   w.Name,
			scope:  scope,
			status: w.Status,
		}
	}
	return rows, nil
}

func countLinkedProfiles() map[string]int {
	counts := make(map[string]int)
	profiles, err := core.ListProfiles()
	if err != nil {
		return counts
	}
	for _, pid := range profiles {
		profile, err := core.LoadProfile(pid)
		if err != nil {
			continue
		}
		if profile.Workspace != nil {
			counts[profile.Workspace.Name]++
		}
	}
	return counts
}

func scopeBadge(scope string) string {
	switch scope {
	case "global":
		return styles.KindBadge("builtin") // blue
	case "profile":
		return styles.TypeBadge("managed") // teal
	default:
		return scope
	}
}
```

**Step 2: Register in cmd.go**

Add to `internal/cli/workspaces/cmd.go` in `NewWorkspacesCmd()`:

```go
	cmd.AddCommand(NewListCmd())
```

**Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 4: Commit**

```bash
git add internal/cli/workspaces/list.go internal/cli/workspaces/cmd.go
git commit -m "feat(cli): add 'workspaces list' command"
```

---

## Task 7: Add `workspaces show` command

**Files:**
- Create: `internal/cli/workspaces/show.go`
- Modify: `internal/cli/workspaces/cmd.go` (add `NewShowCmd()`)

**Step 1: Create the show command**

Create `internal/cli/workspaces/show.go`:

```go
package workspaces

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <name>",
		Aliases: []string{"inspect"},
		Short:   "Show workspace details",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			// Try global first, then profile scopes
			workspace, scope, err := resolveWorkspace(ctx, name)
			if err != nil {
				fmt.Printf("Error: workspace '%s' not found\n", name)
				fmt.Println()
				fmt.Println("  Run 'aps workspaces list' to see available workspaces.")
				return err
			}

			fmt.Println(styles.Title.Render(workspace.Name))
			fmt.Println()
			fmt.Printf("%-15s %s\n", styles.Bold.Render("Scope:"), scope)
			fmt.Printf("%-15s %s\n", styles.Bold.Render("Status:"), workspace.Status)

			// Linked profiles
			profiles, _ := core.ListProfiles()
			var linked []string
			for _, pid := range profiles {
				p, err := core.LoadProfile(pid)
				if err != nil {
					continue
				}
				if p.Workspace != nil && p.Workspace.Name == name {
					linked = append(linked, pid)
				}
			}

			if len(linked) > 0 {
				fmt.Println()
				fmt.Println(styles.Bold.Render("Linked Profiles:"))
				for _, pid := range linked {
					p, _ := core.LoadProfile(pid)
					displayName := pid
					if p != nil && p.DisplayName != "" {
						displayName = p.DisplayName
					}
					fmt.Printf("  %-16s %s\n", pid, styles.Dim.Render(displayName))
				}
			}

			fmt.Printf("\n%d profiles linked\n", len(linked))

			return nil
		},
	}

	return cmd
}

// resolveWorkspace tries to find a workspace across scopes
func resolveWorkspace(ctx context.Context, name string) (*ws.Workspace, string, error) {
	// Try global
	dataDir, _ := workspaceDataDir("global", "")
	adapter, err := ws.NewAdapter(dataDir)
	if err == nil {
		defer adapter.Close()
		w, err := adapter.Get(ctx, name)
		if err == nil {
			return w, "global", nil
		}
	}

	// Try each profile scope
	profiles, _ := core.ListProfiles()
	for _, pid := range profiles {
		dataDir, _ = workspaceDataDir("profile", pid)
		adapter, err := ws.NewAdapter(dataDir)
		if err != nil {
			continue
		}
		w, err := adapter.Get(ctx, name)
		adapter.Close()
		if err == nil {
			return w, "profile", nil
		}
	}

	return nil, "", core.NewNotFoundError(fmt.Sprintf("workspace '%s'", name))
}
```

**Step 2: Register in cmd.go**

Add `cmd.AddCommand(NewShowCmd())` in `NewWorkspacesCmd()`.

**Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 4: Commit**

```bash
git add internal/cli/workspaces/show.go internal/cli/workspaces/cmd.go
git commit -m "feat(cli): add 'workspaces show' command"
```

---

## Task 8: Add `workspaces link` and `workspaces unlink` commands

**Files:**
- Create: `internal/cli/workspaces/link.go`
- Create: `internal/cli/workspaces/unlink.go`
- Modify: `internal/cli/workspaces/cmd.go`

**Step 1: Create link command**

Create `internal/cli/workspaces/link.go`:

```go
package workspaces

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewLinkCmd() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "link <profile> <workspace>",
		Short: "Link a profile to a workspace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID, workspaceName := args[0], args[1]

			// Validate profile exists
			_, err := core.LoadProfile(profileID)
			if err != nil {
				fmt.Printf("Error: profile '%s' not found\n", profileID)
				fmt.Println()
				profiles, _ := core.ListProfiles()
				if len(profiles) > 0 {
					fmt.Println("  Available profiles:")
					for _, p := range profiles {
						prof, _ := core.LoadProfile(p)
						name := p
						if prof != nil && prof.DisplayName != "" {
							name = prof.DisplayName
						}
						fmt.Printf("    %-16s %s\n", p, styles.Dim.Render(name))
					}
					fmt.Println()
				}
				fmt.Printf("  To create a new profile:\n    aps profile new %s\n", profileID)
				return err
			}

			// Validate workspace exists
			ctx := context.Background()
			_, _, err = resolveWorkspace(ctx, workspaceName)
			if err != nil {
				fmt.Printf("Error: workspace '%s' not found\n", workspaceName)
				fmt.Println()
				fmt.Println("  To create a new workspace:")
				fmt.Printf("    aps workspaces new %s\n", workspaceName)
				return err
			}

			// Link
			if err := ws.LinkProfile(profileID, workspaceName, scope); err != nil {
				return fmt.Errorf("failed to link: %w", err)
			}

			fmt.Printf("%s Linked '%s' to '%s' (%s)\n",
				styles.Success.Render("Linked"), profileID, workspaceName, scope)
			fmt.Printf("\n  View workspace:\n    aps workspaces show %s\n", workspaceName)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// Complete profile IDs
				profiles, _ := core.ListProfiles()
				return profiles, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				// Complete workspace names
				return completeWorkspaceNames(toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "global", "Scope of the link: global or profile")

	return cmd
}

func completeWorkspaceNames(prefix string) []string {
	ctx := context.Background()
	var names []string

	dataDir, _ := workspaceDataDir("global", "")
	adapter, err := ws.NewAdapter(dataDir)
	if err == nil {
		wsList, _ := adapter.List(ctx, ws.ListOptions{})
		for _, w := range wsList {
			names = append(names, w.Name)
		}
		adapter.Close()
	}

	return names
}
```

**Step 2: Create unlink command**

Create `internal/cli/workspaces/unlink.go`:

```go
package workspaces

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewUnlinkCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "unlink <profile>",
		Short: "Unlink a profile from its workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileID := args[0]

			profile, err := core.LoadProfile(profileID)
			if err != nil {
				return fmt.Errorf("profile '%s' not found: %w", profileID, err)
			}

			if profile.Workspace == nil {
				fmt.Printf("Profile '%s' is not linked to any workspace.\n", profileID)
				return nil
			}

			wsName := profile.Workspace.Name

			if !force {
				fmt.Printf("Unlinking profile '%s' from workspace '%s'...\n", profileID, wsName)
				fmt.Println()
				fmt.Println("  This will:")
				fmt.Println("    - Remove workspace context from profile")
				fmt.Println("    - Active sessions will lose workspace access")
				fmt.Println()
				fmt.Print("  Proceed? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := ws.UnlinkProfile(profileID); err != nil {
				return fmt.Errorf("failed to unlink: %w", err)
			}

			fmt.Printf("%s Unlinked '%s' from '%s'\n",
				styles.Success.Render("Unlinked"), profileID, wsName)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				profiles, _ := core.ListProfiles()
				return profiles, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}
```

**Step 3: Register both in cmd.go**

Add to `NewWorkspacesCmd()`:

```go
	cmd.AddCommand(NewLinkCmd())
	cmd.AddCommand(NewUnlinkCmd())
```

**Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 5: Commit**

```bash
git add internal/cli/workspaces/link.go internal/cli/workspaces/unlink.go internal/cli/workspaces/cmd.go
git commit -m "feat(cli): add 'workspaces link' and 'workspaces unlink' commands"
```

---

## Task 9: Add `workspaces archive` and `workspaces delete` commands

**Files:**
- Create: `internal/cli/workspaces/archive.go`
- Create: `internal/cli/workspaces/delete.go`
- Modify: `internal/cli/workspaces/cmd.go`

**Step 1: Create archive command**

Create `internal/cli/workspaces/archive.go`:

```go
package workspaces

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewArchiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive <name>",
		Short: "Archive a workspace",
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

			if err := adapter.Archive(ctx, name); err != nil {
				return fmt.Errorf("failed to archive workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' archived\n",
				styles.Success.Render("Archived"), name)
			return nil
		},
	}

	return cmd
}
```

**Step 2: Create delete command**

Create `internal/cli/workspaces/delete.go`:

```go
package workspaces

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/styles"
	ws "oss-aps-cli/internal/workspace"
)

func NewDeleteCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a workspace",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()

			// Find linked profiles
			profiles, _ := core.ListProfiles()
			var linked []string
			for _, pid := range profiles {
				p, err := core.LoadProfile(pid)
				if err != nil {
					continue
				}
				if p.Workspace != nil && p.Workspace.Name == name {
					linked = append(linked, pid)
				}
			}

			if len(linked) > 0 && !force {
				fmt.Printf("%s '%s' is linked to %d profiles:\n",
					styles.Warn.Render("Warning:"), name, len(linked))
				for _, pid := range linked {
					fmt.Printf("  %s\n", pid)
				}
				fmt.Println()
				fmt.Println("  Use --force to delete and unlink all profiles.")
				return fmt.Errorf("workspace has linked profiles")
			}

			if dryRun {
				fmt.Printf("Would delete workspace '%s'\n", name)
				if len(linked) > 0 {
					fmt.Printf("Would unlink %d profiles: %v\n", len(linked), linked)
				}
				return nil
			}

			// Unlink all profiles
			for _, pid := range linked {
				if err := ws.UnlinkProfile(pid); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to unlink %s: %v\n", pid, err)
				}
			}

			// Delete workspace
			dataDir, _ := workspaceDataDir("global", "")
			adapter, err := ws.NewAdapter(dataDir)
			if err != nil {
				return fmt.Errorf("failed to initialize workspace backend: %w", err)
			}
			defer adapter.Close()

			if err := adapter.Delete(ctx, name); err != nil {
				return fmt.Errorf("failed to delete workspace: %w", err)
			}

			fmt.Printf("%s Workspace '%s' deleted\n",
				styles.Success.Render("Deleted"), name)
			if len(linked) > 0 {
				fmt.Printf("Unlinked %d profiles\n", len(linked))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force delete even if profiles are linked")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without executing")

	return cmd
}
```

**Step 3: Register in cmd.go**

Add to `NewWorkspacesCmd()`:

```go
	cmd.AddCommand(NewArchiveCmd())
	cmd.AddCommand(NewDeleteCmd())
```

**Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 5: Commit**

```bash
git add internal/cli/workspaces/archive.go internal/cli/workspaces/delete.go internal/cli/workspaces/cmd.go
git commit -m "feat(cli): add 'workspaces archive' and 'workspaces delete' commands"
```

---

## Task 10: Update `session list` and `profile show` with workspace info

**Files:**
- Modify: `internal/cli/session/list.go`
- Modify: `internal/cli/profile.go`

**Step 1: Add WORKSPACE column to session list**

In `internal/cli/session/list.go`, change the header and row format:

Replace the header line:
```go
fmt.Fprintln(w, "ID\tPROFILE\tWORKSPACE\tPID\tSTATUS\tTIER\tCREATED\tLAST SEEN")
```

Replace the row format:
```go
wsID := s.WorkspaceID
if wsID == "" {
    wsID = "--"
}
fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n",
    s.ID,
    s.ProfileID,
    wsID,
    s.PID,
    s.Status,
    s.Tier,
    s.CreatedAt.Format("2006-01-02 15:04:05"),
    s.LastSeenAt.Format("15:04:05"))
```

**Step 2: Add workspace section to profile show**

In `internal/cli/profile.go`, in `profileShowCmd`, after the YAML output (after line 96 `fmt.Println(string(data))`), add:

```go
		// Workspace link
		if profile.Workspace != nil {
			fmt.Printf("\nWorkspace: %s (%s)\n",
				styles.Bold.Render(profile.Workspace.Name),
				profile.Workspace.Scope)
		}
```

**Step 3: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 4: Commit**

```bash
git add internal/cli/session/list.go internal/cli/profile.go
git commit -m "feat(cli): show workspace info in session list and profile show"
```

---

## Task 11: Add `--workspace` filter to session list

**Files:**
- Modify: `internal/cli/session/list.go`

**Step 1: Add the flag and filter**

Add flag registration:
```go
cmd.Flags().StringP("workspace", "w", "", "Filter sessions by workspace ID")
```

Add to the `RunE` function:
```go
workspaceFilter, _ := cmd.Flags().GetString("workspace")
```

Update `filterSessions` to accept workspace filter:
```go
func filterSessions(sessions []*session.SessionInfo, profileFilter, statusFilter, tierFilter, workspaceFilter string) []*session.SessionInfo {
```

Add filter logic:
```go
if workspaceFilter != "" && s.WorkspaceID != workspaceFilter {
    continue
}
```

**Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

**Step 3: Commit**

```bash
git add internal/cli/session/list.go
git commit -m "feat(cli): add --workspace filter to session list"
```

---

## Task 12: Final integration test and cleanup

**Step 1: Run full test suite**

Run: `go test ./internal/core/ ./internal/workspace/ ./internal/a2a/... -v`
Expected: All PASS

**Step 2: Run build**

Run: `go build ./... && go vet ./...`
Expected: Clean (ignore pre-existing webhook.go warning)

**Step 3: Verify CLI help**

Run: `go run ./cmd/aps workspaces --help`
Expected: Shows all subcommands: new, list, show, link, unlink, archive, delete

Run: `go run ./cmd/aps ws --help`
Expected: Same output (alias works)

**Step 4: Commit any remaining changes**

```bash
git add -A
git commit -m "feat(workspace): complete workspace integration"
```
