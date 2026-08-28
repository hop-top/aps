# Developing aps

This guide is for changing the aps codebase. If you only want to *use*
aps, start at [SKILL.md](SKILL.md). Contribution etiquette (commit
style, PR rules, `replace` directive policy) lives in
[CONTRIBUTING.md](CONTRIBUTING.md).

## Setup

Toolchain is pinned with [mise](https://mise.jdx.dev) (see
[mise.toml](mise.toml)): Go, goreleaser, act, lychee, and
golangci-lint pinned to a specific minor version so local and CI agree
on enabled linters.

```bash
mise install        # install the pinned toolchain
make build          # build bin/aps
make build-vault    # opt-in openbao secrets backend (build tag: openbao)
make run ARGS="profile list"
make help           # every target, one line each
```

## Verification gate

Run these before pushing; CI runs the same tools at the same pinned
versions.

```bash
make test        # Go tests + GitHub workflows locally via act
make test-go     # just the Go tests
make lint        # docs-check + golangci-lint run ./...
make lint-docs   # scripts/check-links.sh: lychee over docs/**/*.md
                 # + story-referenced .go paths must exist
```

`make lint` runs `docs-check` first — stale generated doc regions fail
the lint gate, not just a docs job.

## Generated docs — never hand-edit

Tables between cog begin/end comment markers — see the marker pairs in
README.md's Adapters section for the exact syntax — are **generated**;
README.md and several `docs/` pages carry them. `make docs-gen` runs
[cog](https://cog.readthedocs.io) (via `uvx`, pinned in the Makefile)
over every marked file; the cog blocks shell out to the renderers under
`internal/tools/` (`adaptermd`, `configmd`, `exportmd`, `messengermd`,
`servicemd`).

The source of truth is metadata in code:

- `internal/core/secrets_meta.go`, `internal/core/service_meta.go` —
  secrets/service tables
- `internal/core/adapter/types.go` — adapter kinds and strategies

To change a generated table: edit the metadata (or the renderer), then
`make docs-gen`, and commit both. Editing the rendered region by hand
is always wrong — `make docs-check` (part of `make lint`) fails on any
drift between markers and their source.

## Testing conventions

- **Layout**: `tests/unit`, `tests/e2e`, `tests/fixtures` — the
  classic split. User-journey E2E also runs inside Docker
  (`make docker-test-e2e-user`; see
  [docs/dev/testing/docker-testing-strategy.md](docs/dev/testing/docker-testing-strategy.md)).
- **Pin tests**: `tests/docsync` pins judgment-laden doc prose to the
  code it describes — `exitcodes_test.go` pins the exit-code table in
  [docs/cli/reference.md](docs/cli/reference.md) to
  `internal/cli/exit`. `internal/cli` carries inventory-style pins of
  the CLI surface itself (`note_inventory_test.go`,
  `flag_alignment_test.go`, `strict_gates_test.go`,
  `messenger_surface_test.go`). When you change documented behavior,
  update the doc and its pin in the same commit.
- **Story-linked tests**: files in `docs/stories/` reference the `.go`
  tests that cover them; `make test-stories` runs exactly those, and
  `make lint-docs` fails if a referenced path no longer exists.
- **Conformance**: `e2e/conformance` holds 12-factor AI-CLI cassettes
  and the badge data behind `.12fc.json`; regenerate with the
  `make 12fcc-record` / `12fcc-scan` / `12fcc-grade` / `12fcc-badge`
  chain (see [e2e/conformance/README.md](e2e/conformance/README.md)
  for the required kit binary).

## Release flow

Releases are automated; the in-repo pieces are:

- [RELEASING.md](RELEASING.md) — the authoritative description.
  Conventional commits land on `main`; the `release-please` workflow
  (`.github/workflows/release-please.yml`, config in
  `.github/release-please-config.json`) maintains a single release PR.
  Merging it creates the `aps/v<version>` tag, the GitHub Release, and
  the CHANGELOG.md update. release-please owns CHANGELOG.md — never
  edit it by hand.
- `.github/workflows/goreleaser.yml` — on `aps/v*` tags, calls the
  reusable `goreleaser-on-tag` workflow, which builds
  [.goreleaser.yaml](.goreleaser.yaml): linux/darwin/windows,
  amd64/arm64, a stock archive plus an `aps-vault` variant, checksums
  and SBOMs, then uploads the artifacts to the release and updates the
  Homebrew tap and Scoop bucket.
- Versioning is on the prerelease channel until 1.0
  (`0.x.0-alpha.N → beta.N → 0.x.0`); a `Release-As:` commit footer
  forces a specific version.
- Locally, `make release-snapshot` builds the full artifact set into
  `dist/` without tagging; `make release` is the tag-driven equivalent
  of what CI runs.
