---

description: "Task list for A2A Protocol adoption in APS"
---

# Tasks: A2A Protocol Adoption

**Input**: Design documents from `/specs/005-a2a-protocol/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, quickstart.md

**Tests**: This implementation follows TDD approach - test tasks are included for critical components.

**Organization**: Tasks are grouped by functional phase to enable incremental implementation and validation of A2A protocol adoption.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go modules**: `internal/a2a/` for A2A protocol components
- **CLI commands**: `internal/cli/a2a/` for CLI integration
- **Tests**: `tests/unit/a2a/` for unit tests, `tests/e2e/` for E2E tests

---

## Phase 1: Setup (Protocol Adoption) ✅ COMPLETE

**Purpose**: Archive custom protocol and prepare for official A2A adoption

- [X] T001 Move custom A2A protocol spec to legacy/custom-spec.md in specs/005-a2a-protocol/
- [X] T002 [P] Create internal/a2a/ package directory structure
- [X] T003 [P] Add github.com/a2aproject/a2a-go@v0.3.4 to go.mod
- [X] T004 Run `go mod tidy` to resolve dependencies
- [X] T005 [P] Create .gitignore rules for A2A-specific files (internal/a2a/*.test, a2a-storage/)
- [X] T006 [P] Create internal/a2a/README.md with package overview

---

## Phase 2: Foundational (Core Infrastructure) ✅ COMPLETE

**Purpose**: Core A2A protocol infrastructure that MUST be complete before any feature work

**⚠️ CRITICAL**: No feature work can begin until this phase is complete

### Tests for Foundational Components

- [X] T007 [P] Test A2A package initialization in internal/a2a/config_test.go
- [X] T008 [P] Test Agent Card generation framework in internal/a2a/agentcard_test.go

### Foundational Implementation

- [X] T009 Define A2A configuration structures in internal/a2a/config.go
- [X] T010 [P] Implement Agent Card generator skeleton in internal/a2a/agentcard.go
- [X] T011 [P] Implement A2A task storage skeleton in internal/a2a/storage.go
- [X] T012 Define A2A error types and error handling in internal/a2a/errors.go
- [X] T013 Implement A2A transport interface in internal/a2a/transport/interface.go

**Checkpoint**: Foundation ready - A2A protocol integration can now begin

---

## Phase 3: User Story 1 - SDK Integration & Agent Cards (Priority: P1) 🎯 MVP ✅ COMPLETE

**Goal**: Integrate a2a-go SDK and implement Agent Card generation for APS profiles

**Independent Test**: Create a profile with A2A enabled and verify it generates a valid Agent Card

### Tests for User Story 1

- [X] T014 [P] [US1] Test Agent Card generation from profile config in internal/a2a/agentcard_generation_test.go
- [X] T015 [P] [US1] Test Agent Card validation in internal/a2a/agentcard_validation_test.go
- [ ] T016 [P] [US1] Integration test for profile-to-Agent Card mapping in tests/e2e/a2a_profile_test.go

### Implementation for User Story 1

- [X] T017 [P] [US1] Implement profile → Agent Card mapping in internal/a2a/agentcard.go
- [X] T018 [P] [US1] Implement Agent Card capabilities from profile config in internal/a2a/agentcard.go
- [X] T019 [P] [US1] Implement Agent Card security schemes in internal/a2a/agentcard.go
- [X] T020 [US1] Implement Agent Card generation for APS profiles in internal/a2a/agentcard.go
- [X] T021 [US1] Add Agent Card serialization to JSON in internal/a2a/agentcard.go
- [X] T022 [US1] Add Agent Card validation logic in internal/a2a/agentcard.go
- [X] T023 [US1] Integrate Agent Card generation with core.Profile in internal/a2a/agentcard.go
- [X] T024 [US1] Add Agent Card caching in internal/a2a/cache.go

**Checkpoint**: APS profiles can generate valid Agent Cards (port 8081)

---

## Phase 4: User Story 2 - A2A Server (Priority: P2) IN PROGRESS

**Goal**: Implement A2A Server using a2asrv to expose APS profiles as A2A agents

**Independent Test**: Start A2A server for a profile and verify it responds to task requests

### Tests for User Story 2

- [X] T025 [P] [US2] Test A2A Server initialization in internal/a2a/server_test.go
- [ ] T026 [P] [US2] Test A2A Server request handling in tests/unit/a2a/server_handler_test.go
- [ ] T027 [P] [US2] Integration test for task submission to A2A server in tests/e2e/a2a_server_test.go

### Implementation for User Story 2

- [X] T028 [P] [US2] Implement APS profile executor in internal/a2a/executor.go
- [X] T029 [US2] Implement A2A Server using a2asrv in internal/a2a/server.go
- [ ] T030 [US2] Implement Task lifecycle management in internal/a2a/server.go
- [ ] T031 [US2] Add message handling in internal/a2a/server.go
- [ ] T032 [US2] Add task storage integration in internal/a2a/server.go
- [ ] T033 [US2] Implement task status tracking in internal/a2a/server.go
- [ ] T034 [US2] Add streaming support (SendMessageStream) in internal/a2a/server.go
- [ ] T035 [US2] Add push notification support (SubscribeToTask) in internal/a2a/server.go

**Checkpoint**: APS profiles can receive and process A2A tasks

---

## Phase 5: User Story 3 - A2A Client (Priority: P3)

**Goal**: Implement A2A Client using a2aclient to enable profile-to-profile communication

**Independent Test**: Create a task from one APS profile to another and verify completion

### Tests for User Story 3

- [ ] T036 [P] [US3] Test A2A Client initialization in tests/unit/a2a/client_test.go
- [ ] T037 [P] [US3] Test A2A Client SendMessage in tests/unit/a2a/client_message_test.go
- [ ] T038 [P] [US3] Integration test for profile-to-profile communication in tests/e2e/a2a_client_test.go

### Implementation for User Story 3

- [ ] T039 [P] [US3] Implement Agent Card resolver in internal/a2a/resolver.go
- [ ] T040 [US3] Implement A2A Client using a2aclient in internal/a2a/client.go
- [ ] T041 [US3] Implement SendMessage for task creation in internal/a2a/client.go
- [ ] T042 [US3] Implement GetTask for task retrieval in internal/a2a/client.go
- [ ] T043 [US3] Implement ListTasks for task querying in internal/a2a/client.go
- [ ] T044 [US3] Implement CancelTask for task cancellation in internal/a2a/client.go
- [ ] T045 [US3] Implement SubscribeToTask for push notifications in internal/a2a/client.go
- [ ] T046 [US3] Implement streaming support (SendMessageStream) in internal/a2a/client.go
- [ ] T047 [US3] Add transport selection logic in internal/a2a/client.go
- [ ] T048 [US3] Integrate A2A client with core.Profile in internal/core/profile.go

**Checkpoint**: APS profiles can create and manage A2A tasks

---

## Phase 6: User Story 4 - Transport Adapters (Priority: P4)

**Goal**: Implement transport adapters for APS isolation tiers (IPC, HTTP, gRPC)

**Independent Test**: Communicate across isolation tiers using appropriate transports

### Tests for User Story 4

- [ ] T049 [P] [US4] Test IPC transport in tests/unit/a2a/transport/ipc_test.go
- [ ] T050 [P] [US4] Test HTTP transport in tests/unit/a2a/transport/http_test.go
- [ ] T051 [P] [US4] Test gRPC transport in tests/unit/a2a/transport/grpc_test.go
- [ ] T052 [P] [US4] Integration test for cross-tier communication in tests/e2e/a2a_transport_test.go

### Implementation for User Story 4

- [ ] T053 [P] [US4] Implement IPC transport adapter in internal/a2a/transport/ipc.go
- [ ] T054 [US4] Implement IPC queue management in internal/a2a/transport/ipc.go
- [ ] T055 [P] [US4] Implement HTTP transport adapter in internal/a2a/transport/http.go
- [ ] T056 [US4] Implement JSON-RPC handler in internal/a2a/transport/http.go
- [ ] T057 [P] [US4] Implement gRPC transport adapter in internal/a2a/transport/grpc.go
- [ ] T058 [US4] Implement gRPC server/client in internal/a2a/transport/grpc.go
- [ ] T059 [US4] Add transport registration in internal/a2a/transport/registry.go
- [ ] T060 [US4] Implement automatic transport fallback in internal/a2a/transport/selector.go
- [ ] T061 [US4] Map isolation tiers to transports in internal/a2a/isolation.go
- [ ] T062 [US4] Add authentication per transport type in internal/a2a/transport/auth.go

**Checkpoint**: A2A communication works across all APS isolation tiers

---

## Phase 7: User Story 5 - CLI Integration (Priority: P5)

**Goal**: Update CLI commands to use A2A protocol instead of custom protocol

**Independent Test**: Execute CLI commands to create, list, and manage A2A tasks

### Tests for User Story 5

- [ ] T063 [P] [US5] Test CLI a2a commands in tests/unit/cli/a2a_test.go
- [ ] T064 [P] [US5] Integration test for CLI task management in tests/e2e/cli_a2a_test.go

### Implementation for User Story 5

- [ ] T065 [P] [US5] Create a2a command group in internal/cli/a2a/cmd.go
- [ ] T066 [P] [US5] Implement `aps a2a list-tasks` command in internal/cli/a2a/list_tasks.go
- [ ] T067 [P] [US5] Implement `aps a2a get-task` command in internal/cli/a2a/get_task.go
- [ ] T068 [P] [US5] Implement `aps a2a send-task` command in internal/cli/a2a/send_task.go
- [ ] T069 [P] [US5] Implement `aps a2a send-stream` command in internal/cli/a2a/send_stream.go
- [ ] T070 [P] [US5] Implement `aps a2a cancel-task` command in internal/cli/a2a/cancel_task.go
- [ ] T071 [P] [US5] Implement `aps a2a subscribe-task` command in internal/cli/a2a/subscribe_task.go
- [ ] T072 [P] [US5] Implement `aps a2a show-agent-card` command in internal/cli/a2a/show_card.go
- [ ] T073 [P] [US5] Implement `aps a2a fetch-agent-card` command in internal/cli/a2a/fetch_card.go
- [ ] T074 [P] [US5] Implement `aps a2a register` command in internal/cli/a2a/register.go
- [ ] T075 [P] [US5] Implement `aps a2a discover` command in internal/cli/a2a/discover.go
- [ ] T076 [US5] Register a2a command group in cmd/aps/root.go
- [ ] T077 [US5] Add A2A configuration flags to `aps profile new` command in internal/cli/profile/new.go
- [ ] T078 [US5] Add A2A output to `aps profile show` command in internal/cli/profile/show.go

**Checkpoint**: Users can manage A2A tasks via CLI

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple components

- [ ] T079 [P] Update documentation for A2A integration in README.md
- [ ] T080 [P] Update AGENTS.md with A2A-specific guidelines
- [ ] T081 [P] Create examples directory with A2A usage examples
- [ ] T082 [P] Add performance benchmarks for A2A operations in tests/bench/a2a_bench_test.go
- [ ] T083 [P] Add security audit validation in tests/e2e/a2a_security_test.go
- [ ] T084 [P] Implement legacy protocol read-only access in internal/a2a/legacy.go
- [ ] T085 [P] Add task archival functionality in internal/a2a/server.go
- [ ] T086 [P] Run all tests with `go test ./...`
- [ ] T087 [P] Run E2E test suite with `go test -v ./tests/e2e`
- [ ] T088 Verify quickstart.md scenarios work end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can proceed in priority order (P1 → P2 → P3 → P4 → P5)
  - Each story should be independently testable
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Depends on US1 (Agent Cards) for server configuration
- **User Story 3 (P3)**: Depends on US2 (Server) for client-to-server communication
- **User Story 4 (P4)**: Depends on US2 (Server) and US3 (Client) for transport integration
- **User Story 5 (P5)**: Depends on US1-4 for CLI command implementation

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD approach)
- Models/structures before services
- Services before integration
- Core implementation before CLI/UI integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tests marked [P] can run in parallel
- All Foundational implementation tasks marked [P] can run in parallel
- Tests for each user story marked [P] can run in parallel
- CLI commands in US5 marked [P] can run in parallel
- Different user stories can be worked on in parallel after dependencies are met

---

## Parallel Example: User Story 1 (Agent Cards)

```bash
# Launch all tests for User Story 1 together:
Task: "Test Agent Card generation from profile config in tests/unit/a2a/agentcard_generation_test.go"
Task: "Test Agent Card validation in tests/unit/a2a/agentcard_validation_test.go"
Task: "Integration test for profile-to-Agent Card mapping in tests/e2e/a2a_profile_test.go"

# Launch all implementation tasks for User Story 1 together:
Task: "Implement profile → Agent Card mapping in internal/a2a/agentcard.go"
Task: "Implement Agent Card capabilities from profile config in internal/a2a/agentcard.go"
Task: "Implement Agent Card security schemes in internal/a2a/agentcard.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (Agent Cards)
4. **STOP and VALIDATE**: Create profile, generate Agent Card, validate against A2A spec
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test Agent Card generation → Validate (MVP!)
3. Add User Story 2 → Test A2A Server → Validate
4. Add User Story 3 → Test A2A Client → Validate
5. Add User Story 4 → Test transport adapters → Validate
6. Add User Story 5 → Test CLI integration → Validate
7. Complete Polish → End-to-end validation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Agent Cards)
   - Developer B: User Story 2 (A2A Server)
   - Developer C: User Story 3 (A2A Client)
3. After US1-3 complete:
   - Developer A: User Story 4 (Transports)
   - Developer B: User Story 5 (CLI)
   - Developer C: Testing & Polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (TDD)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Follow official A2A Protocol specification: https://a2a-protocol.org/latest/specification/
- Reference a2a-go SDK documentation: https://github.com/a2aproject/a2a-go
