# Releasing aps

`aps` releases through [release-please](https://github.com/googleapis/release-please).
The flow is automated end-to-end; humans review the release
PR and merge it.

## How releases work

1. Conventional commits land on `main` (squash merge is the
   norm)
2. The `release-please` workflow runs on every push and
   maintains a single release PR titled
   `chore(release):aps <version>`
3. Merging the release PR creates a Git tag `aps/v<version>`
   and a GitHub Release; `CHANGELOG.md` is updated in the
   same merge
4. `proxy.golang.org` picks up the new tag automatically —
   there is no separate publish step for the Go module

## Version scheme

`aps` is on the prerelease channel until 1.0:

```
0.5.0-alpha.0 -> .1 -> .2 -> ... -> 0.5.0-beta.0 -> ... -> 0.5.0
```

Configured in `.github/release-please-config.json`:

- `release-type: go`
- `prerelease: true`
- `prerelease-type: alpha.0` (the trailing `.0` is required)
- `versioning: prerelease`
- `bump-minor-pre-major: true`

`Release-As: <version>` footers in commit messages force a
specific version on the next release PR (useful to jump
between alpha → beta → rc → release).

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

For an out-of-band patch against an already-tagged release:

1. Branch from the tag: `git checkout -b hotfix/v<x.y.z> aps/v<x.y.z>`
2. Cherry-pick the fix; ensure it is a `fix:` Conventional Commit
3. Merge to `main` via PR — release-please picks it up on the next run

## Do not edit CHANGELOG.md by hand

release-please owns `CHANGELOG.md`. Any manual edit will be
overwritten on the next release.
