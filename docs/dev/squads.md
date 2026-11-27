# Squads — Implementation Reference

This document covers the APS implementation of Squad Topologies. For the underlying theory, see [squad-topologies-spec.md](squad-topologies-spec.md).

## What is a Squad?

A squad is an autonomous agent team unit. It owns a domain, has member profiles, and interacts with other squads via declared contracts. Squad shape is designed first; agent communication follows.

## Squad Types

| Type | `--type` flag | Owns Domain | Interaction Mode |
|------|--------------|-------------|-----------------|
| Stream-Aligned | `stream-aligned` | Yes | Consumes XaaS |
| Enabling | `enabling` | No (temporary) | Facilitating |
| Complicated Subsystem | `complicated-subsystem` | Yes (narrow) | Provides XaaS |
| Platform | `platform` | Yes (infra) | Provides XaaS |

## CLI Reference

### Managing Squads

```bash
# List all squads
aps squad list

# Create a squad
aps squad create <name> --type stream-aligned --domain <domain> [--description "..."] [--members profile1,profile2]

# Show squad details
aps squad show <squad-id>

# Delete a squad
aps squad delete <squad-id>

# Validate all squads against the 8-item checklist (takes no arguments)
aps squad check
```

### Membership

```bash
# Add profile to squad
aps squad members add <squad-id> <profile-id>

# Remove profile from squad
aps squad members remove <squad-id> <profile-id>
```

`members add|remove` edit the squad's manifest only (see Storage below);
profile records are not touched. The squad's `members` list is what
`aps squad show`, `list --member` and `check` read.

`profile.yaml` has its own optional `squads:` list. It is a separate field,
edited with the profile, and is what `aps profile list --squad <id>` and
`aps org snapshot --squad <id>` read:
```yaml
squads:
  - squad-id-1
  - squad-id-2
```

### Storage

Squads persist under the data dir, one directory per squad, like profiles
(`profiles/<id>/profile.yaml`) and capabilities (`capabilities/<name>/manifest.yaml`):

```
$APS_DATA_PATH/squads/<id>/manifest.yaml     # default: ~/.local/share/aps
```

```yaml
id: demo-squad
name: Demo Squad
type: platform
domain: core
members:
    - alice
    - bob
created_at: 2026-11-27T06:04:43.119964-05:00
updated_at: 2026-11-27T06:04:43.119964-05:00
```

- Every command reads the store afresh, so separate invocations agree.
- A manifest is replaced atomically (temp file in the squad dir, synced,
  renamed), so a crash leaves the old or the new record, never a partial one.
- There is no cross-process lock: two commands changing the same squad at
  the same instant can lose one of the changes (last writer wins). The
  manifest is never left corrupt.
- Member IDs are stored verbatim; they are not checked against the profile store.
- IDs must be safe directory names on every platform: no path separators or
  `..`, no `: * ? " < > |`, no control characters, no trailing dot or space,
  no Windows device names (`con`, `nul`, `com1`, ...), at most 255 bytes.
- A manifest that cannot be read, or whose `id` differs from its directory
  name, is reported with its file path. `list` skips it and names it on
  stderr; `show <id>` and mutations of that squad fail with the error;
  `check` fails rather than validate a partial topology; `delete <id>` removes it.

`--format json|yaml` on `show` prints one squad object (`id`, `name`, `type`,
`domain`, `description`, `members`, `golden_path_defined`, `created_at`,
`updated_at`; `members` is `[]` when empty; `scope` appears when set).
`list` prints an array of summary rows. `create` and `members add|remove`
print the resulting squad object; `delete` prints `{"id": ..., "deleted": true}`.

## Contracts

Contracts, the router, exit conditions, timeboxes, evolution signals and
context load are Go library types in `internal/core/squad`. There are no
`aps squad` verbs for them and nothing persists them: contracts live in an
in-memory `ContractStore`.

Contracts declare the interaction between two squads. They are required before squads communicate.

**Contract fields:**
```
provider_squad    which squad provides the capability
consumer_squad    which squad consumes it
mode              x-as-a-service | collaboration | facilitating
version           semver string
input_schema      JSON schema for inputs
output_schema     JSON schema for outputs
sla.max_latency   maximum acceptable latency
sla.availability  target availability (0.0–1.0)
deprecation_window  notice period before breaking changes
timebox           required for collaboration contracts
exit_condition    required for facilitating contracts
```

**Validation rules:**
- All contracts must specify provider, consumer, and mode
- `collaboration` contracts **must** have a `timebox`
- `facilitating` contracts **must** have an `exit_condition`
- `x-as-a-service` contracts should define `input_schema` and `output_schema`

## Router

The router matches work to the right squad based on domain and type.

```go
// Score breakdown:
// +10  domain match
// +5   stream-aligned squad
// +3   complicated-subsystem squad
// +2   platform squad
```

Results are sorted by score. `RouteRequest.Domain` is required; `RequiredType` filters by squad type. `RequiredMode` is accepted but does not yet filter.

## Exit Conditions

Exit conditions track when an enabling squad engagement is complete.

```
squad_id          the enabling squad
target_squad      the stream squad being enabled
criteria          what does "done" look like?
deadline          when must this be complete?
handoff_artifacts  documentation, tooling, prompts left behind
```

An engagement is overdue when `deadline` has passed and `completed_at` is unset.

## Timebox

Timeboxes track the lifecycle of collaboration contracts.

```
contract_id          the collaboration contract
started_at           when collaboration began
duration             agreed length (e.g. 2 weeks)
graduation_target    what mode this graduates to (usually x-as-a-service)
graduated / graduated_at  completion state
```

A timebox is **active** when it has not graduated and has not expired.

## Evolution Signals

The evolution monitor watches for patterns that indicate the topology should change.

| Signal | Meaning | Threshold |
|--------|---------|-----------|
| `stale_collaboration` | Collaboration running too many cycles | >4 cycles |
| `stale_enabling` | Enabling attachment too long | >6 cycles |
| `narrow_interface` | Subsystem called too heavily per task | <5 calls/task |
| `interface_churn` | Subsystem interface changing frequently | >3 changes |

These thresholds are configurable via `EvolutionThresholds`.

## Topology Checklist (`aps squad check`)

Validates every stored squad (`$APS_DATA_PATH/squads`) against 8 criteria:

| Check | What it verifies |
|-------|----------------|
| `stream-independent` | Every stream-aligned squad has at least one member |
| `subsystem-interface-clean` | Complicated subsystem squads have XaaS contracts |
| `enabling-exit-defined` | All enabling squads have exit conditions |
| `platform-golden-path` | Platform squads have `golden_path_defined: true` |
| `collab-timeboxed` | All collaboration contracts are time-boxed |
| `no-platform-internals` | No collaboration contract has a platform squad as provider |
| `routing-type-aware` | A topology of several squads declares at least one contract |
| `topology-first` | Squad context loads are well-scoped (coordination ratio < 0.5) |

`aps squad check` supplies only the stored squads. Contracts, exit conditions
and context loads are not persisted, so it passes empty lists for them. In
effect, `subsystem-interface-clean` fails for any complicated-subsystem squad,
`enabling-exit-defined` for any enabling squad, and `routing-type-aware` as soon
as there is more than one squad. `collab-timeboxed`, `no-platform-internals` and
`topology-first` have nothing to evaluate and pass. The command exits non-zero
when any check fails.

```bash
$ aps squad check
CHECK                      STATUS  DETAIL
stream-independent         PASS    -
subsystem-interface-clean  FAIL    subsystem squad "sub" has no XaaS contract as provider
enabling-exit-defined      PASS    -
platform-golden-path       FAIL    platform squad "nomem" missing golden path definition
collab-timeboxed           PASS    -
no-platform-internals      PASS    -
routing-type-aware         FAIL    multiple squads but no interaction contracts declared
topology-first             PASS    -

5/8 checks passed
```

## Context Load

Context load measures how much an agent must hold in working memory:

```
Total KB = (tool_schemas × 2KB) + domain_knowledge_KB + (interaction_protos × 1KB) + session_memory_KB
Coordination KB = (interaction_protos × 1KB) + session_memory_KB
Coordination ratio = coordination_KB / total_KB
```

A squad is **well-scoped** when `coordination_ratio < 0.5`. If most context is coordination overhead rather than domain knowledge, the topology needs redesign.

## Key Files

| File | Purpose |
|------|---------|
| `internal/core/squad/types.go` | Core types: `Squad`, `SquadType` |
| `internal/core/squad/manager.go` | CRUD and membership: `Create`, `Get`, `List`, `AddMember`, `GetSquadsForProfile` |
| `internal/core/squad/contract.go` | Contract type and validation |
| `internal/core/squad/router.go` | Domain-aware routing: `Router.Route()` |
| `internal/core/squad/exit_condition.go` | `ExitCondition`, `IsComplete()`, `IsOverdue()` |
| `internal/core/squad/timebox.go` | `Timebox`, `IsActive()`, `Graduate()`, `Remaining()` |
| `internal/core/squad/evolution.go` | `EvolutionMonitor`, signals, thresholds |
| `internal/core/squad/checklist.go` | `ValidateTopology()` → `[]CheckResult` |
| `internal/core/squad/context_load.go` | `ContextLoad`, `CoordinationRatio()`, `IsWellScoped()` |
| `internal/core/squad/export.go` | Tarball bundle helpers (library only; no CLI verb) |
| `internal/cli/squad/cmd.go` | CLI entry points |
| `internal/cli/squad/check.go` | `aps squad check` |
| `internal/cli/squad/create.go` | `aps squad create` |
