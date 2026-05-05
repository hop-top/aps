# Aps policies

Aps uses kit's runtime policy engine (`hop.top/kit/go/runtime/policy`)
to gate state changes. Policies are declarative YAML rules,
compiled once at boot, and evaluated synchronously on every state
change. A denial vetoes the change before any row mutation.

For engine semantics, expression vocabulary, deny-overrides
composition, and the canonical worked examples see kit ADR-0008
(`docs/adr/0008-kit-runtime-policy-engine.md` in the kit repo).
This doc covers aps-specific adoption only.

## Default policies

Aps ships two rules — both gate destructive operations on shared
state by requiring an audit `--note`:

```yaml
policies:
  - name: session-delete-requires-note
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || context.note != ""'
    effect: allow
    otherwise: deny
    message: "deleting a session requires --note explaining why"

  - name: workspace-ctx-delete-requires-note
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || context.note != ""'
    effect: allow
    otherwise: deny
    message: "deleting a workspace context variable requires --note explaining why"
```

Effect: `aps session delete` and `aps workspace ctx delete` without
`--note` are rejected with exit 4.

```text
$ aps session delete sess-7c41
Error: policy "session-delete-requires-note" denied: deleting a session requires --note explaining why
$ echo $?
4
```

```text
$ aps session delete sess-7c41 --note "stale; profile retired"
Deleted session sess-7c41
```

Non-destructive transitions (create, update, attach, link, …) are
not gated by default — adopters opt in by adding rules to their
own file.

A future rule, `cross-agent-context-access-requires-role`, will
gate `aps workspace ctx set` cross-profile reads/writes by
`principal.role`. Ships in T-1302; not enforced by the current
default file.

## Where the policy file lives

| Resolution order | Source |
|------------------|--------|
| 1 | `$APS_POLICY_FILE` env var (used by tests + CI) |
| 2 | `$XDG_CONFIG_HOME/aps/policies.yaml` (default `~/.config/aps/policies.yaml`) |

On first boot, aps seeds option 2 from the bundled default if it
is missing or empty. **Existing non-empty user files are never
clobbered.** Edit the file in place to extend or override.

If both seeding fails (read-only home, etc.) and `$APS_POLICY_FILE`
is unset, aps falls back to the embedded default — enforcement
still applies. The error surfaces the next time the user tries to
edit and the file isn't there.

## Anatomy of a policy

```yaml
policies:
  - name: <unique-string>          # for error messages + logs
    on: <kit-topic>                # see "Available topics" below
    when: <CEL expression>         # evaluated per matching event
    effect: allow | deny           # outcome when `when` is true
    otherwise: allow | deny        # outcome when `when` is false
    message: <string>              # surfaced via PolicyDeniedError
```

Both `effect` and `otherwise` are required — no implicit defaults.
This forces every policy to state both match-outcome and
no-match-outcome explicitly.

When multiple policies match a single event, kit applies
**deny-overrides**: ANY policy resolving to `deny` wins, and the
first denying policy's message is surfaced. See ADR-0008 §8.

Worked example — pulling apart `session-delete-requires-note`:

| Field | Value | Why |
|-------|-------|-----|
| `name` | `session-delete-requires-note` | Surfaced in errors and logs; must be unique within the file |
| `on` | `kit.runtime.entity.pre_persisted` | Fires after validation, before the SessionManager removes the row |
| `when` | `payload.Op != "delete" \|\| context.note != ""` | Match (allow) on every non-delete op; for deletes, only match when `--note` was supplied |
| `effect` | `allow` | When `when` is true, allow the op |
| `otherwise` | `deny` | When `when` is false (delete + empty note), veto |
| `message` | `deleting a session requires --note explaining why` | Stamped into the `PolicyDeniedError`; surfaces verbatim on the CLI |

## Available topics

Aps subscribes the canonical kit entity topics published by
`domain.Service` (T-1290 refactor):

| Topic | When it fires | Aps mutators that publish |
|-------|--------------|---------------------------|
| `kit.runtime.entity.pre_validated` | Before validation; raw payload | `SessionManager` create/update/delete; `WorkspaceContext` set/delete |
| `kit.runtime.entity.pre_persisted` | After validation, before repo write | Same as above; default rules veto here |
| `kit.runtime.state.pre_transitioned` | Valid kit topic; aps does not publish it yet | (none) |

Writing a policy on `kit.runtime.state.pre_transitioned` is not an
error — it loads, compiles, never matches, never fires. Aps tracks
topic-coverage expansion as a follow-up: as more entities adopt
`domain.Service`, the same two topics widen automatically.

In addition to the canonical topics, aps fans out
`aps.profile.*`, `aps.adapter.*`, and `aps.session.*` aliases on
SUCCESS only (see `internal/events/events.go`). Those are
notification-only topics — they fire AFTER the kit pre-events have
already allowed the op, so they are not veto-able. Subscribers that
need to block must register on the kit pre-topics, not the aps.*
aliases.

## Available context attributes

The CEL `when` expression sees four bindings (see ADR-0008 §4):

| Binding | Type | Notes for aps |
|---------|------|---------------|
| `payload.Op` | string | `"create"`, `"update"`, `"delete"` from `domain.Service` |
| `payload.EntityID` | string | Session id (`"sess-…"`), context-variable key, etc. |
| `payload.Phase` | string | `"pre_validated"` or `"pre_persisted"` |
| `payload.from` / `payload.to` | string | Set on `kit.runtime.state.pre_transitioned` (not yet published by aps) |
| `payload.force` | bool | Set on state transitions (not yet published by aps) |
| `principal.id` | string | Resolved from kit's `DefaultPrincipalResolver` (today: `$USER`). Aps profile lookup is a follow-up — see T-1302 |
| `principal.role` | string | From `KIT_POLICY_ROLE` env var. Empty when unset |
| `context.note` | string | Value passed via `--note\|-n` on every state-changing aps subcommand (T-1291). Aps always sets the key, so an empty value reads as `""` not `unset` |
| `resource.kind`, `resource.fields` | dyn | Per-event payload reflection — see ADR-0008 §4 |

Use `has(context.note)` to check presence; aps always sets the
key, so an empty value reads as `""` not `unset`.

The full list of subcommands that plumb `--note` is in
[cli/reference.md](cli/reference.md); 52 entries spanning profile,
identity, session, workspace, capability, bundle, squad, adapter,
and directory groups.

## Examples

### Block deletion of active sessions

```yaml
policies:
  - name: session-delete-not-active
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || resource.fields.status != "active"'
    effect: allow
    otherwise: deny
    message: "active sessions cannot be deleted; close or terminate first"
```

### Require admin role to delete profiles

```yaml
policies:
  - name: profile-delete-admin-only
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || principal.role == "admin"'
    effect: allow
    otherwise: deny
    message: "only role:admin may delete profiles"
```

Set the role per-shell:

```bash
export KIT_POLICY_ROLE=admin
aps profile delete noor --note "rotated identity to noor-2"
```

### Compose with the default

Aps applies deny-overrides across all matching policies. With both
the default `session-delete-requires-note` and the
`profile-delete-admin-only` rule above active, profile deletes need
both `--note` AND `role:admin`. Order in the file does not matter;
the first denying policy's message is surfaced.

### Note required minimum length

```yaml
policies:
  - name: delete-note-min-length
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || size(context.note) >= 20'
    effect: allow
    otherwise: deny
    message: "delete note must be ≥20 chars; explain why this row goes away"
```

## Authoring custom policies

Kit's `e2e/` directory is the canonical reference. Each test file
is one runnable user story; lift the YAML and CEL idioms from
there:

- `kit/go/runtime/policy/e2e/README.md` — story index + reading order
- `kit/go/runtime/policy/e2e/story_delete_requires_note_test.go` — the
  story aps's default rules were modeled on
- `kit/go/runtime/policy/e2e/story_admin_only_cancel_test.go` —
  role-gated transitions
- `kit/go/runtime/policy/e2e/story_deny_overrides_compose_test.go` —
  multi-policy semantics

Compile-time validation:

- All CEL programs compile at boot. A broken expression fails the
  CLI loud, before any command runs.
- Topic strings are validated against the closed
  `allowedTopics` set in `kit/go/runtime/policy/config.go`. A
  typo'd `on:` field fails parsing with a clear error listing the
  valid topics.

## Exit codes

| Code | When |
|------|------|
| 4 | Policy denied — `PolicyDeniedError` wraps `domain.ErrConflict`, mapped to `output.CodeConflict` (`internal/cli/exit/exit.go`) |

The policy denial is distinguishable from local validation
errors by the message prefix:

```text
Error: policy "<name>" denied: <message>
```

## Operator notes

- The default rules change behavior for `aps session delete` and
  `aps workspace ctx delete`. Old scripts that called these without
  `--note` exit 4 after upgrade. Update them to pass
  `--note "<reason>"`.
- The note is recorded against the bus event payload (e.g.
  `SessionStoppedPayload.Note`, `ProfileDeletedPayload.Note` —
  see `internal/events/events.go`) and surfaces to any subscriber
  that audits aps state changes.
- There is **no operator-visible escape hatch** for the policy
  engine in this version. Once a policy file loads (default or
  user-supplied), enforcement is always-on. The only way to
  loosen enforcement is to edit the user policy file at
  `$XDG_CONFIG_HOME/aps/policies.yaml` and remove or relax the
  rules. Aps does not re-seed once a user file exists, so the
  change persists across upgrades.
- `$APS_POLICY_FILE=/path/to/empty.yaml` (an empty `policies: []`
  file) is the closest thing to a per-shell bypass — useful in CI
  and tests. It is not intended for production use.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `policy: load <path>: ...` at boot | Malformed YAML | Validate the file; aps fails loud rather than ignore |
| `policy: build engine: ...` at boot | Broken CEL in `when` | Compare to ADR-0008 §3 examples and the kit e2e stories |
| `policy "<name>" denied: ...` on every command | A new rule's `otherwise: deny` fires for unrelated events | Tighten the `when` clause so non-target events match and `effect: allow` |
| Custom rule never fires | Topic typo, or aps doesn't publish that topic yet | See "Available topics" — aps publishes the kit entity pre-topics, not state pre-transitioned |
| Policy fires but message is missing | `message:` field absent | Add `message: "..."` — kit falls back to a generic string otherwise |
| `aps session delete` exits 4 after upgrade | Default policy enforced; `--note` missing | Pass `--note "<reason>"`; see `CHANGELOG.md` Unreleased entry |

## See also

- [cli/reference.md](cli/reference.md) — CLI reference, including
  the full `--note` inventory (52 subcommands) and exit codes
- [dev/configuration.md](dev/configuration.md) — XDG config paths
  including where `policies.yaml` lives
- [dev/event-topics.md](dev/event-topics.md) — aps bus topics (the
  `aps.*` notification aliases that fan out alongside the kit
  pre-events)
- kit ADR-0008
  (`~/.w/ideacrafterslabs/kit/hops/main/docs/adr/0008-kit-runtime-policy-engine.md`)
  — engine design, full vocabulary, alternatives
- `internal/config/policies_default.yaml` — the bundled default
  (added in T-1292)
