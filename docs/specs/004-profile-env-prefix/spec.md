# Feature Specification: Configurable Profile Env Var Prefix

**Feature Branch**: `004-profile-env-prefix`  
**Created**: 2026-01-15  
**Status**: Draft  
**Input**: User description: "feat: configurable profile env var prefix. Currently hardcoded to AGENT_PROFILE, should default to APS and allow user to configure in $XDG_CONFIG_HOME/aps/config.yaml"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Standardized Default Prefix (Priority: P1)

As a user, I want the CLI to use `APS` as the default prefix for profile environment variables instead of `AGENT_PROFILE` so that the environment variables are consistently named after the tool.

**Why this priority**: High value for brand consistency and following user expectations. This is a breaking change from the initial implementation but aligns with the tool's identity.

**Independent Test**: Can be tested by running an action that prints environment variables and verifying they start with `APS_` instead of `AGENT_` when no configuration is present.

**Acceptance Scenarios**:

1. **Given** no configuration file exists, **When** I run an action that executes a command, **Then** the process should have `APS_PROFILE_ID` and other `APS_` prefixed variables in its environment.
2. **Given** no configuration file exists, **When** I run an action, **Then** `AGENT_PROFILE_ID` should NO LONGER be present in the environment (unless explicitly set by the user).

---

### User Story 2 - Custom Prefix via Configuration (Priority: P2)

As a user, I want to define a custom prefix in my `config.yaml` file so that I can integrate the tool into environments with specific naming conventions or avoid collisions with other tools.

**Why this priority**: Provides flexibility for power users and enterprise environments.

**Independent Test**: Can be tested by creating a `config.yaml` with a `prefix` key and verifying that executed processes receive environment variables with that custom prefix.

**Acceptance Scenarios**:

1. **Given** a configuration file at `$XDG_CONFIG_HOME/aps/config.yaml` with `prefix: CUSTOM`, **When** I run an action, **Then** the process environment should contain `CUSTOM_PROFILE_ID`, `CUSTOM_PROFILE_DIR`, etc.
2. **Given** a custom prefix is configured, **When** I run an action, **Then** the default `APS_` prefixed variables should NOT be automatically injected (only the custom ones).

---

### User Story 3 - XDG-Compliant Configuration Discovery (Priority: P3)

As a user, I want the tool to automatically find my configuration file in the standard XDG location for my operating system so that I don't have to manage extra environment variables just to point to a config file.

**Why this priority**: Follows OS best practices (XDG Base Directory Specification).

**Independent Test**: Can be tested by placing the config file in `~/.config/aps/config.yaml` (on Darwin/Linux) and verifying it is picked up correctly.

**Acceptance Scenarios**:

1. **Given** `$XDG_CONFIG_HOME` is set to `/tmp/myconfig`, **When** I have a config file at `/tmp/myconfig/aps/config.yaml`, **Then** the tool should use the settings from that file.
2. **Given** `$XDG_CONFIG_HOME` is NOT set, **When** I have a config file at `~/.config/aps/config.yaml`, **Then** the tool should use the settings from that file as the default fallback.

---

### Edge Cases

- **Empty Prefix**: What happens when the user sets an empty prefix in `config.yaml`? (Recommendation: Fallback to default `APS` or treat as error).
- **Invalid YAML**: How does the system handle a malformed `config.yaml`? (Recommendation: Log a warning and fallback to default `APS`).
- **Permissions**: What if the config file exists but is not readable? (Recommendation: Fallback to default `APS`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST use `APS` as the default prefix for profile-related environment variables injected into child processes.
- **FR-002**: System MUST support loading configuration from `$XDG_CONFIG_HOME/aps/config.yaml`.
- **FR-003**: System MUST support a `prefix` setting in the `config.yaml` file.
- **FR-004**: System MUST override the default prefix if a valid `prefix` is specified in the configuration file.
- **FR-005**: If the configuration file is missing or invalid, the system MUST fallback to the default prefix `APS` without crashing.
- **FR-006**: The following environment variables MUST be injected using the resolved prefix:
    - `[PREFIX]_PROFILE_ID`
    - `[PREFIX]_PROFILE_DIR`
    - `[PREFIX]_PROFILE_YAML`
    - `[PREFIX]_PROFILE_SECRETS`
    - `[PREFIX]_PROFILE_DOCS_DIR`

### Key Entities *(include if feature involves data)*

- **Configuration**: A YAML file containing global settings for the `aps` CLI.
    - Attributes: `prefix` (string)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of injected profile environment variables use the correctly resolved prefix (default or configured).
- **SC-002**: Configuration is correctly loaded from the standard XDG location on Linux and Darwin.
- **SC-003**: System remains functional (using defaults) even if the configuration file is missing or unreadable.