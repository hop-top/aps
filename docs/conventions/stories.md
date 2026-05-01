# Story → e2e test linkage (v2)

Author: $USER
Status: active
Version: story-e2e linkage convention v2
Updated: 2026-04-30

Cross-repo convention. Every user story under `docs/stories/` MUST link
its acceptance criteria to concrete e2e tests by `file::function`. No
prose hand-waves, no "TBD", no missing section.

Adopted across: aps, ctxt, tlc, c12n, routellm.

## What changed in v2

v2 EXTENDS v1 (no rules removed). New: rule 8 `fixture:` prefix;
rule 9 multi-language `### E2E (Go|Python|Rust)`; rule 10 test-plan
class with `## Test Plan`; canonical audit script at
`~/.agents/scripts/audit-stories.sh`; rule 7 `status:` softened to
WARN (backfill is a separate task).

## The 10 rules

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
7. Frontmatter `status:` field on every story (warn, not fail in v2):
   - `shipped` — code present, e2e green.
   - `shipped-no-e2e` — code present, e2e missing/broken/planned.
   - `partial` — code present, only some criteria met.
   - `paper` — story written, no impl.
8. **NEW**: Fixture-driven tests use `fixture:` prefix instead of
   `file::function`. Path points to a fixture directory or YAML file
   that the test runner consumes. See worked example below.
9. **NEW**: Multi-language repos MAY group entries under language
   subheadings: `### E2E (Go)`, `### E2E (Python)`, `### E2E (Rust)`,
   `### E2E (Node)`, `### E2E (TypeScript)`, `### E2E (JS)`. Each
   subheading is validated independently (≥ 1 conforming entry).
10. **NEW**: Test-plan docs (e.g. `user-stories-taskflow-0.1.md`) are
    a separate class. Title contains "Test Plan" or filename matches
    `*test-plan*`/`*taskflow*`. Use `## Test Plan` section, not
    `## E2E Tests`. Entries follow same format (planned: / fixture: /
    file::function).

## Worked examples

### v1: `file::function` (still valid)

```markdown
## E2E Tests

- `tests/e2e/webhook_test.go::TestWebhookServer`
- planned: `tests/e2e/webhook_test.go::TestWebhookRetry`
```

### v2: `fixture:` prefix

For tests driven by a fixture runner (e.g. `tlc flow test <runner> <dir>`):

```markdown
## E2E Tests

### E2E
- `tests/e2e/flow_single_step_test.go::TestFlow_SingleStep_HappyPath`
- fixture: `examples/flows/fixtures/single-step/happy-path/`
- fixture: `examples/flows/fixtures/single-step/error-path/`
```

The fixture path is a directory containing the test inputs (`test.yaml`,
`record/` cassettes, etc.) consumed by the runner. The runner itself is
referenced once at the top via `file::function`.

### v2: multi-language sections

For polyglot repos (e.g. c12n: Go + Python + Rust):

```markdown
## E2E Tests

### E2E (Go)
- `e2e_test.go::TestGoBinding_NewPipelineInitOK`
- planned: `e2e_test.go::TestGoBinding_EvaluateRespectsTimeout`

### E2E (Python)
- `tests/e2e/test_python_bindings.py::test_pipeline_init`
- planned: `tests/e2e/test_python_bindings.py::test_evaluate_timeout`

### E2E (Rust)
- planned: `tests/e2e/rust_api_test.rs::test_pipeline_evaluate`
```

Each language section is validated independently. Audit reports
counts per section.

### v2: test-plan doc

For a doc that is a test plan, not a feature story:

```markdown
# E2E Test Plan — Task Flow (TFS) v0.1

## Test Plan

- planned: `tests/e2e/taskflow_v0_1_test.go::TestE2E01_SequentialOrdering`
- planned: `tests/e2e/taskflow_v0_1_test.go::TestE2E02_DependencyGate`
```

Audit recognizes by title prefix or filename pattern. No `status:`
required (test plans are not features).

## Non-conforming patterns to fix

- `## E2E Tests \n > Not yet implemented.` — no path, no function.
- `## E2E Test Checklist` listing prose scenarios with no test names.
- Story missing the section entirely.
- Linking only to an issue tracker ID (`see #1234`) with no test path.
- Prose `❌ NOT YET IMPLEMENTED` without `planned:` prefix and path.

When fixing: add the section if missing; convert prose checklist items
into `planned:` entries with a committed test path; do NOT delete the
acceptance criteria — map them.

## Authoring flow

1. Write the story (goal, context, acceptance criteria).
2. For each acceptance criterion, decide e2e or `unit-only`/`manual-only`.
3. For each e2e criterion, pick a test path + function name. If the
   test exists, link it. If not, prefix `planned:`. If fixture-driven,
   use `fixture:` plus a runner reference.
4. Set frontmatter `status:` honestly. `paper` is fine; lying is not.
5. When impl lands and the test goes green, drop `planned:` and bump
   `status:` to `shipped`.

## Verification

Use the canonical audit script:

```bash
~/.agents/scripts/audit-stories.sh [REPO_ROOT]
```

Exit 0 = all PASS or only WARN. Exit 1 = at least one FAIL.

A story is conformant iff:

- File contains `## E2E Tests`, `### E2E`, `### E2E (<lang>)`, or
  `## Test Plan` section.
- Section has ≥ 1 entry. Entry forms accepted:
  - `<path>::<func>`
  - `<path> — <func>` / `<path> -- <func>`
  - `[path](url) — <descr>` (markdown-link form)
  - `planned:` or `fixture:` prefix
  - sub-bullet `Test*` names under a parent file bullet
  - table row containing `\`Test*\``
- Frontmatter `status:` field present (WARN if missing, not FAIL).

## Why

Stories without test linkage drift. Forcing `file::function` per
criterion makes the spec machine-checkable. v2 adds fixture-style
and multi-language support so repos with these realities (tlc,
c12n) don't paper over them with shadow `planned:` entries.

## Migration from v1

If a repo is on v1 and has audit drift:

1. Run `~/.agents/scripts/audit-stories.sh` from the repo root.
2. Fix any `FAIL` (missing section, empty section, prose-only).
3. `WARN` lines (missing `status:`) are deferred — file a follow-up
   task to backfill `status:` per story. v2 does not block on this.
4. For tlc-style fixture entries: rewrite shadow `planned:` entries
   into `fixture:` form pointing at the fixture dir.
5. For c12n-style polyglot stories: split a single `### E2E` block
   into `### E2E (Go)` / `### E2E (Python)` / `### E2E (Rust)`.
6. For test-plan docs (e.g. `*-test-plan.md`, `taskflow-*.md`):
   rename `## E2E Tests` to `## Test Plan` and drop `status:`.

New in v2 (recap): `fixture:` prefix, multi-language `### E2E (X)`
sections, test-plan class, canonical audit script.

## Scope

This document codifies the rule. It does NOT retrofit existing
stories — that is per-repo follow-up work, tracked separately.
