# Tasks: Automated E2E Test Suite

**Feature Branch**: `002-e2e-tests`
**Spec**: [specs/002-e2e-tests/spec.md](../spec.md)

## Phase 1: Setup & Infrastructure

**Goal**: Prepare the test harness and dependencies.

- [x] T001 Install `github.com/stretchr/testify` dependency
- [x] T002 Create `tests/e2e` directory structure
- [x] T003 Implement `TestMain` in `tests/e2e/main_test.go` to compile `aps` binary
- [x] T004 Implement helper `runAPS(t, home, args...)` in `tests/e2e/helpers_test.go` to execute binary with isolated environment

## Phase 2: Profile Tests (User Story 1)

**Goal**: Verify profile lifecycle.

- [x] T005 [US1] Implement `TestProfileLifecycle` in `tests/e2e/profile_test.go` (Create, List, Show)
- [x] T006 [P] [US1] Implement `TestProfileOverwrite` in `tests/e2e/profile_test.go` (Force flag logic)

## Phase 3: Execution Tests (User Story 2)

**Goal**: Verify command execution and injection.

- [x] T007 [US2] Implement `TestExecutionInjection` in `tests/e2e/run_test.go` (Verify AGENT_* vars)
- [x] T008 [P] [US2] Implement `TestSecretInjection` in `tests/e2e/run_test.go` (Verify secrets.env)

## Phase 4: Action Tests (User Story 3)

**Goal**: Verify action discovery and running.

- [x] T009 [US3] Implement `TestActionDiscovery` in `tests/e2e/action_test.go` (List actions)
- [x] T010 [US3] Implement `TestActionRun` in `tests/e2e/action_test.go` (Run script, check output)
- [x] T011 [P] [US3] Implement `TestActionPayload` in `tests/e2e/action_test.go` (Stdin handling)

## Phase 5: Webhook Tests (User Story 4)

**Goal**: Verify webhook server.

- [x] T012 [US4] Implement `TestWebhookServer` in `tests/e2e/webhook_test.go` (Start server, send POST, verify 200/401/400)

## Phase 6: Polish

- [x] T013 Verify entire suite passes locally (`go test ./tests/e2e -v`)
- [x] T014 Add test running instruction to `README.md`

## Dependencies

- Phase 1 blocks everything.
- Phases 2-5 are parallelizable but logically ordered by feature dependency (cannot run action without profile).

## Implementation Strategy

- Implement harness first.
- Build tests incrementally.
- Ensure cleanup of binary in `TestMain` shutdown.
