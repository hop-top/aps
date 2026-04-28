# Feature Specification: Automated E2E Test Suite

**Feature Branch**: `002-e2e-tests`
**Created**: 2026-01-15
**Status**: Draft
**Input**: User description: "Implement comprehensive E2E test suite"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Profile & Lifecycle Tests (Priority: P1)

As a maintainer, I want automated verification of profile management commands to ensure the basic data model works correctly across changes.

**Why this priority**: Profile management is the prerequisite for all other features.

**Independent Test**: Run `go test ./tests/e2e -v -run TestProfileLifecycle`

**Acceptance Scenarios**:

1. **Given** a clean state, **When** `aps profile list` is run, **Then** output is empty.
2. **Given** a new profile `e2e-agent`, **When** created via `aps profile new`, **Then** it appears in `aps profile list`.
3. **Given** an existing profile, **When** `aps profile show` is run, **Then** valid YAML matches expectations.
4. **Given** an existing profile, **When** attempting to overwrite without `--force`, **Then** it fails.
5. **Given** an existing profile, **When** overwritten with `--force`, **Then** it succeeds.

---

### User Story 2 - Execution & Environment Tests (Priority: P1)

As a maintainer, I want automated verification that `aps run` correctly injects secrets and variables.

**Why this priority**: Core value proposition is isolated execution environments.

**Independent Test**: Run `go test ./tests/e2e -v -run TestExecution`

**Acceptance Scenarios**:

1. **Given** a profile with a known secret in `secrets.env`, **When** running `aps run ... -- env`, **Then** the secret is present in output.
2. **Given** a profile, **When** running `aps run ... -- env`, **Then** `AGENT_PROFILE_ID` and `AGENT_PROFILE_DIR` are set correctly.
3. **Given** a profile with `git.enabled=true`, **When** running `env`, **Then** `GIT_CONFIG_GLOBAL` is injected.

---

### User Story 3 - Action Discovery & Run Tests (Priority: P2)

As a maintainer, I want verification that actions (scripts) are discovered and executed properly with payloads.

**Why this priority**: Ensures the scripting capability works.

**Independent Test**: Run `go test ./tests/e2e -v -run TestActions`

**Acceptance Scenarios**:

1. **Given** a profile with a `.sh` script in `actions/`, **When** `aps action list` is run, **Then** the script is listed.
2. **Given** a simple echo script, **When** `aps action run` is called, **Then** output matches expected.
3. **Given** a script reading stdin, **When** `aps action run ... --payload-file` is used, **Then** script receives the content.

---

### User Story 4 - Webhook Integration Tests (Priority: P3)

As a maintainer, I want verification that the webhook server correctly routes events and validates signatures.

**Why this priority**: Complex integration point that needs regression testing.

**Independent Test**: Run `go test ./tests/e2e -v -run TestWebhooks`

**Acceptance Scenarios**:

1. **Given** a running webhook server, **When** a valid signed request is sent, **Then** it returns 200 and triggers the action.
2. **Given** a running webhook server, **When** an invalid signature is sent, **Then** it returns 401.
3. **Given** a running webhook server, **When** an unmapped event is sent, **Then** it returns 400.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Test suite MUST compile the `aps` binary from source before running tests to ensure it tests the current code.
- **FR-002**: Test suite MUST use a temporary directory for `~/.agents` to avoid modifying the user's actual profiles (using `HOME` override).
- **FR-003**: Tests MUST cover all CLI subcommands: `profile`, `run`, `action`, `webhook`.
- **FR-004**: Tests MUST use Go's standard `testing` package (e.g., `tests/e2e/main_test.go`).
- **FR-005**: Tests MUST be able to run in parallel where possible (t.Parallel()).

### Key Entities

- **TestHarness**: Helper struct to manage temp dirs, compilation, and command execution.
- **E2E Suite**: The collection of test functions in `tests/e2e/`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `go test ./tests/e2e` passes with 100% success rate on the `main` branch.
- **SC-002**: Test suite execution time is under 10 seconds (excluding compilation).
- **SC-003**: Coverage includes at least one test case for every User Story defined in `001-build-cli-core`.

### Edge Cases

- **EC-001**: Handling of OS-specific path separators in assertions.
- **EC-002**: Cleanup of temp directories on test failure.