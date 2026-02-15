# Design: APS-Workspace Integration

**Date**: 2026-02-14
**Status**: Approved
**Dependencies**: hop.top/wsm (workspace-cli)

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Command name | `workspaces` (plural), alias `ws` | Consistent with `messengers` pattern |
| Verbs | APS convention: `new`/`show` | APS is host; register `create`/`inspect` as aliases |
| Cardinality | 1:1 profile-to-workspace | Simplest for v1. Expand to 1:N in Plan 7 |
| Delete safety | Warn + `--force` | Follows `cap delete` pattern |
| Auto-link | Yes, if `APS_PROFILE` set | `--no-link` to skip |
| Vendoring | Import `hop.top/wsm` | WSM exists; use its Manager API directly |
| Scope default | `global` for `new`, `all` for `list` | Global is the common case |
| Sort default | By last activity, recent first | `--sort name|activity|created` |

## Profile Extension

Add optional workspace field to Profile struct in `internal/core/profile.go`:

```go
type WorkspaceLink struct {
    Name  string `yaml:"name"`
    Scope string `yaml:"scope"` // "global" or "profile"
}

type Profile struct {
    // ... existing fields ...
    Workspace *WorkspaceLink `yaml:"workspace,omitempty"`
}
```

100% backward compatible — pointer + omitempty.

## Session Extension

Add optional workspace ID to SessionInfo in `internal/core/session/registry.go`:

```go
type SessionInfo struct {
    // ... existing fields ...
    WorkspaceID string `json:"workspace_id,omitempty"`
}
```

## Adapter Layer

`internal/workspace/adapter.go` wraps `wsm` Manager, providing:

1. **Manager initialization** — creates SQLite backend at `~/.aps/workspaces/wsm.db` (global) or `~/.aps/profiles/<id>/workspaces/wsm.db` (profile-scoped)
2. **Error translation** — maps wsm errors to APS error types:

```
wsm ErrorKindNotFound → core.NotFoundError     (exit 3)
wsm ErrorKindConflict → core.InvalidInputError  (exit 2)
wsm ErrorKindBackend  → fmt.Errorf wrapped      (exit 1)
wsm ErrorKindUsage    → core.InvalidInputError  (exit 2)
```

3. **Profile linking** — reads/writes WorkspaceLink on Profile structs

Key functions:

```go
func NewAdapter(scope string, profileID string) (*Adapter, error)
func (a *Adapter) Create(ctx context.Context, name string, opts CreateOptions) (*Workspace, error)
func (a *Adapter) List(ctx context.Context, opts ListOptions) ([]Workspace, error)
func (a *Adapter) Get(ctx context.Context, ref string) (*Workspace, error)
func (a *Adapter) Archive(ctx context.Context, ref string) error
func (a *Adapter) Delete(ctx context.Context, ref string, force bool) error
func (a *Adapter) Close() error

func LinkProfile(profileID string, workspaceName string, scope string) error
func UnlinkProfile(profileID string) error
func GetLinkedWorkspace(profileID string) (*WorkspaceLink, error)
```

## CLI Commands

### Command Group: `internal/cli/workspaces/cmd.go`

```
aps workspaces <subcommand>
aps ws <subcommand>
```

### `workspaces new <name> [--scope global|profile] [--no-link]`

Aliases: `create`

- Creates workspace via adapter
- If `APS_PROFILE` is set and `--no-link` not passed, auto-links
- Default scope: `global`

Output (TTY):
```
Workspace 'dev-project' created (scope: global)
Linked to active profile 'agent-alpha'

  View workspace:
    aps workspaces show dev-project
```

Output (`--json`): `{"name":"dev-project","scope":"global","linked_profile":"agent-alpha"}`
Output (`--quiet`): exit code only

### `workspaces list [--scope all|global|profile] [--sort name|activity|created]`

Aliases: `ls`

Default scope: `all`. Default sort: `activity`.

Output (TTY):
```
Workspaces

NAME           SCOPE     PROFILES   STATUS
dev-project    global    2          active
staging        global    1          active
my-sandbox     profile   1          active

3 workspaces (2 global, 1 profile-scoped)
```

Empty state:
```
No workspaces yet.

  Create your first workspace:
    aps workspaces new my-project
```

### `workspaces show <name>`

Aliases: `inspect`

Output (TTY):
```
dev-project

Scope:         global
Status:        active
Created:       2026-02-10 14:30
Last Activity: 12 minutes ago

Linked Profiles:
  agent-alpha    Agent Alpha
  agent-beta     Agent Beta

2 profiles linked
```

### `workspaces link <profile> <workspace> [--scope global]`

- Profile is arg[0] (entity being modified comes first)
- Validates both profile and workspace exist
- Shell completion: profiles for arg[0], workspace names for arg[1]

Error UX:
```
$ aps workspaces link agent-x my-workspace
Error: profile 'agent-x' not found

  Available profiles:
    agent-alpha    Agent Alpha
    agent-beta     Agent Beta

  To create a new profile:
    aps profile new agent-x
```

### `workspaces unlink <profile> [--force]`

Without `--force`:
```
Unlinking profile 'agent-alpha' from workspace 'dev-project'...

  This will:
    - Remove workspace context from profile
    - Active sessions will lose workspace access

  Proceed? [y/N]:
```

With `--force`: unlinks silently, prints confirmation.

### `workspaces archive <name>`

Archives workspace. Linked profiles retain the link but workspace becomes inactive.

### `workspaces delete <name> [--force] [--dry-run]`

Without `--force`:
```
Warning: 'dev-project' is linked to 2 profiles:
  agent-alpha, agent-beta

  Use --force to delete and unlink all profiles.
```

`--dry-run`: shows impact without executing.

## Updated Existing Commands

### `session list` — add WORKSPACE column

```
ID     PROFILE       WORKSPACE     STATUS   TIER
abc    agent-alpha   dev-project   active   standard
def    agent-beta    --            active   basic
```

### `profile show <id>` — add Workspace section

```
...
Workspace: dev-project (global)
...
```

If no workspace linked, omit the section entirely.

## Error Handling

All workspace commands use APS error patterns:

- Profile not found → suggest `aps profile list` + create hint
- Workspace not found → suggest `aps workspaces list` + create hint
- Workspace already exists → suggest different name
- Scope mismatch → explain scope semantics

## Shell Completion

- `workspaces show/archive/delete`: complete workspace names
- `workspaces link` arg[0]: complete profile IDs
- `workspaces link` arg[1]: complete workspace names
- `workspaces unlink`: complete profile IDs

## Files

| File | Purpose |
|------|---------|
| `internal/core/profile.go` | Add `WorkspaceLink` type + field |
| `internal/core/session/registry.go` | Add `WorkspaceID` field |
| `internal/workspace/adapter.go` | Adapter wrapping wsm Manager |
| `internal/workspace/adapter_test.go` | Adapter unit tests |
| `internal/workspace/link.go` | Profile-workspace linking logic |
| `internal/workspace/link_test.go` | Linking unit tests |
| `internal/cli/workspaces/cmd.go` | Command group + registration |
| `internal/cli/workspaces/new.go` | `new` / `create` command |
| `internal/cli/workspaces/list.go` | `list` / `ls` command |
| `internal/cli/workspaces/show.go` | `show` / `inspect` command |
| `internal/cli/workspaces/link.go` | `link` command |
| `internal/cli/workspaces/unlink.go` | `unlink` command |
| `internal/cli/workspaces/archive.go` | `archive` command |
| `internal/cli/workspaces/delete.go` | `delete` / `rm` command |
| `internal/cli/workspaces.go` | Root registration file |

## Backward Compatibility

100% backward compatible:
- `WorkspaceLink` is a pointer field with `omitempty`
- `WorkspaceID` on sessions is a string with `omitempty`
- Existing profiles/sessions load without workspace data
- No migration needed — new fields are simply absent
