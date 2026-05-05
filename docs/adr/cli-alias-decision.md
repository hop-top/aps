# ADR — kit/console/alias adoption

- Status: superseded — adopted 2026-05-02 in commit `6384c04` (T-0388)
- Date: 2026-04-29
- Task: T-0379
- Author: $USER

## Supersession note (2026-05-04, T-0458)

The "Defer" decision below is **no longer in force**. kit/console/alias
was adopted in aps three days after this ADR landed:

- Adoption commit: `6384c04` — `fix(cli): parity re-check round, kit
  adoption (T-0388, T-0392, T-0398)`
- Wiring lives in `internal/cli/alias.go` (`alias.NewStore(...)`,
  `root.AliasCmd(store)`); legacy shell-alias generator moved to
  `aps alias shell` exactly as the "Notes for future adoption"
  section anticipated.

The gap was surfaced by the kit-integration audit on 2026-05-04, which
found the original ADR still claimed "Defer" while the code already
imported `hop.top/kit/go/console/alias`. This block records the
correction; the body below is preserved verbatim as a historical
artifact only.

**Process lesson:** decision-now-adopt-later ADRs must be updated on
adoption (status flip + commit link). The same loop that lands the
adopting code should touch the ADR.

## Context

Two distinct alias surfaces exist in the aps codebase neighbourhood:

| Surface              | Where                          | What it does                                                         |
|----------------------|--------------------------------|----------------------------------------------------------------------|
| `aps alias` (today)  | `internal/cli/alias.go`        | Generates shell aliases (`alias <profile>='aps <profile>'`) for `eval` |
| `kit/console/alias`  | `vendor/hop.top/kit/go/console/alias` + `cli.Root.Alias / LoadAliasStore` | Runtime alias resolver — user-defined CLI shorthands like `aps c → aps capability list`, persisted in a JSON store |

They are orthogonal: the existing generator emits POSIX/Powershell
shell-level shorthands keyed off discovered profiles; the kit
primitive registers in-process cobra aliases keyed off user
preference. Adopting kit/console/alias would not displace
`aps alias`.

## Survey

- ctxt and tlc do not currently adopt kit/console/alias either
  (rg of `hops/main` returns zero hits in both repos as of
  2026-04-29). aps would be first.
- kit/console/alias is fully built: `Store` (JSON persistence),
  `Set/Remove/Get/All/Expand`, and `Root.Alias / LoadAliasStore`
  on the kit CLI side. Wiring is one call in `init`.

## Decision

Defer.

## Rationale

- No user has asked for runtime aliases on `aps`. The closest
  request — typing `aps c` instead of `aps capability list` —
  is already partly served by cobra's prefix matching for
  unique subcommand prefixes.
- aps's sibling tools (ctxt, tlc) haven't adopted it; better to
  let one of them pioneer the UX and copy the wiring once it's
  battle-tested in user hands.
- Cost of waiting is near-zero: kit/console/alias is small,
  stable, additive — adoption later is a 5-line `init` patch
  plus a top-level `aps alias-store` command for management.
- Adopting now adds a config surface (`aliases:` map in
  `config.yaml`) and an alias-management command that would
  ship under-used and confuse users vs. the existing shell-
  alias generator (also called `alias`).

## Revisit when

- A user file requests runtime alias support for aps subcommands.
- ctxt or tlc adopts kit/console/alias — copy their wiring
  (config key, command names) to keep cross-tool consistency.
- aps grows past ~20 top-level subcommands and prefix matching
  becomes ambiguous.

## Notes for future adoption

- Rename `aliasCmd` (the shell-alias generator) to `shellAliasCmd`
  with `Use: "shell-alias"` and keep `alias` as a back-compat
  alias for one release. Then mount `kit/console/alias` under
  the freed `alias` name.
- Persist runtime aliases at `<APS_DATA_PATH>/aliases.json` via
  `alias.NewStore`, not viper config — keeps profile-scoped
  state out of the global config file.
- Wire `root.LoadAliasStore(store)` after subcommand registration
  in `cli.Execute`, before `root.Execute`.
