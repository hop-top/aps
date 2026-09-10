# Running tests

How to run the aps test suites locally. Conventions (pin tests, story-linked tests,
generated docs) are in [DEVELOPING.md](../../../DEVELOPING.md).

## Layout

- `tests/unit`: centralized unit tests, grouped by package (`core`, `adapter`, `a2a`,
  `acp`, `skills`, and others). OS-specific files such as `isolation_darwin_test.go` run
  only on their OS.
- `tests/e2e`: end-to-end CLI tests against a built binary.
- `tests/fixtures`: fixtures and the Docker user-journey scripts.
- `tests/docsync`: pins documented tables (for example exit codes) to the code they
  describe.
- `internal/...`: package-level tests next to the code.

## Make targets

```sh
make test              # Go tests plus the GitHub workflows locally via act
make test-go           # go test -v ./...
make test-unit         # go test -v ./tests/unit/...
make test-e2e          # go test -v ./tests/e2e/...
make test-stories      # only the tests referenced from docs/stories/
make test-workflows    # GitHub Actions locally with act
```

Direct forms:

```sh
go test -v ./tests/unit/...
go test -v ./tests/e2e
go test -v ./tests/unit/core/isolation/...        # platform isolation tests for this OS
go test -timeout 5m -tags openbao ./internal/core/...   # the opt-in OpenBao backend
```

CI (`.github/workflows/ci.yml`) runs the unit tests with coverage, the internal packages
except `core/isolation`, the `openbao`-tagged tests, and the end-to-end suite (skipped on
Windows) on Ubuntu, macOS, and Windows.

## Docker testing

A Docker-based Linux environment simulates a user's machine, so installation, setup, and
workflows can be tested without touching the local development environment.

```sh
make docker-build-test       # build the test image
make docker-test-e2e-user    # build the binary, the image, install it, run all user-journey tests
make docker-test-up          # start the test container in the background
make docker-test-install     # install the built binary into the container
make docker-test-shell       # interactive shell in the container for manual testing
make docker-test-down        # stop and remove the containers
make docker-test-cleanup     # remove containers, volumes, and the image
make docker-quick-start      # guided start via tests/fixtures/scripts/quick-start.sh
```

Guides: [Docker testing user guide](../../agent/docker-testing.md),
[Docker testing strategy](docker-testing-strategy.md),
[container test strategy](container-test-strategy.md),
[Unix test strategy](unix-test-strategy.md).

## Conformance

`e2e/conformance` holds the 12-factor AI-CLI cassettes and verdicts behind the badge data
in `.12fc.json`; regenerate them with the `make 12fcc-record`, `12fcc-scan`,
`12fcc-grade`, and `12fcc-badge` chain. Requirements and the refresh recipe:
[e2e/conformance/README.md](../../../e2e/conformance/README.md).
