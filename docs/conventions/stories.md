# Story → e2e test linkage

Author: $USER
Status: active

Cross-repo convention. Every user story under `docs/stories/` MUST link
its acceptance criteria to concrete e2e tests by `file::function`. No
prose hand-waves, no "TBD", no missing section.

Adopted across: aps, ctxt, tlc, c12n, routellm.

## The 7 rules

1. Every story file MUST have an `## E2E Tests` section. No exceptions.
2. Listed tests MUST be concrete `file::function` references — repo-
   relative path + test function name.
   Example: `tests/e2e/federation/dag_chain_test.go::TestDAGChain_3HopPropagation`.
3. If a test does not yet exist: prefix the entry with `planned:` —
   the file path + function name are still required. Path is the path
   the test will live at; name is the function the author commits to.
4. "Not yet implemented" prose with no path is non-conforming. Same
   for an empty section, or a section that only links to a tracker
   ticket.
5. Acceptance-criteria checkboxes map 1:1 to tests, OR are explicitly
   marked `unit-only` / `manual-only` inline. Untagged checkboxes
   default to e2e-required.
6. Failing tests are acceptable. A red e2e is more honest than a
   missing one — it surfaces doc-rot vs. real impl gaps. CI may skip
   `planned:` entries; it must run resolved ones.
7. Frontmatter `status:` field on every story:
   - `shipped` — code present, e2e green.
   - `shipped-no-e2e` — code present, e2e missing/broken/planned.
   - `partial` — code present, only some criteria met.
   - `paper` — story written, no impl.

## Worked example

Concrete reference: `aps/docs/stories/004-webhook-server.md` — short,
two acceptance scenarios, single `## Tests` block with an `### E2E`
subsection naming a real Go test.

```markdown
## Tests

### E2E
- `tests/e2e/webhook_test.go` — `TestWebhookServer`
```

`planned:` form, when impl lags spec (routellm pattern):

```markdown
## E2E tests

- planned: `routellm/tests/test_auth.py::test_request_without_token_401`
- planned: `routellm/tests/test_auth.py::test_token_revoke_blocks_next_call`
```

Both forms are conformant. Mixed lists (some shipped, some `planned:`)
are fine.

## Non-conforming patterns to fix

These are the shapes a retroactive backfill must replace:

- `## E2E Tests \n > Not yet implemented.` — no path, no function.
- `## E2E Test Checklist` listing prose scenarios with no test names.
- Story missing the section entirely.
- Linking only to an issue tracker ID (`see #1234`) with no test path.

When fixing: add the section if missing; convert prose checklist items
into `planned:` entries with a committed test path; do NOT delete the
acceptance criteria — map them.

## Authoring flow

1. Write the story (goal, context, acceptance criteria).
2. For each acceptance criterion, decide e2e or `unit-only`/`manual-only`.
3. For each e2e criterion, pick a test path + function name. If the
   test exists, link it. If not, prefix `planned:`.
4. Set frontmatter `status:` honestly. `paper` is fine; lying is not.
5. When impl lands and the test goes green, drop `planned:` and bump
   `status:` to `shipped`.

## Verification

A story is conformant iff:

- File contains an `## E2E Tests` (or `### E2E` under `## Tests`) section.
- Section has ≥ 1 entry, each entry matches `<path>::<func>` or
  `<path> — <func>` form, optionally prefixed `planned:`.
- Frontmatter has a `status:` field with one of the 4 values.

A simple grep gate: a story without `E2E` and without `status:` in
the first 20 lines is non-conforming.

## Why

Stories without test linkage drift. Six months in, nobody knows
which acceptance criteria are real, which are aspirational, which
broke silently. Forcing `file::function` per criterion makes the
spec machine-checkable: CI can assert every story has at least one
test referenced, and every referenced test exists (or is `planned:`).

The convention is cheap to write, cheap to enforce, and prevents the
class of bugs where docs say "shipped" and code says otherwise.

## Scope

This document codifies the rule. It does NOT retrofit existing
stories — that is per-repo follow-up work, tracked separately.
