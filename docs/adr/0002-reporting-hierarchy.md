# ADR 0002 — Reporting hierarchy on profiles

- Status: accepted
- Date: 2026-08-05
- Author: $USER

## Numbering note

This ADR adopts zero-padded numeric prefixes (`NNNN-<slug>.md`) for
`docs/adr/` going forward. The pre-existing `cli-alias-decision.md`
keeps its name and is retroactively counted as 0001; renaming it would
break inbound links for no informational gain. New ADRs continue from
0002.

## Context

aps profiles describe agents (and now people) but carried no notion of
who supervises whom. Fleet operators kept the org structure in their
heads or in out-of-band documents; nothing could validate it, query
it, or render it. This ADR records the decisions behind the optional
reporting hierarchy: a `reports_to` field on `Profile`, a `type:
agent|human` discriminator, a pure graph package
(`internal/core/org`), and the `aps org` command group
(`check` / `show` / `snapshot`).

## Decisions

### (a) `reports_to` is a data relation on the profile

The supervisor link is a single optional field on `Profile`, resolved
into a graph at read time by `internal/core/org` (`Build`, `Chain`,
`DirectReports`, `TransitiveReports`, `Roots`, `Validate` — no I/O,
visited-set guarded, typed `*CycleError`).

Alternatives considered:

- **Separate org-edges file** (e.g. `<data-dir>/org.yaml` holding
  manager→report pairs). Rejected: a second source of truth that can
  drift from the profiles it describes; every profile create / delete /
  rename would need a paired edge-file mutation, and shipping a
  profile bundle would no longer carry its org position.
- **Extending squads.** Rejected: squads are the Team-Topologies axis
  (delivery grouping), orthogonal to reporting lines — one report can
  serve several squads and a squad has members from several reporting
  chains. Squad "manager" is also a non-persistent runtime notion;
  overloading it would freeze a scheduling detail into identity data.

Write-time validation is strict: the target must exist, no
self-reference, no cycle. Manifest **import** keeps the same path but
its dangling-target failure is `--force`-downgradable to a warning
(bulk imports may bring the supervisor later); cycle rejection is
never downgradable — cyclic data on disk breaks every consumer.

### (b) Humans are type-discriminated profiles

A manager who is a person is a profile with `type: human`. Allowed
values: `agent` (default, empty means agent) and `human`.

Alternatives considered:

- **Contact URIs in `reports_to`** (point at the contact/vCard store).
  Rejected: makes humans second-class graph nodes — they could not be
  traversed, validated, or rendered like any other node, and every
  graph consumer would need a second resolution path.
- **Overloading Roles.** Rejected: roles carry task-role semantics
  (what a profile does in a flow), not what a profile *is*.

Relationship to the contact vCard store: the same person may exist as
both a human profile (graph node) and a contact (address-book entry).
The duplication is accepted; linking the two records is deferred until
a concrete consumer needs it.

`type` validation is write-strict, read-tolerant: `create`/`edit`
reject anything outside `agent|human`, but the loader still loads an
on-disk profile with an unknown `type` — a silent-skip loader would
make a typo'd profile vanish from `list`/`show` with no diagnostic.
`aps org check` is the surface that reports unknown types (and
unloadable profile files). Human profiles are representable, not
executable: `aps run` and session start reject them; a2a card serving
is not blocked.

### (c) Snapshots are derived on demand, never stored

`aps org snapshot` computes its output from the profiles on disk at
invocation time. No server-side or cached snapshot artifact exists —
the profiles are the single source of truth, and a stored snapshot
would be stale the moment a profile changed. Persistence is the
operator's choice via the global `--output` flag (atomic temp-file +
rename write). All formats are deterministic (sorted nodes and edges);
`tree` and `mermaid` carry no timestamp so unchanged state diffs
clean, while `json`/`yaml` carry `captured_at` as document metadata.

### (d) Channels are annotations; the hierarchy transports nothing

Org views annotate each node with its reachable channels (`a2a`,
`acp`, `email`, `webhooks`), derived purely from profile config
(`org.Channels`). The hierarchy itself never routes or transports
messages. A2A remains the agent-to-agent channel — including its
documented supervisor/subordinate interaction pattern — and adapters
(email, messengers) remain the way to reach humans. Messenger links
are deliberately excluded from the channel summary: they live in
per-profile `LinkStore` files, not on the `Profile` struct, and
probing them would break the pure-function contract.

### (e) Write-strict / read-tolerant type validation

Recorded as its own decision because both halves are load-bearing:
strict writes keep bad values from entering the system through aps
itself; tolerant reads keep hand-edited or foreign-written files
visible so the operator can find and fix them (via `aps org check`)
instead of wondering where a profile went.

### (f) Delete guards inbound reporters; `--force` overrides

`aps profile delete` refuses to delete a profile that others report
to, listing the reporters; `--force` proceeds with a warning that
their `reports_to` will dangle.

Alternatives considered:

- **Block outright** (no override). Rejected: the operator may
  intend the dangling state (e.g. mid-restructure) and `aps org
  check` surfaces it afterwards.
- **Cascade** (clear or re-point reporters' `reports_to`). Rejected:
  silent mutation of profiles the operator did not name; a delete
  should never edit other files.

## Consequences

- `reports_to` changes reuse the existing `aps.profile.updated` event
  with `fields: ["reports_to"]` — no new event type for consumers to
  learn.
- All org semantics live in `internal/core/org`; the CLI layer only
  loads profiles, projects rows, and renders. New consumers (server
  endpoints, TUI) reuse the same pure graph.
- Cyclic on-disk data (written by hand or by older binaries) renders
  where possible and exits non-zero; nothing hangs.
