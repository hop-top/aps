# Tasks: Configurable Profile Env Var Prefix

**Input**: Design documents from `/specs/004-profile-env-prefix/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/config.schema.json

**Tests**: TDD approach requested. Tests MUST be written and fail before implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 [P] Verify `gopkg.in/yaml.v3` dependency in `go.mod`
- [X] T002 Create `internal/core/config.go` for global configuration structure

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure for configuration loading

- [X] T003 [P] Create unit tests for configuration loading in `tests/unit/core/config_test.go` (should fail)
- [X] T004 Implement `LoadConfig()` in `internal/core/config.go` to handle defaults and XDG discovery
- [X] T005 [P] Implement `GetConfigDir()` helper in `internal/core/config.go` using `os.UserConfigDir()`
- [X] T006 Verify all tests pass in `tests/unit/core/config_test.go`

**Checkpoint**: Configuration loading foundation ready.

---

## Phase 3: User Story 1 - Standardized Default Prefix (Priority: P1) 🎯 MVP

**Goal**: Change default prefix from `AGENT_PROFILE` to `APS`

**Independent Test**: Run an action without a config file and verify `APS_PROFILE_ID` is set and `AGENT_PROFILE_ID` is NOT set.

### Tests for User Story 1

- [X] T007 [US1] Create unit tests in `tests/unit/core/execution_test.go` to verify default `APS_` prefix injection (should fail)
- [X] T008 [US1] Update E2E tests in `tests/e2e/run_test.go` to expect `APS_` prefix instead of `AGENT_PROFILE_` (should fail)

### Implementation for User Story 1

- [X] T009 [US1] Modify `InjectEnvironment` in `internal/core/execution.go` to use `APS_` as the default prefix
- [X] T010 [US1] Update existing code to remove `AGENT_PROFILE_` hardcoding in `internal/core/execution.go`
- [X] T011 [US1] Verify all tests pass for US1

**Checkpoint**: MVP Ready - tool uses `APS_` prefix by default.

---

## Phase 4: User Story 2 - Custom Prefix via Configuration (Priority: P2)

**Goal**: Allow user to override prefix in `config.yaml`

**Independent Test**: Create `config.yaml` with `prefix: CUSTOM` and verify `CUSTOM_PROFILE_ID` is set.

### Tests for User Story 2

- [X] T012 [P] [US2] Add test case to `internal/core/config_test.go` for custom prefix loading (should fail)
- [X] T013 [P] [US2] Add test case to `internal/core/execution_test.go` verifying injection uses configured prefix (should fail)
- [X] T014 [US2] Create integration test in `tests/e2e/config_test.go` mocking a custom config file (should fail)

### Implementation for User Story 2

- [X] T015 [US2] Update `InjectEnvironment` in `internal/core/execution.go` to load global config and use its prefix
- [X] T016 [US2] Ensure `InjectEnvironment` handles missing or invalid config by falling back to `APS_`
- [X] T017 [US2] Verify all tests pass for US2

**Checkpoint**: Custom prefixes are now functional via configuration.

---

## Phase 5: User Story 3 - XDG-Compliant Configuration Discovery (Priority: P3)

**Goal**: Ensure config file is found in standard XDG locations

**Independent Test**: Set `XDG_CONFIG_HOME` and verify config is picked up from the new location.

### Tests for User Story 3

- [X] T018 [P] [US3] Add unit tests in `internal/core/config_test.go` mocking different `XDG_CONFIG_HOME` values (should fail)
- [X] T019 [US3] Add E2E test in `tests/e2e/config_test.go` that overrides `HOME` or `XDG_CONFIG_HOME` and verifies config lookup

### Implementation for User Story 3

- [X] T020 [US3] Refine configuration discovery logic in `internal/core/config.go` to strictly follow XDG priority
- [X] T021 [US3] Verify all tests pass for US3

**Checkpoint**: Full feature complete with XDG compliance.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T022 [P] Update `spec.md` at root if needed to reflect environment variable changes
- [X] T023 [P] Update `README.md` to document the new configuration file and prefix behavior
- [X] T024 Perform final run of all E2E tests to ensure no regressions
- [X] T025 Run `go fmt ./...` and `go vet ./...`

---

## Dependencies & Execution Order

1. **Setup & Foundational (Phase 1-2)**: MUST complete first.
2. **User Story 1 (Phase 3)**: Implements the breaking change (default shift).
3. **User Story 2 (Phase 4)**: Adds configurability on top of US1.
4. **User Story 3 (Phase 5)**: Finalizes OS-specific discovery logic.

### Parallel Opportunities
- Unit tests for different components (config vs execution) can be written in parallel.
- Documentation tasks can be done in parallel with implementation.

---

## Implementation Strategy

### MVP First (User Story 1 Only)
We will first move the hardcoded value to `APS_` and ensure the existing core functionality works with the new default.

### Incremental Delivery
Each user story builds on the previous one, adding a new layer of flexibility (defaults -> config -> discovery).

---

## Notes
- TDD is strictly enforced: Tests first, then code.
- Prefix should be sanitized to ensure it's a valid environment variable component.
