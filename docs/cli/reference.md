# Aps CLI reference

This page documents global flags, exit codes, and the `--note|-n`
flag inventory for state-changing subcommands.

## Global flags

These flags are available on every aps subcommand:

| Flag | Description |
|------|-------------|
| `--profile` | Active profile id (overrides `$APS_PROFILE`) |
| `--format <text\|json\|yaml>` | Output format for list / show commands |
| `--quiet` | Suppress progress / non-essential stderr |
| `--no-redact` | Disable kit/core/redact egress filter (audit only) |

## Exit codes

Aps maps `domain` errors to kit's standard exit codes via
`internal/cli/exit/exit.go`:

| Code | Source | Meaning |
|------|--------|---------|
| 0 | — | Success |
| 1 | Generic error | Any unmapped error |
| 2 | `domain.ErrInvalidArgument` | Bad flag, missing required arg, validation failed |
| 3 | `domain.ErrNotFound` | Profile / session / capability does not exist |
| 4 | `domain.ErrConflict`, `policy.PolicyDeniedError` | Uniqueness violation OR policy veto |
| 5 | `domain.ErrPermissionDenied` | ACL / scope check failed |
| 64 | `output.CodeRateLimited` | Factor-10 max-ops budget exceeded |

Policy denials wrap `ErrConflict`, so they share exit 4 with
ordinary uniqueness conflicts. The message prefix
(`policy "<name>" denied: …`) distinguishes them. See
[../policies.md](../policies.md) for details.

## `--note|-n` flag

Every state-changing subcommand exposes `--note|-n`. The value is
recorded against the bus event payload AND attached to
`context.Context` via `policy.ContextAttrsKey` so kit's policy
engine can read it as `context.note` from CEL during
`pre_validated` / `pre_persisted` veto evaluation.

**Always set it.** Some defaults (see
[../policies.md](../policies.md)) reject destructive operations
without a note and exit 4.

```bash
aps session delete sess-7c41 --note "stale; profile retired"
aps workspace ctx delete CHANNEL_ID -n "redacted PII"
aps profile delete noor -n "rotated identity to noor-2"
```

The shorthand `-n` resolves to `--note` on commands where it is not
already taken by another flag (a small number of legacy commands
reserve `-n` for `--dry-run` or `--limit`; on those, only the long
`--note` form is registered).

### Inventory (52 subcommands)

Source of truth: `scripts/verify_note_flag.sh`. Every entry below
exits 0 from that script after T-1291.

#### Profile (identity)

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps profile create` | `--note` | `-n` |
| `aps profile edit` | `--note` | `-n` |
| `aps profile delete` | `--note` | `-n` |
| `aps profile import` | `--note` | `-n` |
| `aps profile capability add` | `--note` | `-n` |
| `aps profile capability remove` | `--note` | `-n` |
| `aps profile workspace set` | `--note` | `-n` |

#### Identity

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps identity init` | `--note` | `-n` |

#### Sessions

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps session attach` | `--note` | `-n` |
| `aps session detach` | `--note` | `-n` |
| `aps session delete` | `--note` | `-n` |
| `aps session terminate` | `--note` | `-n` |

#### Workspaces and context

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps workspace create` | `--note` | `-n` |
| `aps workspace remove` | `--note` | `-n` |
| `aps workspace archive` | `--note` | `-n` |
| `aps workspace join` | `--note` | `-n` |
| `aps workspace leave` | `--note` | `-n` |
| `aps workspace role` | `--note` | `-n` |
| `aps workspace use` | `--note` | `-n` |
| `aps workspace send` | `--note` | `-n` |
| `aps workspace sync` | `--note` | `-n` |
| `aps workspace ctx set` | `--note` | `-n` |
| `aps workspace ctx delete` | `--note` | `-n` |
| `aps workspace policy` | `--note` | `-n` |
| `aps policy set` | `--note` | `-n` |

#### Capabilities and bundles

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps capability adopt` | `--note` | `-n` |
| `aps capability watch` | `--note` | `-n` |
| `aps capability link` | `--note` | `-n` |
| `aps capability delete` | `--note` | `-n` |
| `aps capability install` | `--note` | `-n` |
| `aps capability enable` | `--note` | `-n` |
| `aps capability disable` | `--note` | `-n` |
| `aps bundle create` | `--note` | `-n` |
| `aps bundle edit` | `--note` | `-n` |
| `aps bundle delete` | `--note` | `-n` |

#### Multi-agent (squads, adapters)

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps squad create` | `--note` | `-n` |
| `aps squad delete` | `--note` | `-n` |
| `aps squad members add` | `--note` | `-n` |
| `aps squad members remove` | `--note` | `-n` |
| `aps adapter create` | `--note` | `-n` |
| `aps adapter attach` | `--note` | `-n` |
| `aps adapter detach` | `--note` | `-n` |
| `aps adapter link add` | `--note` | `-n` |
| `aps adapter link delete` | `--note` | `-n` |
| `aps adapter approve` | `--note` | `-n` |
| `aps adapter reject` | `--note` | `-n` |
| `aps adapter revoke` | `--note` | `-n` |
| `aps adapter pair` | `--note` | `-n` |
| `aps adapter start` | `--note` | `-n` |
| `aps adapter stop` | `--note` | `-n` |

#### AGNTCY directory

| Subcommand | Long form | Short |
|------------|-----------|-------|
| `aps directory register` | `--note` | `-n` |
| `aps directory delete` | `--note` | `-n` |

### Verifying the inventory

CI runs `scripts/verify_note_flag.sh` against a freshly built aps
binary; it greps `--help` output for every entry above and exits
non-zero if any are missing the flag. To run locally:

```bash
go build -buildvcs=false -o ./aps_t1291 ./cmd/aps/
APS_BIN=./aps_t1291 bash scripts/verify_note_flag.sh
```

When introducing a new state-changing subcommand, add it to the
`SUBCOMMANDS` array in the script AND to the table above.

## See also

- [../policies.md](../policies.md) — policy engine, default rules,
  and how `context.note` plumbs from `--note` into CEL
- [redaction.md](redaction.md) — kit/core/redact wiring at the
  output and logger boundaries
- [progress-inventory.md](progress-inventory.md) — long-running
  ops that emit `kit/console/progress` events
