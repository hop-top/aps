# Kit Deep Adoption — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans
> to implement this plan task-by-task.

**Goal:** Bring aps from `{cli, log}` to full kit parity with pod/tip/foo,
replacing hand-rolled infrastructure with shared kit packages.

**Architecture:** Six tracks mirror tlc's kit-* pattern. Each track is
independently shippable. Foundation first (xdg, config, upgrade),
then output harmonization, then deeper structural adoption (domain,
api, bus, ext, tui). Flag audit is cross-cutting prerequisite for
output + CLI alignment.

**Tech Stack:** Go 1.26, hop.top/kit v0.1.1-alpha.3+, cobra, viper,
charm.land/*, SQLite

**Cross-tool constraint:** all CLI flags/names/shortnames/aliases MUST
match the ecosystem convention. `hop-top/tlc#T-0504` (flag audit) is
the canonical reference — aps tasks depend on or contribute to it.

---

## Track Overview

| Track ID | Title | Type | Deps |
|----------|-------|------|------|
| kit-foundation | Kit Foundation (xdg, config, upgrade) | refactor | — |
| kit-output | Kit Output + Flag Harmonization | refactor | kit-foundation |
| kit-api | Kit API Server | refactor | kit-foundation |
| kit-bus | Kit Bus Wiring | refactor | kit-foundation |
| kit-domain | Kit Domain + SQLStore | refactor | kit-foundation |
| kit-tui-ext | Kit TUI + Ext + Toolspec | refactor | kit-output |

---

## Track 1: kit-foundation

Replace hand-rolled XDG, config loading, and consolidate upgrade dep.

### Task 1.1: Replace GetConfigDir with kit/xdg

**Files:**
- Modify: `internal/core/config.go:27-39`
- Modify: `internal/skills/paths.go` (if uses GetConfigDir)
- Modify: `internal/cli/upgrade.go:79-94` (installAPSPreamble)
- Test: `internal/core/config_test.go` (add if missing)

**Steps:**
1. Write test: `GetConfigDir()` returns same path as
   `xdg.ConfigDir("aps")`
2. Run test — verify it passes (both should resolve identically)
3. Replace `GetConfigDir()` body with `return xdg.ConfigDir("aps")`
4. Add `xdg.DataDir("aps")`, `xdg.StateDir("aps")`,
   `xdg.CacheDir("aps")` helpers if any code uses those patterns
5. Delete manual `XDG_CONFIG_HOME` / `os.UserConfigDir()` logic
6. Update `installAPSPreamble` to use `xdg.ConfigDir("aps")`
7. Run: `go build ./... && go test ./internal/core/...`
8. Commit: `refactor: replace GetConfigDir with kit/xdg`

### Task 1.2: Replace config loading with kit/config.Load

**Files:**
- Modify: `internal/core/config.go:42-84` (LoadConfig)
- Modify: `internal/core/config.go:87-108` (SaveConfig)
- Test: `internal/core/config_test.go`

**Steps:**
1. Write test: `LoadConfig()` with system + user + project YAML files
   returns merged result (kit/config merges system→user→project→env)
2. Run test — fails (current impl reads single file)
3. Rewrite `LoadConfig()` using `config.Load(&cfg, config.Options{...})`
   with `UserConfigPath: xdg.ConfigDir("aps") + "/config.yaml"`
4. Keep `SaveConfig()` as-is (kit/config is read-only; write is custom)
5. Keep `MigrateConfig()` as-is (migration logic is aps-specific)
6. Run: `go test ./internal/core/...`
7. Commit: `refactor: replace LoadConfig with kit/config.Load`

### Task 1.3: Consolidate hop.top/upgrade → kit/upgrade

**Files:**
- Modify: `internal/cli/upgrade.go` (imports)
- Modify: `internal/cli/root.go:14` (import)
- Modify: `go.mod` (remove standalone hop.top/upgrade dep)
- Test: `internal/cli/upgrade_test.go`

**Steps:**
1. Check if `kit/upgrade` API matches `hop.top/upgrade` (same origin;
   should be identical or superset)
2. Replace `"hop.top/upgrade"` → `"hop.top/kit/upgrade"` in all files
3. Replace `"hop.top/upgrade/skill"` → `"hop.top/kit/upgrade/skill"`
4. Run: `go build ./... && go test ./internal/cli/...`
5. Remove `hop.top/upgrade` from go.mod requires
6. Run: `go mod tidy && go mod vendor`
7. Verify: `grep 'hop.top/upgrade' go.mod` shows zero direct deps
8. Commit: `refactor: consolidate upgrade dep into kit/upgrade`

### Task 1.4: Remove superseded code + tidy

**Files:**
- Modify: `go.mod`, `go.sum`
- Verify: `internal/core/config.go` (no manual XDG logic remains)

**Steps:**
1. Run: `go mod tidy && go mod vendor`
2. Run: `go build ./... && go test ./...`
3. Verify no `os.Getenv("XDG_CONFIG_HOME")` remains in core/config.go
4. Commit: `chore: tidy deps after kit foundation adoption`

---

## Track 2: kit-output + Flag Harmonization

Replace ad-hoc `--json` bools with kit/output `--format` flag.
Requires flag audit across ecosystem first.

### Task 2.1: Flag audit (cross-tool, contributes to tlc#T-0504)

**Files:**
- Create: `docs/plans/2026-04-13-kit-deep-adoption/flag-audit.md`

**Steps:**
1. For each tool (pod, tlc, tip, foo, aps): list all global +
   per-subcommand flags with `--name`, `-short`, type, default
2. Document in table: flag name | shortname | tools that use it
3. Identify conflicts (same shortname, different meaning)
4. Identify missing (aps has `--json` where others use `--format`)
5. Propose canonical flag set aligned with kit/cli + kit/output
6. Commit: `docs: flag audit for kit CLI harmonization`

### Task 2.2: Wire kit/output.RegisterFlags on root command

**Files:**
- Modify: `internal/cli/root.go`
- Test: `internal/cli/root_test.go` (add flag presence test)

**Steps:**
1. Write test: root command has `--format` persistent flag
2. Run test — fails
3. Add `output.RegisterFlags(rootCmd, root.Viper)` in `init()`
4. Add `output.RegisterHintFlags(rootCmd, root.Viper)` for `--no-hints`
5. Run test — passes
6. Commit: `feat: wire kit/output --format + --no-hints flags`

### Task 2.3: Replace `--json` with kit/output.Render

**Files:**
- Modify: `internal/cli/action.go` (actionListCmd)
- Modify: `internal/cli/profile.go` (profileListCmd)
- Modify: `internal/cli/version.go` (versionCmd)
- Modify: all other commands with `--json` flag

**Steps:**
1. For each command with `--json`:
   a. Remove `--json` flag registration
   b. Read format from viper: `root.Viper.GetString("format")`
   c. Replace manual JSON marshal with `output.Render(w, fmt, data)`
   d. Table format becomes default (matches current default behavior)
2. Backward compat: `--json` removed; `--format json` is the
   replacement. This is a breaking change — document in CHANGELOG.
3. Run: `go build ./... && go test ./...`
4. Commit: `refactor!: replace --json flags with kit/output --format`

### Task 2.4: Wire hint system for post-command suggestions

**Files:**
- Modify: `internal/cli/root.go`
- Modify: select high-value commands (profile list, workspace list)

**Steps:**
1. Register hints: `root.Hints.Register("profile list", ...)`
   e.g. "Run `aps profile show <id>` to see full details"
2. Hints auto-render when output is TTY and `--no-hints` not set
3. Run: `aps profile list` — verify hint appears
4. Commit: `feat: add post-command hints via kit/output`

---

## Track 3: kit-api Server

Replace raw http.NewServeMux + hand-rolled auth in serve.go.

### Task 3.1: Replace ServeMux with kit/api.Router

**Files:**
- Modify: `internal/cli/serve.go:49-115`
- Test: `tests/e2e/serve_test.go` (or existing)

**Steps:**
1. Write test: `GET /health` returns 200 with JSON
2. Replace `http.NewServeMux()` with `api.NewRouter()`
3. Replace health handler with `router.Handle("GET", "/health", ...)`
4. Mount adapter routes via `router.Mount(...)`
5. Run test — passes
6. Commit: `refactor: replace raw ServeMux with kit/api.Router`

### Task 3.2: Replace hand-rolled auth with kit/api.Auth middleware

**Files:**
- Modify: `internal/cli/serve.go:117-145` (authMiddleware)

**Steps:**
1. Write test: request without token → 401; with token → 200
2. Replace `authMiddleware()` with `api.Auth(func)` middleware
3. Add `api.Recovery()` + `api.RequestID()` middleware
4. Wire via `api.NewRouter(api.WithMiddleware(...))`
5. Delete `authMiddleware` function
6. Run test — passes
7. Commit: `refactor: replace auth middleware with kit/api.Auth`

### Task 3.3: Add structured error responses

**Files:**
- Modify: adapter handlers that return errors

**Steps:**
1. Replace `http.Error(w, msg, status)` with `api.Error(w, status, err)`
2. Map domain errors via `api.MapError(err)` where applicable
3. Run: `go test ./...`
4. Commit: `refactor: adopt kit/api structured error responses`

---

## Track 4: kit-bus Wiring

Add in-process event bus for decoupling audit, webhooks, observability.

### Task 4.1: Create shared bus instance

**Files:**
- Create: `internal/core/events.go`
- Test: `internal/core/events_test.go`

**Steps:**
1. Define topic constants: `"profile.create"`, `"profile.delete"`,
   `"session.start"`, `"webhook.receive"`, `"action.run"`, etc.
2. Create `NewBus() bus.Bus` factory
3. Write test: publish event, subscriber receives it
4. Commit: `feat: add kit/bus event bus with topic constants`

### Task 4.2: Emit events from core operations

**Files:**
- Modify: `internal/core/profile.go` (create/delete publish events)
- Modify: `internal/core/session.go` (start/stop publish events)

**Steps:**
1. Add bus parameter to core functions (or inject via context)
2. Publish events after successful operations
3. Existing behavior unchanged; events are additive
4. Write tests: operation triggers expected event
5. Commit: `feat: emit bus events from core profile/session ops`

### Task 4.3: Wire audit log as bus subscriber

**Files:**
- Modify: `internal/core/collaboration/audit.go`

**Steps:**
1. Subscribe audit logger to `"#"` (all events) or specific patterns
2. Replace direct audit writes with bus-driven audit writes
3. Run: `go test ./internal/core/collaboration/...`
4. Commit: `refactor: wire audit log as kit/bus subscriber`

---

## Track 5: kit-domain + SQLStore

Formalize entity/repository/service patterns.

### Task 5.1: Evaluate domain model fit

**Files:**
- Create: `docs/plans/2026-04-13-kit-deep-adoption/domain-mapping.md`

**Steps:**
1. List all aps entities: Profile, Workspace, Session, Capability,
   Bundle, Action, Squad, Webhook, etc.
2. For each: does it implement `domain.Entity` (has `GetID() string`)?
3. For each: does it have CRUD ops that fit `domain.Repository[T]`?
4. Identify which entities benefit from `domain.StateMachine`
   (Session: created→active→ended; Workspace lifecycle)
5. Document mapping decisions
6. Commit: `docs: domain model mapping for kit/domain adoption`

### Task 5.2: Adopt domain.Entity on Profile

**Files:**
- Modify: `internal/core/profile.go`
- Create: `internal/storage/profile_repo.go`
- Test: `internal/storage/profile_repo_test.go`

**Steps:**
1. Add `GetID() string` to Profile (returns ProfileID)
2. Create `ProfileRepository` implementing `domain.Repository[Profile]`
3. Back with existing file-based storage initially
4. Write tests for CRUD operations
5. Commit: `refactor: adopt domain.Entity + Repository on Profile`

### Task 5.3: Adopt domain.StateMachine for Session

**Files:**
- Modify: `internal/core/session.go`
- Test: `internal/core/session_test.go`

**Steps:**
1. Define session states + valid transitions in `domain.StateMachine`
2. Replace manual state checks with state machine enforcement
3. Write tests for valid + invalid transitions
4. Commit: `refactor: adopt domain.StateMachine for Session`

### Task 5.4: Introduce kit/sqlstore for caching

**Files:**
- Create: `internal/storage/cache.go`
- Test: `internal/storage/cache_test.go`

**Steps:**
1. Identify cacheable data (capability resolution, adapter discovery)
2. Create `sqlstore.Open(xdg.CacheDir("aps") + "/cache.db", opts)`
3. Wire into capability resolver as cache layer
4. Write tests with TTL expiry
5. Commit: `feat: add kit/sqlstore cache for capability resolution`

---

## Track 6: kit-tui + ext + toolspec

Adopt themed TUI components, extension framework, and CLI knowledge.

### Task 6.1: Replace internal/tui spinner/progress with kit/tui

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/view.go`

**Steps:**
1. Audit: does internal/tui use spinner, progress, list, confirm?
2. Replace any matching components with `kit/tui.*` equivalents
3. Kit/tui components are pre-themed with hop.top accent colors
4. Run TUI: `aps` — verify visual parity
5. Commit: `refactor: adopt kit/tui components`

### Task 6.2: Formalize adapter registry as kit/ext

**Files:**
- Modify: `internal/adapters/registry.go`
- Test: `internal/adapters/registry_test.go`

**Steps:**
1. Audit: `adapters.Registry` vs `ext.Manager` + `ext.Extension`
2. Implement `ext.Extension` interface on each adapter
3. Replace `adapters.GetRegistry()` with `ext.NewManager()`
4. Wire lifecycle: `manager.InitAll()` / `manager.CloseAll()`
5. Run: `go test ./internal/adapters/...`
6. Commit: `refactor: formalize adapters as kit/ext extensions`

### Task 6.3: Create aps toolspec for agent consumption

**Files:**
- Create: `internal/toolspec/aps.go`
- Test: `internal/toolspec/aps_test.go`

**Steps:**
1. Define `toolspec.ToolSpec` for aps: all commands, flags,
   error patterns, common workflows
2. Register as source: `toolspec.NewRegistry(apsSource)`
3. Expose via `aps toolspec` command (JSON/YAML output)
4. Write test: spec includes all registered cobra commands
5. Commit: `feat: add aps toolspec for agent CLI knowledge`

---

## Dependency Graph

```text
kit-foundation ──→ kit-output ──→ kit-tui-ext
       │
       ├──→ kit-api
       ├──→ kit-bus
       └──→ kit-domain
```

kit-foundation is the only hard prerequisite. Tracks 3–5 are
independent of each other. Track 6 depends on output (for
flag conventions).

## Flag Naming Convention (from audit)

All hop.top tools MUST use these canonical flags:

| Flag | Short | Scope | Source |
|------|-------|-------|--------|
| `--quiet` | `-q` | global persistent | kit/cli |
| `--no-color` | — | global persistent | kit/cli |
| `--format` | `-f` | global persistent | kit/output |
| `--no-hints` | — | global persistent | kit/output |
| `--verbose` | `-v` | per-command | convention |
| `--yes` | `-y` | per-command (confirm skip) | convention |
| `--force` | — | per-command (overwrite) | convention |
| `--dry-run` | `-n` | per-command | convention |
| `--json` | — | **REMOVED** — use `--format json` | kit/output |

`-v` MUST NOT be used for `--version` (kit/cli uses `-v` flag-free;
version is `aps --version` or `aps -v` via fang shortcut).
