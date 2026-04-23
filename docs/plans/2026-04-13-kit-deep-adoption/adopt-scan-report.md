# Kit Adoption Scan Report -- hop.top/aps

**Date:** 2026-04-13
**Module:** `hop.top/aps` | Go 1.26.1
**Kit version:** `v0.1.1-alpha.3`
**Current kit imports:** `kit/cli`, `kit/log` (2 packages)
**Codebase:** ~49,500 LOC (non-test Go); ~14,200 LOC in `internal/cli/`

---

## 1. Dependency Inventory

| Dependency | Classification | Notes |
|------------|---------------|-------|
| `hop.top/kit v0.1.1-alpha.3` | ADOPTED | cli, log used; output/xdg/config/bus/domain/api pending |
| `hop.top/upgrade v0.0.0-...` | REPLACEABLE | kit/upgrade is the canonical home |
| `hop.top/cxr v0.0.0-...` | ADOPTED | used in adapter manager |
| `hop.top/xrr v0.1.0-alpha.1` | ADOPTED (indirect) | transitive via kit |
| `charm.land/bubbletea/v2` | KEEP | v2 already; kit/tui wraps this |
| `charm.land/lipgloss/v2` | KEEP | v2 already; 17 files import directly |
| `charm.land/log/v2` | KEEP | underlying kit/log engine |
| `github.com/charmbracelet/huh v1.0.0` | KEEP | form input; 6 files; no kit replacement |
| `github.com/spf13/cobra` | KEEP | kit/cli wraps cobra; direct use expected |
| `github.com/spf13/viper` | KEEP | kit/cli provides viper; logging wires it |
| `github.com/a2aproject/a2a-go` | STANDALONE | A2A protocol; no kit equivalent |
| `github.com/golang-jwt/jwt/v5` | STANDALONE | auth tokens; no kit equivalent |
| `github.com/google/uuid` | STANDALONE | ID generation |
| `github.com/gorilla/websocket` | STANDALONE | WS transport |
| `github.com/joho/godotenv` | STANDALONE | .env loading |
| `github.com/skip2/go-qrcode` | STANDALONE | QR for mobile pairing |
| `github.com/stretchr/testify` | KEEP | test framework |
| `go.opentelemetry.io/otel*` | STANDALONE | observability; no kit equivalent yet |
| `golang.org/x/sys` | KEEP | OS-level |
| `golang.org/x/term` | KEEP | terminal detection |
| `gopkg.in/yaml.v3` | KEEP | YAML parsing; used everywhere |

### Summary

- **ADOPTED:** 3 (kit, cxr, xrr)
- **REPLACEABLE:** 1 (hop.top/upgrade -> kit/upgrade)
- **STANDALONE:** 7 (a2a-go, jwt, uuid, websocket, godotenv,
  go-qrcode, otel)
- **KEEP:** 8 (charm v2 libs, cobra, viper, testify, x/sys,
  x/term, yaml.v3)

---

## 2. Pattern Scan -- Kit Replacements

### 2.1 kit/xdg -- XDG path handling

**Status:** NOT ADOPTED -- hand-rolled in 3 core files

| File | LOC | Pattern |
|------|-----|---------|
| `internal/core/paths.go` | 53 | Manual `XDG_DATA_HOME`, `XDG_CACHE_HOME`, `XDG_STATE_HOME` |
| `internal/core/config.go:27-39` | 12 | Manual `XDG_CONFIG_HOME` + `os.UserConfigDir()` |
| `internal/skills/paths.go` | ~30 | Manual `XDG_DATA_HOME` + `os.UserHomeDir()` |

Additional scattered `os.UserHomeDir()` calls: 12 files in
`internal/core/` and `internal/cli/`. Many are XDG-adjacent
(resolve data/config paths).

**Total affected LOC:** ~95 (production), ~250+ (tests doing
`t.Setenv("XDG_*")`)

### 2.2 kit/config -- Config loading

**Status:** NOT ADOPTED -- viper + manual yaml.Unmarshal

| File | LOC | Pattern |
|------|-----|---------|
| `internal/core/config.go:42-84` | 43 | `os.ReadFile` + `yaml.Unmarshal` for config |
| `internal/core/config.go:87-151` | 64 | `SaveConfig` + `MigrateConfig` |

`yaml.Unmarshal` appears in 30+ files total, but most are
data files (profiles, bundles, capabilities) -- NOT config.
Only `config.go` is a kit/config replacement candidate.

`viper` direct usage: `internal/logging/logger.go:74` (SetViper)
-- already bridges to kit/log via viper. No other direct viper
calls in production code.

### 2.3 kit/output -- CLI output formatting

**Status:** NOT ADOPTED -- manual json.Marshal + tabwriter + --json

| Pattern | File Count | Affected LOC |
|---------|-----------|--------------|
| `tabwriter.NewWriter` | 13 CLI files | ~200 |
| `json.MarshalIndent` + stdout | 12 CLI files | ~150 |
| `json.NewEncoder(os.Stdout)` | 6 CLI files | ~40 |
| `--json` bool flags | ~15 commands | ~45 |
| **Total** | **25 CLI files** | **~435** |

Key files (LOC):
- `internal/cli/workspace/activity.go` (274)
- `internal/cli/audit/log.go` (250)
- `internal/cli/skill/skill.go` (428)
- `internal/cli/adapter/pair.go` (342)
- `internal/cli/capability/list.go` (171)
- `internal/cli/action.go` (152)

### 2.4 kit/log -- Structured logging

**Status:** PARTIALLY ADOPTED

- `internal/logging/logger.go` -- already wraps `kit/log.New(v)`
  via `SetViper()`. Kit integration is wired.
- However, 40+ callsites use raw `fmt.Fprintf(os.Stderr, ...)`
  or `log.Printf(...)` instead of the structured logger.

| Pattern | Count | Key Locations |
|---------|-------|---------------|
| `fmt.Fprintf(os.Stderr, ...)` | 35+ | cli/root, cli/adapter, cli/acp |
| `log.Printf(...)` | 10+ | cli/serve, core/webhook, core/execution |
| `fmt.Fprintln(os.Stderr, ...)` | 5+ | cli/root, cli/workspace |

### 2.5 kit/upgrade -- Self-update

**Status:** NOT ADOPTED -- uses standalone `hop.top/upgrade`

| File | Import |
|------|--------|
| `internal/cli/root.go:14` | `hop.top/upgrade` |
| `internal/cli/upgrade.go:9-10` | `hop.top/upgrade`, `hop.top/upgrade/skill` |
| `internal/cli/upgrade_test.go:11` | `hop.top/upgrade` |

Direct 1:1 replacement. Kit/upgrade is the same package
rehosted under kit's vanity URL.

### 2.6 kit/bus -- Event bus

**Status:** NOT ADOPTED -- no bus imports anywhere

Zero event emission in the codebase. Audit log writes, webhook
dispatch, and metrics collection are all synchronous inline
calls. Bus adoption is additive (no existing code to replace).

---

## 3. Boundary Analysis -- ash / wsm / usp

### Ownership Table Reference

| Domain | Owner | Key Types |
|--------|-------|-----------|
| Profile, Grant, Capability | **aps** | Profile, Grant, Capability |
| Session, Snapshot, Turn, Fork | **ash** | Session, Snapshot |
| Workspace, Space, Mutation, Device | **wsm** | Workspace |

### Findings

**Session:** aps defines `ACPSession` in `internal/acp/` (35+
references). This is an ACP-protocol-level session, NOT the ash
domain session. The type tracks: `SessionID`, `ProfileID`,
`Mode` (default/auto-approve/read-only). This is **valid** --
ACP sessions are aps-owned transport sessions, distinct from
ash's conversation sessions.

Additionally `internal/core/session/` contains:
- `registry.go` -- session registry with `WorkspaceID` field
- `ssh_keys.go` -- SSH key management for sessions
- `tmux_config.go` -- tmux session configuration

These manage the runtime isolation/execution session, not ash's
conversation session. **Acceptable but warrants documentation.**

**Workspace:** `internal/core/collaboration/workspace.go` defines
a full `Workspace` struct with `AddAgent`, `RemoveAgent`,
`SetState`, agent roles, capacity limits.
`internal/storage/collaboration.go` persists workspaces to disk.
`internal/core/profile.go:116` defines `WorkspaceLink`.

**BOUNDARY VIOLATION (MODERATE):** aps implements its own
Workspace entity with CRUD + state machine. Per architecture,
wsm owns Workspace. However, this appears to be a collaboration
workspace for multi-agent coordination within aps, predating
wsm's existence. Needs reconciliation:
- Option A: Migrate to wsm dependency when wsm stabilizes
- Option B: Rename to `CollaborationContext` to avoid collision

**Device/Mutation/Space:** No direct `Mutation` or `Space` types
defined. `Device` appears only in adapter/mobile context
(`MaxDevices` on profile config) -- device pairing, not wsm
device concept. **No violation.**

**Snapshot/Turn/Fork:** Zero hits. **Clean.**

---

## 4. Protocol Integration Gaps

| Protocol | Status | Notes |
|----------|--------|-------|
| `hop.top/uri` | MISSING | Not imported; profile IDs are plain strings |
| `hop.top/xrr` | INDIRECT | In go.mod as transitive dep; not directly imported |
| `hop.top/eva` | MISSING | No evaluation/contract framework |
| `hop.top/ben` | MISSING | No benchmarking harness |
| `kit/toolspec` | MISSING | No CLI knowledge spec for agent consumption |
| `kit/bus` | MISSING | No event bus (see 2.6) |

### Priority Assessment

| Protocol | Priority | Rationale |
|----------|----------|-----------|
| `kit/bus` | P1 | Decouple audit, webhooks, metrics |
| `kit/toolspec` | P2 | Agent ecosystem needs CLI discovery |
| `hop.top/uri` | P2 | Structured identity for profile refs |
| `hop.top/xrr` | P3 | Error registry; useful but not blocking |
| `hop.top/eva` | P3 | Contract testing; nice-to-have |
| `hop.top/ben` | P4 | Benchmarking; lowest priority |

---

## 5. Access Control Audit

### Profile Ownership

aps correctly owns profiles:
- `internal/core/profile.go` -- full CRUD (create, load, list,
  delete, export, import)
- `internal/core/profile_bundle.go` -- bundle assignment
- `internal/core/profile_export.go` -- share/import
- `internal/core/profile_delete.go` -- deletion with cleanup

### Grant/Permission Patterns

- `internal/acp/permissions.go` -- `PermissionManager` with
  `GrantPermission()`, request/response lifecycle
- Grants are session-scoped (ACP session, not persistent)
- No persistent grant store (grants reset on session end)

### Findings

- **No violations.** Profile lifecycle is entirely within aps.
- **Gap:** No `hop.top/uri` for profile identity; profiles use
  plain string IDs. Structured URIs would enable cross-tool
  profile references (`aps://profile/<id>`).
- **Gap:** Grant model is ephemeral (ACP session-scoped). No
  persistent capability grants. This is fine for current scope
  but limits cross-session permission persistence.

---

## 6. Charm v2 Migration Check

### Direct Imports (source files, non-test)

| Package | Version | Files | Status |
|---------|---------|-------|--------|
| `charm.land/bubbletea/v2` | v2.0.2 | 3 (tui/) | MIGRATED |
| `charm.land/lipgloss/v2` | v2.0.2 | 17 | MIGRATED |
| `charm.land/log/v2` | v2.0.0 | 1 (logging/) | MIGRATED |
| `github.com/charmbracelet/huh` | v1.0.0 | 6 | NOT v2 |

### Indirect (go.mod, transitive)

| Package | Version | Source |
|---------|---------|--------|
| `github.com/charmbracelet/bubbletea` | v1.3.10 | indirect; huh dep |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | indirect; huh dep |
| `github.com/charmbracelet/bubbles` | v1.0.0 | indirect; huh dep |

### Assessment

Core TUI and styling use **charm.land v2** (correct). The
`charmbracelet/huh v1.0.0` form library pulls in v1 bubbletea
and lipgloss as indirect deps. `huh` has no v2 release yet
under `charm.land/`. When it ships, update to eliminate the v1
transitive deps.

**6 files import huh v1:**
- `internal/cli/prompt/confirm.go`
- `internal/cli/collab/new.go`
- `internal/cli/collab/resolve.go`
- `internal/cli/collab/role.go`
- `internal/cli/profile.go`
- `internal/cli/adapter/create.go`

---

## 7. LOC Estimation

### Replacement Opportunities

| Kit Package | Current LOC | Est. After | Delta | Files |
|-------------|------------|------------|-------|-------|
| kit/xdg | 95 | 15 | -80 | 3 prod + ~20 test |
| kit/config | 107 | 35 | -72 | 1 prod + 1 test |
| kit/upgrade | 98 | 90 | -8 | 3 (import swap) |
| kit/output | 435 | 180 | -255 | 25 CLI files |
| kit/log (stderr cleanup) | 50 | 50 | 0 | 20 files (refactor, no LOC drop) |
| kit/bus | 0 | +120 | +120 | new: events.go + subscribers |
| kit/api | 145 | 80 | -65 | serve.go + handlers |
| kit/toolspec | 0 | +80 | +80 | new: toolspec/aps.go |
| kit/domain | 0 | +150 | +150 | new: entity interfaces |
| kit/tui | 456 | 350 | -106 | 4 TUI files |
| **Total** | **1,386** | **1,150** | **-236 net** | |

### Notes

- kit/output delivers the largest LOC savings (-255) across 25
  files by eliminating tabwriter + json.Marshal boilerplate
- kit/bus and kit/domain are additive (+270 combined) but
  provide structural value (decoupling, consistency)
- Test LOC savings from kit/xdg are substantial (~200 lines of
  `t.Setenv("XDG_*")` boilerplate) but not counted above
- Existing plan correctly prioritizes foundation (xdg, config,
  upgrade) before output and structural tracks

---

## 8. Plan Validation

### Existing Plan Coverage

The plan at `plan.md` covers 6 tracks with 20 tasks. Comparing
against scan findings:

| Area | Plan Coverage | Scan Finding | Gap? |
|------|--------------|--------------|------|
| kit/xdg | Task 1.1 | 3 files, 95 LOC | Covered |
| kit/config | Task 1.2 | config.go only | Covered |
| kit/upgrade | Task 1.3 | 3 files, import swap | Covered |
| kit/output | Tasks 2.1-2.4 | 25 files, 435 LOC | Covered |
| kit/api | Tasks 3.1-3.3 | serve.go + 3 http files | Covered |
| kit/bus | Tasks 4.1-4.3 | Zero current bus usage | Covered |
| kit/domain | Tasks 5.1-5.4 | Additive | Covered |
| kit/tui | Tasks 6.1-6.3 | 456 LOC in tui/ | Covered |
| Boundary: Workspace | NOT in plan | Moderate violation | **GAP** |
| hop.top/uri | NOT in plan | Missing structured IDs | **GAP** |
| huh v1 cleanup | NOT in plan | 6 files, blocked upstream | Minor |
| stderr/log.Printf cleanup | NOT in plan | 50+ callsites | **GAP** |

### Recommended Additions to Plan

1. **Task 1.5 (new): Audit workspace boundary** -- Document
   relationship between `internal/core/collaboration/workspace.go`
   and wsm. Decide: rename to `CollaborationContext` or plan
   wsm migration. Effort: XS.

2. **Task 1.6 (new): Adopt hop.top/uri for profile identity** --
   Replace plain string profile IDs with `uri.Parse("aps",
   "profile", id)`. Enables cross-tool profile references.
   Effort: M (touches many files).

3. **Task 2.5 (new): Migrate raw stderr writes to kit/log** --
   Replace `fmt.Fprintf(os.Stderr, ...)` and `log.Printf(...)`
   with structured logger calls. 50+ callsites. Effort: S.

4. **Task 6.4 (new): Track huh v2 migration** -- When
   `charm.land/huh/v2` releases, update 6 files and eliminate
   v1 transitive deps. Effort: S (blocked upstream).

### Plan Accuracy

The existing plan is **well-structured and accurate**. Track
ordering (foundation -> output -> structural) is correct. LOC
estimates align with scan. The 4 gaps above are minor/additive
and do not invalidate any existing tasks.

---

## Appendix: File Index

### Top Priority Files (Track 1: kit-foundation)

```
internal/core/paths.go           53 LOC  -> kit/xdg
internal/core/config.go         151 LOC  -> kit/xdg + kit/config
internal/cli/upgrade.go          98 LOC  -> kit/upgrade
internal/cli/root.go            111 LOC  -> kit/upgrade import
internal/skills/paths.go         ~30 LOC -> kit/xdg
```

### High-Value Files (Track 2: kit-output)

```
internal/cli/workspace/activity.go   274 LOC
internal/cli/audit/log.go            250 LOC
internal/cli/skill/skill.go          428 LOC
internal/cli/adapter/pair.go         342 LOC
internal/cli/capability/list.go      171 LOC
internal/cli/action.go               152 LOC
internal/cli/serve.go                145 LOC
internal/cli/bundle/list.go          145 LOC
internal/cli/adapter/channels.go     141 LOC
```

### Boundary Files (Workspace overlap)

```
internal/core/collaboration/workspace.go   ~250 LOC
internal/storage/collaboration.go          ~310 LOC
internal/core/profile.go:96,115-117        WorkspaceLink
```
