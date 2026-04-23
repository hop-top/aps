# Tasks: Shell Integration & Shorthands

**Feature Branch**: `003-shell-integration`
**Spec**: [specs/003-shell-integration/spec.md](../spec.md)

## Phase 1: Setup & Infrastructure

**Goal**: Establish core shell detection and validation logic.

- [x] T001 Create `internal/core/shell.go` with `DetectShell()` and `IsCommandAvailable(name)` logic

## Phase 2: Foundation (Shared Core)

**Goal**: Update Profile data model to support shell preferences.

- [x] T002 Update `Profile` struct in `internal/core/profile.go` to add `Shell` to `Preferences`
- [x] T003 Update `CreateProfile` in `internal/core/profile.go` to populate `Shell` preference using `DetectShell()`

## Phase 3: Profile Shorthand (User Story 1)

**Goal**: Enable `aps <profile>` shorthand for execution and sessions.
**Independent Test**: `aps agent-a` starts shell, `aps agent-a cmd` runs command.

- [x] T004 [US1] Refactor `rootCmd` in `internal/cli/root.go` to accept `ArbitraryArgs`
- [x] T005 [US1] Implement dispatch logic in `internal/cli/root.go`: check if arg[0] is profile ID vs TUI launch
- [x] T006 [US1] Implement session launch logic in `internal/cli/root.go` (loading profile's preferred shell) for no-args case

## Phase 4: Shell Completion (User Story 2)

**Goal**: Enable tab completion for profiles and commands.
**Independent Test**: `aps completion zsh` outputs valid script.

- [x] T007 [US2] Create `internal/cli/completion.go` with standard Cobra completion command
- [x] T008 [US2] Add `ValidArgsFunction` to `rootCmd` in `internal/cli/root.go` to autocomplete profile IDs

## Phase 5: Alias Generation (User Story 3)

**Goal**: Generate shell aliases for quick access.
**Independent Test**: `aps alias` outputs `alias name='aps name'`.

- [x] T009 [US3] Create `internal/cli/alias.go` command
- [x] T010 [US3] Implement alias generation logic with conflict warning (using `core.IsCommandAvailable`)

## Phase 6: Polish

**Goal**: Verify and finalize.

- [x] T011 Verify build and manual test of alias generation and conflicts
- [x] T012 Update README with shell integration instructions

## Dependencies

- Phase 1 & 2 are blocking for Phase 3 & 5.
- Phase 3 logic (root command changes) affects Phase 4 (completion context).
- Phase 5 depends on Phase 1 logic (`IsCommandAvailable`).

## Implementation Strategy

- **MVP**: Shorthand execution (Phase 1, 2, 3) is the highest value.
- **Increment 1**: Add Completion (Phase 4).
- **Increment 2**: Add Aliases (Phase 5).
