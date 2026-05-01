# Feature Specification: Capability Management

**Feature Branch**: `007-capability-management`
**Created**: 2026-01-30
**Status**: Draft
**Input**: User request for a "Package Manager" style system that manages tools/skills/configs by symlinking them into `$APS_HOME` or injecting them into system locations.

## Overview

APS aims to be agnostic to the underlying tools (skills, commands, procedures) but capable of managing their lifecycle and configuration. This feature introduces a Capability Management system that treats `$APS_HOME` (`~/.aps`) as the central registry for all capabilities.

It supports two primary integration patterns:
1.  **Inbound Link**: Symlinking an external tool's configuration *into* `$APS_HOME` (making APS the viewer/manager of external state).
2.  **Outbound Link**: Installing a capability in `$APS_HOME` and symlinking it *out* to the tool's expected configuration location (making APS the source of truth).

## User Scenarios & Testing

### User Story 1 - Install Capability (Priority: P1)

As a user, I want to install a capability (e.g., a specific set of solid-js snippets or a git-cliff config) so that I can use it across my profiles.

**Acceptance Scenarios**:
1.  **Given** a capability `gh-cli-extensions`, **When** I run `aps capability install gh-cli-extensions`, **Then** the artifacts are downloaded to `~/.aps/capabilities/gh-cli-extensions`.
2.  **Given** an installed capability, **When** I list capabilities, **Then** it appears in the output.

### User Story 2 - Symlink Configuration (Priority: P1)

As a user, I want APS to manage my tool configurations by linking them, so that I have a central place for all my "dotfiles" and agent tools.

**Acceptance Scenarios**:
1.  **Given** an installed capability `my-vim-config` in `~/.aps/capabilities/vim`, **When** I run `aps capability link --target ~/.vimrc`, **Then** a symlink is created at `~/.vimrc` pointing to the APS managed file.
2.  **Given** existing specialized config at `~/.config/gh/config.yml`, **When** I run `aps capability adopt ~/.config/gh/config.yml --name gh-config`, **Then** the file is moved to `~/.aps/capabilities/gh-config` and a symlink is left in its place (Outbound) OR a symlink is created in `~/.aps` pointing to the original (Inbound) depending on flags.

### User Story 3 - Smart Linking (Priority: P1)

As a user, I want specific tools to be automatically configured without remembering their paths, so I can just say "link copilot" and have it work.

**Acceptance Scenarios**:
1.  **Given** a pattern registry with `copilot -> .github/agents/agent.agent.md`, **When** I run `aps capability link copilot`, **Then** the capability is linked to `.github/agents/agent.agent.md` in the current directory.
2.  **Given** I run `aps capability watch --tool windsurf`, **Then** it links the capability to `.windsurf/workflows/agent.md`.

### User Story 4 - Profile Assignment (Priority: P2)

As a user, I want to enable specific capabilities only for certain profiles, so that my "Work" profile has different tools than my "Personal" profile.

**Acceptance Scenarios**:
1.  **Given** profile `work-agent`, **When** I run `aps profile capability add work-agent git-enterprise`, **Then** the `git-enterprise` config is active when I run `aps run work-agent`.

## Requirements

### Functional Requirements

-   **FR-001**: System MUST maintain a registry of capabilities in `~/.aps/capabilities/`.
-   **FR-002**: `aps capability install <source>` MUST fetch artifacts (git repo, tarball, or local path) into the registry.
-   **FR-003**: `aps capability link <capability> <target_path>` MUST create a symlink at `<target_path>` pointing to the capability.
-   **FR-004**: `aps capability enable <profile> <capability>` MUST configure the profile to include the capability's bin/config paths in its runtime environment.
-   **FR-005**: System MUST support "Managed Mode" where it takes ownership of a target file (moves it to APS_HOME and symlinks back).
-   **FR-006**: System MUST support "Reference Mode" where it simply links an external file into APS_HOME for visibility.
-   **FR-007**: System MUST support "Smart Linking Mode":
    -   It MUST read a registry of known tool patterns (e.g., from `~/.agents/lessons/agent-configuration-patterns.md` or a config file).
    -   It MUST resolve tool names (e.g., "copilot", "claude") to their default relative paths (e.g., `.github/agents/...`, `.claude/commands/...`).
    -   It MUST create parent directories automatically when smart linking.

### Key Entities

-   **Capability**: A distinct unit of functionality.
-   **Pattern Registry**: A mapping of `tool_name -> relative_path` used for smart resolution.


## CLI Commands

```bash
# formatting: <capability_id>
aps capability list
aps capability install <source_url>
aps capability uninstall <id>

# Managing Links
aps capability link <id> --target <system_path>   # Outbound: APS -> System
aps capability adopt <system_path> --name <id>    # Import: System -> APS (mv + link)
aps capability watch <system_path> --name <id>    # Inbound: System -> APS (link only)

# Profile Association
aps profile capability add <profile> <id>
aps profile capability remove <profile> <id>
```

## Success Criteria

-   **SC-001**: Can successfully "adopt" an existing `.gitconfig` (move to APS, link back) without breaking git behavior.
-   **SC-002**: Can install a dummy "hello-world" capability and run it from within a profile.
-   **SC-003**: Symlinks are correctly cleaned up on uninstall.
