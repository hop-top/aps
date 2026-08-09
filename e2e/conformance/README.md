# Conformance grading (12fcc service tier)

Scenario library, recorded cassettes, and captured verdicts for the
service-graded factors of the 12-Factor AI-CLI conformance contract:
F1 Capability Introspection, F3 Structured I/O, F4 Corrective Error
Model, F5 Explicit Contracts, F6 Previewability, F7 Idempotency, F8
State Transparency, F10 Delegation Safety, and F11 Exit Code
Semantics. The story-graded factors (F2, F9, F12) and leak scanning
are covered by the `12fcc` workflow over `e2e/stories/`.

## Layout

```
scenarios/aps/<id>/1.0.0/scenario.yaml   graded rubric, one per scenario
stories/<id>.yaml                        user story each scenario is bound to
cassettes/<id>/                          captures from real binary runs
  manifest.yaml                          upload manifest (story hash, steps)
  story.yaml                             byte-exact copy of the story
  steps/<step-id>/result.json            exit code + duration as observed
  steps/<step-id>/stdout.txt             stdout as observed
  steps/<step-id>/stderr.txt             stderr as observed
verdicts/<id>.json                       grading service verdict, tier 3
```

Cassettes are recorded, never authored: every stdout/stderr byte and
exit code comes from executing the actual binary via `kit conformance
harness record`. Scenarios encode the spec-correct expectation even
where current behavior violates it — a failing verdict is the honest
outcome, not a broken pipeline. Re-records churn only `recorded_at`,
`binary_version`, and `duration_ms`; the captures themselves are
byte-stable.

## Isolation and secrets

aps manages profiles, secret files, and bus credentials, so a cassette
recorded against a real environment would bake personal data into a
committed artifact. `scripts/12fcc-record.sh` runs every scenario
with:

- its own `HOME` and `APS_DATA_PATH` under a throwaway work dir, so
  profile writes never touch the developer's store;
- every `XDG_*` directory redirected inside that work dir;
- a scrubbed environment — `env -i` starts from nothing and only an
  explicit allowlist is passed back in, so `APS_*`, `BUS_TOKEN`, and
  any ambient credential cannot reach the recorded process;
- the event bus consequently disabled, so no network call is made.

The work root is a fixed path rather than `$TMPDIR`-derived: `aps
config paths` echoes the absolute paths it probed, and a machine-
specific root would churn the committed captures on every re-record.

Isolation is a claim, so it is checked rather than trusted.
`scripts/12fcc-scan-cassettes.sh` greps the bytes that are actually
about to be committed for credential-shaped values, `secrets.env`
contents, and the recorder's home directory and username. It runs in
CI on every change (the `cassette-secrets` job) and fails the build on
any finding. Run it after every re-record:

```sh
make 12fcc-scan
```

## Re-running

Record fresh cassettes (builds the binary, isolates a work dir per
scenario, runs every scenario step as a real subprocess):

```sh
KIT_BIN=/path/to/kit make 12fcc-record
KIT_BIN=/path/to/kit make 12fcc-scan     # never skip this before committing
```

Grade them against a locally served scenario library (requires a `kit`
binary shipping the `conformance` command group; build one from
`hop.top/kit` `cmd/kit`):

```sh
KIT_BIN=/path/to/kit make 12fcc-grade
```

The grade target boots `kit conformance svc serve` on a loopback port
with `--scenarios-root e2e/conformance`, mints a `grade:aps` token
into a throwaway claims DB, uploads every cassette at tier 3, rewrites
`verdicts/`, and prints per-scenario verdicts plus a per-factor
rollup. Grading is measurement, not a gate: the target fails only when
a cassette cannot be graded at all. Set `STRICT=1` to also fail on
`fail` verdicts.

## Badge

`.12fc.json` at the repo root — the shields.io endpoint behind the
README badge — is committed and is the source of truth for all 12
factors. CI never rewrites it: the `12fcc` workflow can only run the
story and leak leaves, and the badge counts measured factors only, so
a CI refresh would degrade a truthful `12/12 pass` badge. The workflow
keeps its badge commit disabled until a released `kit` can genuinely
grade cassettes in CI; the enable checklist lives inline in
`.github/workflows/12fcc.yaml`.

Refresh the badge locally, from measurements only:

```sh
KIT_BIN=/path/to/kit make 12fcc-record
KIT_BIN=/path/to/kit make 12fcc-grade
KIT_BIN=/path/to/kit make 12fcc-badge
```

An unmeasured factor stays `skip` and is never counted as a pass.

## Current verdicts

As of the captures in `cassettes/` (see `verdicts/` for full facets):

| Scenario | Verdict | Factors |
|---|---|---|
| `status-json` | pass | F3 F8 F11 |
| `contract-surfaces` | pass | F1 F5 F8 F11 |
| `error-envelope-correction` | pass | F4 F11 |
| `profile-lifecycle` | pass | F3 F7 F11 |
| `capability-idempotent` | pass | F7 F11 |
| `delete-confirm-gate` | pass | F6 F10 F11 |

Factor rollup: every graded factor (F1, F3–F8, F10, F11) passes. The
behaviors carrying the verdicts:

- **F1/F5** — `aps toolspec` serves the command tree, per-command
  flags, safety classification, and curated error patterns as one
  structured document pinned to a schema version; `aps capability
  list` enumerates the capability roster.
- **F3** — `status`, `profile list`, `profile show`, `capability
  list`, and `config paths` each render one parseable document on
  stdout under `--format json`, with json and yaml agreeing on key
  names.
- **F4** — error envelopes carry `code` and `suggested_fix`: a
  mistyped subcommand and a missing profile both echo the failing
  input and name the recovery command.
- **F6/F10** — `profile delete` refuses without a typed confirmation
  token, and the refusal is a true no-op: the profile is still present
  afterwards. The flow never requires a TTY.
- **F7** — a reissued `profile create` is refused as a conflict rather
  than duplicating, and a reissued `profile capability add` converges,
  leaving the capability recorded exactly once.
- **F8** — `status` exposes the active profile, workspace, build
  version, and bus wiring; `config paths` exposes the resolved config
  precedence chain with per-entry existence.
- **F11** — the process honors the kit exit-class table: USAGE=2 for a
  mistyped subcommand and an undeclared `--format`, NOT_FOUND=3 for a
  missing profile, CONFLICT=4 for a duplicate create, UNAUTHORIZED=5
  for an unconfirmed destructive command.

If a regression re-introduces a defect, re-record and re-grade; the
scenario library encodes the spec-correct expectation, so the verdict
flips without any rubric change.
