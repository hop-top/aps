# Feature Specification: CLI and Core Engine Implementation

**Feature Branch**: `001-build-cli-core`  
**Created**: 2026-01-15  
**Status**: Draft  
**Input**: User description: "build the cli"

## Clarifications

### Session 2026-01-15

- Q: Should `aps` (no args) launch the full TUI or just a placeholder? → A: Full TUI: Implement the complete interactive TUI (Bubble Tea) in this branch.
- Q: Where should shared logic between CLI and TUI reside? → A: Shared Internal Package: Create `internal/core` (or similar) for shared logic.
- Q: Which directory structure should the Go project use? → A: Standard Go Layout: `cmd/aps` for main, `internal/` for private code, `pkg/` for library code (if any).
- Q: How should the webhook server respond when an event has no mapping? → A: 400 Bad Request: Respond with HTTP 400 and a JSON body describing the mapping failure.
- Q: What format should `secrets.env` use for injection? → A: Standard .env: KEY=VALUE pairs, injected directly into the execution environment.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Profile Management (Priority: P1)

As a user, I want to create and manage agent profiles so that I can maintain isolated identities and configurations.

**Why this priority**: Profiles are the fundamental unit of the system; without them, no other functionality works.

**Independent Test**: Can be tested by creating, listing, and inspecting profiles without running actions.

**Acceptance Scenarios**:

1. **Given** no existing profiles, **When** I run `aps profile list`, **Then** I see no output (or empty list).
2. **Given** no profile `agent-a`, **When** I run `aps profile new agent-a`, **Then** the profile directory structure is created with default files.
3. **Given** an existing profile `agent-a`, **When** I run `aps profile list`, **Then** `agent-a` is listed.
4. **Given** an existing profile `agent-a`, **When** I run `aps profile show agent-a`, **Then** I see the profile configuration and modules.
5. **Given** an existing profile `agent-a`, **When** I run `aps profile new agent-a` without force, **Then** it fails to overwrite.

---

### User Story 2 - Command Execution (Priority: P1)

As a user, I want to run arbitrary commands within a profile's context so that I can utilize the profile's secrets and environment.

**Why this priority**: This is the core utility of the system for scripting and manual operations.

**Independent Test**: Can be tested by running `env` or `echo` commands and verifying output.

**Acceptance Scenarios**:

1. **Given** a profile `agent-a` with a secret `FOO=BAR`, **When** I run `aps run agent-a -- env`, **Then** the output contains `FOO=BAR`.
2. **Given** a profile `agent-a`, **When** I run `aps run agent-a -- whoami`, **Then** the command executes successfully.
3. **Given** a non-existent profile, **When** I run `aps run fake -- cmd`, **Then** it returns an error.

---

### User Story 3 - Action Execution (Priority: P2)

As a user, I want to discover and execute defined actions so that I can perform repeatable tasks.

**Why this priority**: Actions allow for complex, pre-defined behaviors, building upon the basic command execution.

**Independent Test**: Can be tested with sample scripts in the actions directory.

**Acceptance Scenarios**:

1. **Given** a profile with actions, **When** I run `aps action list <profile>`, **Then** I see the available actions.
2. **Given** an action `hello.sh`, **When** I run `aps action run <profile> hello`, **Then** the script executes.
3. **Given** an action accepting payload, **When** I run `aps action run <profile> action --payload-file data.json`, **Then** the action receives the data on stdin.

---

### User Story 4 - Webhook Server (Priority: P3)

As a user, I want to trigger actions via webhooks so that I can integrate with external systems like GitHub.

**Why this priority**: Enables event-driven automation but requires the core execution engine first.

**Independent Test**: Can be tested using `curl` to hit the local server.

**Acceptance Scenarios**:

1. **Given** a running webhook server mapping `event.x` to `profile:action`, **When** I POST to `/webhook` with matching headers, **Then** the action is triggered.
2. **Given** a secured webhook server, **When** I request without a signature, **Then** I receive 401 Unauthorized.

---

### User Story 5 - Documentation Generation (Priority: P3)

As a user, I want to generate local documentation so that I have offline access to the system manual.

**Why this priority**: Good for usability but not critical for core function.

**Independent Test**: Run `aps docs` and check filesystem.

**Acceptance Scenarios**:

1. **Given** the system is installed, **When** I run `aps docs`, **Then** the `~/.agents/docs` directory is populated with markdown files.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement a CLI with subcommands: `profile`, `run`, `action`, `docs`, `webhook`.
- **FR-002**: System MUST store state in `~/.agents` with the specified directory structure.
- **FR-003**: `aps profile new` MUST generate `profile.yaml`, `secrets.env` (0600), and optional `gitconfig`.
- **FR-004**: Core Engine MUST inject environment variables (`AGENT_PROFILE_ID`, `AGENT_PROFILE_DIR`, etc.) and secrets from `secrets.env` into executed processes.
- **FR-005**: `aps run` MUST execute arbitrary commands with the injected profile environment.
- **FR-006**: `aps action run` MUST resolve scripts (sh, js, py), inject environment, and handle stdin payloads.
- **FR-007**: Webhook server MUST validate HMAC signatures if configured.
- **FR-008**: Webhook server MUST dispatch events to mapped actions.
- **FR-009**: `aps` (no args) MUST launch the full interactive TUI application using Bubble Tea, including profile selection, action list, and execution screens.
- **FR-010**: System MUST be compiled as a single self-contained executable, with shared business logic (profiles, execution, secrets) encapsulated in an `internal/core` package used by both CLI and TUI.
- **FR-011**: Project MUST follow standard Go layout: `cmd/aps/` for the entry point and `internal/` for private implementation logic.

### Key Entities

- **Profile**: Directory containing identity, config, and secrets.
- **Action**: Script or executable within a profile that performs a task.
- **Secret**: Environment variable stored securely in `secrets.env`.
- **Webhook Event**: External trigger identified by `X-APS-Event`.

## Dependencies & Assumptions

- **Dependency**: The host system must allow execution of the binary and shell scripts.
- **Assumption**: Users have `git` and `ssh` installed for profile modules that require them.
- **Assumption**: The architectural constraints (Go language, directory layout) defined in the main `spec.md` are binding.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Application builds successfully into a single executable.
- **SC-002**: All 5 defined CLI subcommands execute and return exit code 0 on success.
- **SC-003**: Profile creation (`aps profile new`) takes less than 1 second.
- **SC-004**: `aps run` overhead (environment injection) is less than 50ms.
- **SC-005**: Webhook server successfully handles valid requests and rejects invalid signatures (401).

### Edge Cases

- **EC-001**: Profile directory manually deleted or corrupted.
- **EC-002**: `secrets.env` has permissions too open (should warn).
- **EC-003**: Action script is not executable or missing shebang.
- **EC-004**: Webhook payload exceeds memory limits (should stream or limit).