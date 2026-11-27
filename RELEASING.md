# Releasing aps

`aps` releases through [release-please](https://github.com/googleapis/release-please)
with hop-top's target versioning: the branch decides the version.
The flow is automated; humans review the release PR and merge it.

## Branches

| Branch | Role | What lands there |
|--------|------|------------------|
| `next` | prerelease, default | All work. Every PR targets `next`. |
| `main` | stable | Nothing directly. It receives `next` once a line is at rc. |

Only two kinds of commit are made on `main` itself: the stable release
commit and hotfixes. Both are merged back into `next` right away, so
`next` always contains everything on `main`.

## How releases work

1. Conventional commits land on `next` through pull requests
2. The `release-please` workflow runs on every push to `next` and
   maintains a single release PR titled
   `chore(release):aps <version>`
3. Merging the release PR creates a Git tag `aps/v<version>`
   and a GitHub Release; `CHANGELOG.md` is updated in the
   same merge
4. `proxy.golang.org` picks up the new tag automatically —
   there is no separate publish step for the Go module

## Version scheme

A prerelease line previews one target stable version:

```
0.6.0-alpha.0 -> .1 -> ... -> 0.6.0-beta.0 -> ... -> 0.6.0-rc.0 -> ... -> 0.6.0
```

The target is the last stable release bumped by the commits since:
a breaking change or a `feat` opens the next minor (before 1.0),
anything else the next patch. While the line's target covers the
commits, the counter goes up; when they need more, a new line starts
at `alpha.0`.

Configured in `.github/release-please-config.json`:

- `release-type: go`
- `versioning: target`, with the `target-context` plugin
- `prerelease-type: alpha.0` (every line starts on alpha)
- `bump-minor-pre-major: true`

Nobody picks a version: `Release-As` footers are ignored.

## Promote a line (alpha -> beta -> rc)

Channels move one step at a time and never on their own. A promotion
is recorded in the config with `promote.js` from a
[hop-top/.github](https://github.com/hop-top/.github) checkout
(Node 20+, after `npm ci` in its `scripts/release-please-target`),
run from this repository's root:

```sh
P=path/to/hop-top-dotgithub/scripts/release-please-target/bin/promote.js

node "$P" channel beta --package aps --dry-run   # print, write nothing
node "$P" channel beta --package aps
```

Land the config change on `next`; the next run cuts `<line>-beta.0`.
Repeat with `rc` once the beta is released.

## Cut a stable release

1. Promote the line to rc on `next` and merge its release PR
2. Fast-forward `main` to that commit:
   `git push origin <rc-commit>:main`
   (a pull request would rewrite the commits; the repository allows
   rebase merges only)
3. The run on `main` opens a release PR for the stable version the rc
   previewed; merging it tags `aps/v<x.y.z>`
4. Merge `main` back into `next` so `next` carries the stable
   changelog and manifest

A `feat` or breaking change that reaches `main` without a preview on
`next` blocks the stable release: land it on `next` instead.

## Conventional Commits → CHANGELOG sections

| Commit type | Section         | Visible |
|-------------|-----------------|---------|
| `feat`      | Features        | yes     |
| `fix`       | Bug Fixes       | yes     |
| `perf`      | Performance     | yes     |
| others      | (hidden)        | no      |

See `changelog-sections` in
`.github/release-please-config.json` for the full mapping.

## Hotfix

For an out-of-band patch against an already-released stable version:

1. Branch from `main`
2. Commit the fix as a `fix:` Conventional Commit and merge it to
   `main` — only fixes; the run cuts the next patch
3. Merge `main` back into `next`

A fix for a version that is still a prerelease goes to `next` like any
other change.

## Do not edit CHANGELOG.md by hand

release-please owns `CHANGELOG.md`. Any manual edit will be
overwritten on the next release.
