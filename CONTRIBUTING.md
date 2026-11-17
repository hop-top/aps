# Contributing to aps

Thanks for your interest in contributing!

## Getting Started

1. Fork the repository
2. Clone your fork locally
3. Create a feature branch: `git checkout -b feat/my-change`
4. Make your changes
5. Run tests: `make test`
6. Commit using
   [Conventional Commits](https://conventionalcommits.org)
7. Push and open a Pull Request

## Development Setup

```sh
mise install
make build
```

This installs the pinned toolchain (Go, golangci-lint,
gofumpt) and builds the `aps` binary into `bin/`.

## Code Style

- Follow existing conventions in the codebase
- Run linters before submitting: `make lint`
- Format with the pinned `gofumpt` (`mise exec -- gofumpt -w .`)
- Keep changes focused; one concern per PR
- Default to no comments — prefer self-documenting code

## Local Replace Directives in go.mod

`replace` directives in the committed `go.mod` are not
allowed. They break Docker image builds (the target path is
not present in the build context), break Dependabot rebases
(the resolver cannot graph a local path), and break every
contributor whose checkout layout does not match the author's.

An unpublished upstream plus a local `replace` silently lets
the repo drift hundreds of commits ahead of anything CI can
reach; the build stays green on the author's machine and is
broken everywhere else.

For local development against an unreleased dependency:

- Keep `replace` lines in an untracked `go.mod.local` overlay,
  or
- Use `go work` with a `go.work` file (also untracked)

Neither belongs in git. `go.mod.local`, `go.work`, and
`go.work.sum` are listed in `.gitignore`.

**Exception**: a short-lived migration window when `aps` and
a co-versioned dependency (typically `hop.top/kit`) need to
land together. The acceptable flow is publish-then-bump in a
single day — never park a `replace` on `main` waiting for an
upstream release.

**Responsibility**: the PR author removes any `replace`
directive against `hop.top/*` before requesting review. CI
fails the build if one is present at merge time.

## Commit Messages

Use [Conventional Commits](https://conventionalcommits.org):

```
feat(scope): add new feature
fix(scope): correct a bug
docs: update readme
test: add missing tests
```

Scope is usually the top-level subsystem (`cli`, `core`,
`adapters`, `runtime`, etc.). Commit often; squash on merge
is the norm.

## Pull Requests

- Reference related issues in the PR description
- Keep PRs small and reviewable
- Ensure CI passes before requesting review
- Update documentation if behavior changes

## Issues

- Search existing issues before opening a new one
- Use issue templates when available
- Provide reproduction steps for bugs

## Code of Conduct

Be respectful and constructive. We are all here to build
something great together.

**Unreleased uma**: `hop.top/uma` (unreleased, private repository) is not
in `go.mod` at all. The uma bridge in `internal/adapters/umabridge` builds
only with an untracked `go.work` that uses a uma checkout:

```text
go 1.27.4

use (
	.
	<uma checkout>
	<uma checkout>/adapter/mail
	<uma checkout>/adapter/messaging/meta
	<uma checkout>/adapter/messaging/twilio
	<uma checkout>/testing/xrrtest
)
```

Do not add `require` lines for uma modules while it is supplied this way: a
requirement on the nested module `hop.top/uma/adapter/mail` makes workspace
mode look it up remotely, and the lookup fails. Without the `go.work`, CI
and consumers cannot build the bridge. Once uma publishes a version, a
plain `require` replaces the `go.work`.
