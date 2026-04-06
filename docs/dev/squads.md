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

# Validate topology (8-item checklist)
aps squad check
```

### Membership

```bash
# Add profile to squad
aps squad members add <squad-id> <profile-id>

# Remove profile from squad
aps squad members remove <squad-id> <profile-id>
```

Profiles record their squad membership in `profile.yaml`:
```yaml
squads:
  - squad-id-1
  - squad-id-2
```

### Export / Import

```bash
# Export squad and all member profiles as a gzip tarball
aps squad export <squad-id> --output squad.tar.gz

# Import a squad bundle
aps squad import squad.tar.gz
```

## Contracts

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

Results are sorted by score. Use `--type` to filter by squad type, `--mode` to filter by required interaction mode.

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
| `stale-collaboration` | Collaboration running too many cycles | >4 cycles |
| `stale-enabling` | Enabling attachment too long | >6 cycles |
| `narrow-interface` | Subsystem called too heavily per task | <5 calls/task |
| `interface-churn` | Subsystem interface changing frequently | >3 changes |

These thresholds are configurable via `EvolutionThresholds`.

## Topology Checklist (`aps squad check`)

Validates the entire topology against 8 criteria:

| Check | What it verifies |
|-------|----------------|
| `stream-independent` | Stream squads can complete work without cross-squad coordination |
| `subsystem-interface-clean` | Complicated subsystem squads have XaaS contracts |
| `enabling-exit-defined` | All enabling squads have exit conditions |
| `platform-golden-path` | Platform squads have `golden_path_defined: true` |
| `collab-timeboxed` | All collaboration contracts are time-boxed |
| `no-platform-internals` | No stream-to-platform collaboration contracts |
| `routing-type-aware` | Router is using declared contracts (not capability-only) |
| `topology-first` | Squad context loads are well-scoped (coordination ratio < 0.5) |

```bash
$ aps squad check
CHECK                      STATUS  DETAIL
stream-independent         PASS
subsystem-interface-clean  FAIL    squad "payments" has no XaaS contract
enabling-exit-defined      PASS
platform-golden-path       PASS
collab-timeboxed           PASS
no-platform-internals      PASS
routing-type-aware         PASS
topology-first             PASS

7/8 checks passed
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
| `internal/core/squad/export.go` | Tarball export/import |
| `internal/cli/squad/cmd.go` | CLI entry points |
| `internal/cli/squad/check.go` | `aps squad check` |
| `internal/cli/squad/create.go` | `aps squad create` |
