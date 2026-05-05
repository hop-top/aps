# User Stories

Consolidated user stories. Each story is a single source of truth for its requirement.

Authoring convention: see [Story → e2e test linkage](../conventions/stories.md).

## Story Index

| ID | Title | Feature | Persona | Priority |
|----|-------|---------|---------|----------|
| [001](001-profile-management.md) | Profile Management | CLI Core | User | P1 |
| [002](002-command-execution.md) | Command Execution | CLI Core | User | P1 |
| [003](003-action-execution.md) | Action Execution | CLI Core | User | P2 |
| [004](004-webhook-server.md) | Webhook Server | CLI Core | User | P3 |
| [005](005-documentation-generation.md) | Documentation Generation | CLI Core | User | P3 |
| [006](006-profile-lifecycle-tests.md) | Profile Lifecycle Tests | E2E Tests | Maintainer | P1 |
| [007](007-execution-environment-tests.md) | Execution Environment Tests | E2E Tests | Maintainer | P1 |
| [008](008-action-discovery-tests.md) | Action Discovery Tests | E2E Tests | Maintainer | P2 |
| [009](009-webhook-integration-tests.md) | Webhook Integration Tests | E2E Tests | Maintainer | P3 |
| [010](010-profile-shorthand-execution.md) | Profile Shorthand Execution | Shell Integration | User | P1 |
| [011](011-shell-completion.md) | Shell Completion | Shell Integration | User | P2 |
| [012](012-alias-generation.md) | Alias Generation | Shell Integration | User | P3 |
| [013](013-standardized-default-prefix.md) | Standardized Default Prefix | Profile Env Prefix | User | P1 |
| [014](014-custom-prefix-configuration.md) | Custom Prefix Configuration | Profile Env Prefix | User | P2 |
| [015](015-xdg-configuration-discovery.md) | XDG Configuration Discovery | Profile Env Prefix | User | P3 |
| [016](016-sdk-integration-agent-cards.md) | SDK Integration & Agent Cards | A2A Protocol | User | P1 |
| [017](017-a2a-server.md) | A2A Server | A2A Protocol | User | P2 |
| [018](018-a2a-client.md) | A2A Client | A2A Protocol | User | P3 |
| [019](019-transport-adapters.md) | Transport Adapters | A2A Protocol | User | P4 |
| [020](020-a2a-cli-integration.md) | A2A CLI Integration | A2A Protocol | User | P5 |
| [021](021-stateless-run.md) | Stateless Run | Agent Protocol Adapter | External Client | P1 |
| [022](022-streaming-run.md) | Streaming Run | Agent Protocol Adapter | External Client | P1 |
| [023](023-run-cancellation.md) | Run Cancellation | Agent Protocol Adapter | External Client | P2 |
| [024](024-agent-discovery.md) | Agent Discovery | Agent Protocol Adapter | External Client | P2 |
| [025](025-thread-session-management.md) | Thread/Session Management | Agent Protocol Adapter | External Client | P3 |
| [026](026-install-capability.md) | Install Capability | Capability Management | User | P1 |
| [027](027-symlink-configuration.md) | Symlink Configuration | Capability Management | User | P1 |
| [028](028-smart-linking.md) | Smart Linking | Capability Management | User | P1 |
| [029](029-profile-assignment.md) | Profile Assignment | Capability Management | User | P2 |
| [030](030-execution-isolation.md) | Execution Isolation | Execution Isolation | User | P1 |
| [031](031-platform-sandbox.md) | Platform Sandbox | Execution Isolation | Maintainer | P1 |
| [032](032-key-value-store.md) | Key-Value Store | Agent Protocol Adapter | External Client | P2 |
| [033](033-thread-scoped-runs.md) | Thread-Scoped Runs | Agent Protocol Adapter | External Client | P3 |
| [034](034-custom-profile-tools.md) | Custom Profile Tools | CLI Core | User | P2 |
| [035](035-protocol-server.md) | Protocol Server | Agent Protocol Adapter | User | P2 |
| [036](036-capability-environment.md) | Capability Environment | Capability Management | User | P2 |
| [037](037-a2a-protocol-toggle.md) | A2A Protocol Toggle | A2A Protocol | User | P2 |
| [038](038-acp-protocol-toggle.md) | ACP Protocol Toggle | Agent Protocol Adapter | User | P2 |
| [039](039-webhook-protocol-toggle.md) | Webhook Protocol Toggle | CLI Core | User | P2 |
| [040](040-voice-profile-configuration.md) | Voice Profile Configuration | Voice | User | P2 |
| [041](041-voice-backend-service.md) | Voice Backend Service | Voice | User | P2 |
| [042](042-voice-sessions.md) | Voice Sessions | Voice | User | P2 |
| [050](050-multi-device-workspace-access.md) | Multi-Device Workspace Access | Multi-Device Workspace | User | P2 |
| [055](055-progress-on-long-running-ops.md) | Structured Progress on Long-Running Ops | CLI Core | User | P2 |

## By Feature

### CLI Core
[001](001-profile-management.md), [002](002-command-execution.md), [003](003-action-execution.md), [004](004-webhook-server.md), [005](005-documentation-generation.md), [034](034-custom-profile-tools.md), [039](039-webhook-protocol-toggle.md), [055](055-progress-on-long-running-ops.md)

### E2E Tests
[006](006-profile-lifecycle-tests.md), [007](007-execution-environment-tests.md), [008](008-action-discovery-tests.md), [009](009-webhook-integration-tests.md)

### Shell Integration
[010](010-profile-shorthand-execution.md), [011](011-shell-completion.md), [012](012-alias-generation.md)

### Profile Env Prefix
[013](013-standardized-default-prefix.md), [014](014-custom-prefix-configuration.md), [015](015-xdg-configuration-discovery.md)

### A2A Protocol
[016](016-sdk-integration-agent-cards.md), [017](017-a2a-server.md), [018](018-a2a-client.md), [019](019-transport-adapters.md), [020](020-a2a-cli-integration.md), [037](037-a2a-protocol-toggle.md)

### Agent Protocol Adapter
[021](021-stateless-run.md), [022](022-streaming-run.md), [023](023-run-cancellation.md), [024](024-agent-discovery.md), [025](025-thread-session-management.md), [032](032-key-value-store.md), [033](033-thread-scoped-runs.md), [035](035-protocol-server.md), [038](038-acp-protocol-toggle.md)

### Capability Management
[026](026-install-capability.md), [027](027-symlink-configuration.md), [028](028-smart-linking.md), [029](029-profile-assignment.md), [036](036-capability-environment.md)

### Execution Isolation
[030](030-execution-isolation.md), [031](031-platform-sandbox.md)

### Voice
[040](040-voice-profile-configuration.md), [041](041-voice-backend-service.md), [042](042-voice-sessions.md)

### Multi-Device Workspace
[050](050-multi-device-workspace-access.md)

## By Persona

### [User](../personas/user.md)
001-005, 010-020, 026-030, 034-042, 050

### [Maintainer](../personas/maintainer.md)
006-009, 031

### [External Client](../personas/external-client.md)
021-025, 032-033
