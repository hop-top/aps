# Feature Specification: Shell Integration & Shorthands

**Feature Branch**: `003-shell-integration`
**Created**: 2026-01-15
**Status**: Draft
**Input**: User description: "Implement shell integration including zsh auto-completion, root command shorthand for profiles (aps <profile>), and profile aliasing generation."

## Clarifications

### Session 2026-01-15

- Q: How should alias conflict detection work? → A: System PATH Check: Warn if `exec.LookPath(alias)` succeeds. Prevents shadowing system commands.
- Q: What should be the default shell for `aps <profile>` session? → A: Configurable: Define the default shell in `profile.yaml` (defaulting to user's `$SHELL` during profile creation).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Profile Shorthand Execution (Priority: P1)

As a user, I want to execute commands in a profile without typing `run` or `--` so that I can work faster.

**Why this priority**: Improves daily UX significantly by reducing typing.

**Independent Test**:
1. Create profile `shorthand-agent`.
2. Run `aps shorthand-agent env`.
3. Verify output contains profile injection.

**Acceptance Scenarios**:

1. **Given** profile `agent-a`, **When** I run `aps agent-a`, **Then** it launches the interactive TUI or shell (equivalent to `aps run agent-a -- $SHELL`? No, user said "loads new session (like #2)").
   *Clarification needed: Does `aps agent-a` launch the TUI or a sub-shell? User said "loads new session". `aps run <profile>` currently fails without command. `aps run <profile> -- <shell>` is the pattern for session.*
   *Let's assume `aps agent-a` (no args) -> `aps run agent-a -- $SHELL` (interactive session).*
2. **Given** profile `agent-a`, **When** I run `aps agent-a git status`, **Then** it executes `git status` in the profile context (equivalent to `aps run agent-a -- git status`).
3. **Given** a subcommand `profile`, **When** I run `aps profile`, **Then** it executes the `profile` subcommand, NOT a profile named "profile".

---

### User Story 2 - Shell Completion (Priority: P2)

As a user, I want tab completion for profiles so that I can easily select the right agent.

**Why this priority**: Standard CLI expectation.

**Independent Test**: Source generated completion script and verify `aps <tab>` suggests profiles.

**Acceptance Scenarios**:

1. **Given** valid profiles, **When** I trigger completion on `aps [TAB]`, **Then** profile IDs are suggested alongside subcommands.
2. **Given** `aps agent-a [TAB]`, **When** I trigger completion, **Then** it suggests commands or files (standard shell behavior).

---

### User Story 3 - Alias Generation (Priority: P3)

As a user, I want to generate shell aliases for my profiles so I can invoke them directly by name.

**Why this priority**: Power user feature requested by user.

**Independent Test**: Run `aps alias` and check output.

**Acceptance Scenarios**:

1. **Given** profile `agent-a`, **When** I run `aps alias`, **Then** output includes `alias agent-a='aps agent-a'`.
2. **Given** a collision (profile name same as system command, e.g., `git`), **When** I run `aps alias`, **Then** it warns or skips?
   *Clarification needed: User said "if no conflicts detected". APS checks conflicts with what? System path? Aliases?*
   *Simpler approach: Just output the aliases and let the user decide/filter. Or maybe a `--check` flag?*
   *Let's assume basic generation first.*

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Root command MUST detect if the first argument is a valid profile ID.
- **FR-002**: If first arg is a profile ID, `aps <profile> [args...]` MUST behave exactly like `aps run <profile> -- [args...]`.
- **FR-003**: If first arg is a profile ID and no other args provided, it MUST launch the shell defined in the profile's `profile.yaml` (falling back to `$SHELL` if not set).
- **FR-007**: `aps profile new` MUST capture the current `$SHELL` and store it in the new profile's configuration.
- **FR-004**: System MUST implement `aps completion [bash|zsh|fish|powershell]` command (leveraging Cobra).
- **FR-005**: Completion MUST include dynamic profile names.
- **FR-006**: System MUST implement `aps alias` command to output alias definitions.

### Key Entities

- **CommandResolver**: Logic to distinguish between Subcommand vs Profile vs Unknown.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `aps agent-a` starts a shell session.
- **SC-002**: `aps agent-a whoami` runs the command.
- **SC-003**: `aps completion zsh` generates valid Zsh script.

### Edge Cases

- **EC-001**: Profile name overlaps with existing subcommand (e.g., "run", "profile"). Subcommand MUST take precedence.
- **EC-002**: Profile name overlaps with flag (e.g., "--help"). Flags MUST be handled by root command.