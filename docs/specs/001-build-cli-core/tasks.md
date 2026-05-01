# Tasks: CLI and Core Engine Implementation

**Feature Branch**: `001-build-cli-core`
**Spec**: [specs/001-build-cli-core/spec.md](../spec.md)

## Phase 1: Setup & Infrastructure

**Goal**: Initialize the project structure and dependencies.

- [x] T001 Initialize Go module `oss-aps-cli` in root
- [x] T002 Install dependencies: `spf13/cobra`, `charmbracelet/bubbletea`, `joho/godotenv`, `gopkg.in/yaml.v3`
- [x] T003 Create directory structure: `cmd/aps`, `internal/core`, `internal/cli`, `internal/tui`
- [x] T004 Implement `cmd/aps/main.go` entry point to execute root command

## Phase 2: Foundation (Shared Core)

**Goal**: Implement core business entities and logic required by all user stories.

- [x] T005 [P] Define `Profile` struct and YAML tags in `internal/core/profile.go`
- [x] T006 [P] Implement `LoadProfile(id)` and `SaveProfile(p)` in `internal/core/profile.go`
- [x] T007 [P] Implement `LoadSecrets(path)` in `internal/core/secrets.go` using dotenv parser
- [x] T008 [P] Implement `InjectEnvironment(cmd, profile)` in `internal/core/execution.go` to set `AGENT_*` vars and secrets

## Phase 3: Profile Management (User Story 1)

**Goal**: Enable creation and listing of agent profiles.
**Independent Test**: `aps profile list` returns empty, `aps profile create test` creates files, `aps profile list` shows `test`.

- [x] T009 [US1] Implement `CreateProfile(id, config)` logic in `internal/core/profile.go` (including default files generation)
- [x] T010 [US1] Implement `ListProfiles()` logic in `internal/core/profile.go` (scanning `~/.agents/profiles`)
- [x] T011 [US1] Implement `internal/cli/profile.go` with `profile` subcommands structure
- [x] T012 [P] [US1] Implement `aps profile list` command handler
- [x] T013 [P] [US1] Implement `aps profile new` command handler with flags
- [x] T014 [P] [US1] Implement `aps profile show` command handler

## Phase 4: Command Execution (User Story 2)

**Goal**: Run arbitrary commands in profile context.
**Independent Test**: `aps run test -- env` shows injected variables.

- [x] T015 [US2] Implement `RunCommand(profileID, command, args)` in `internal/core/execution.go`
- [x] T016 [US2] Implement `internal/cli/run.go` command handler

## Phase 5: Action Execution (User Story 3)

**Goal**: Discover and run defined actions.
**Independent Test**: `aps action list test` shows scripts, `aps action run` executes them.

- [x] T017 [US3] Define `Action` struct and `LoadActions(profileID)` in `internal/core/action.go` (scan files + optional manifest)
- [x] T018 [US3] Implement `RunAction(profileID, actionID, payload)` in `internal/core/execution.go`
- [x] T019 [US3] Implement `internal/cli/action.go` with subcommands structure
- [x] T020 [P] [US3] Implement `aps action list` command handler
- [x] T021 [P] [US3] Implement `aps action show` command handler
- [x] T022 [P] [US3] Implement `aps action run` command handler (handling stdin payload)

## Phase 6: Webhook Server (User Story 4)

**Goal**: Trigger actions via HTTP events.
**Independent Test**: `curl` POST to localhost triggers action.

- [x] T023 [US4] Define `WebhookEvent` and `WebhookServer` in `internal/core/webhook.go`
- [x] T024 [P] [US4] Implement HMAC validation logic in `internal/core/webhook.go`
- [x] T025 [P] [US4] Implement `ServeWebhooks(config)` handler logic in `internal/core/webhook.go` (mapping events to actions)
- [x] T026 [US4] Implement `internal/cli/webhook.go` command handler

## Phase 7: Documentation (User Story 5)

**Goal**: Generate local docs.
**Independent Test**: `aps docs` creates markdown files.

- [x] T027 [US5] Embed documentation assets in `internal/core/docs.go` using `embed`
- [x] T028 [US5] Implement `GenerateDocs(dest)` in `internal/core/docs.go`
- [x] T029 [US5] Implement `internal/cli/docs.go` command handler

## Phase 8: TUI Implementation (FR-009)

**Goal**: Full interactive UI for the system.
**Independent Test**: Running `aps` launches TUI.

- [x] T030 [P] Define TUI `Model` and `Init` in `internal/tui/model.go`
- [x] T031 [P] Implement Profile Selection view in `internal/tui/update.go` & `view.go`
- [x] T032 [P] Implement Action List view in `internal/tui/update.go` & `view.go`
- [x] T033 [P] Implement Execution Output view in `internal/tui/update.go` & `view.go`
- [x] T034 Integrate TUI launch into `internal/cli/root.go` (default action)

## Phase 9: Polish

- [x] T035 Add build instructions to README.md
- [x] T036 Verify cross-platform compilation (darwin, linux, windows)

## Dependencies

- **Phase 1 & 2** are blocking for all other phases.
- **Phase 3 (Profiles)** is blocking for Phase 4, 5, 6, 8.
- **Phase 4 (Run)** is blocking for Phase 5 (Actions) & 6 (Webhooks) as they reuse execution logic.
- **Phase 5 (Actions)** is blocking for Phase 6 (Webhooks).
- **Phase 8 (TUI)** depends on Phase 3 & 5 logic.

## Parallel Execution Examples

- **US1**: T012 (List), T013 (New), T014 (Show) can be implemented in parallel after T009-T011.
- **US3**: T020 (List), T021 (Show) can be implemented in parallel after T017.
- **TUI**: View components (T031, T032, T033) can be built in parallel.

## Implementation Strategy

- **MVP**: Complete Phases 1, 2, 3, 4. This gives a functional CLI tool for managing profiles and running commands.
- **Increment 1**: Add Phase 5 (Actions) and Phase 6 (Webhooks).
- **Increment 2**: Add Phase 8 (TUI) and Phase 7 (Docs).
