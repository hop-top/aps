# User Stories

Consolidated user stories extracted from `specs/001-007`. Each story is a single source of truth for its requirement.

## 📋 Story Index

| ID | Title | Feature | Persona | Priority |
|----|-------|---------|---------|----------|
| [001](001-profile-management.md) | Profile Management | CLI Core (specs/001) | User | P1 |
| [002](002-command-execution.md) | Command Execution | CLI Core (specs/001) | User | P1 |
| [003](003-action-execution.md) | Action Execution | CLI Core (specs/001) | User | P2 |
| [004](004-webhook-server.md) | Webhook Server | CLI Core (specs/001) | User | P3 |
| [005](005-documentation-generation.md) | Documentation Generation | CLI Core (specs/001) | User | P3 |
| [006](006-profile-lifecycle-tests.md) | Profile Lifecycle Tests | E2E Tests (specs/002) | Maintainer | P1 |
| [007](007-execution-environment-tests.md) | Execution Environment Tests | E2E Tests (specs/002) | Maintainer | P1 |
| [008](008-action-discovery-tests.md) | Action Discovery Tests | E2E Tests (specs/002) | Maintainer | P2 |
| [009](009-webhook-integration-tests.md) | Webhook Integration Tests | E2E Tests (specs/002) | Maintainer | P3 |
| [010](010-profile-shorthand-execution.md) | Profile Shorthand Execution | Shell Integration (specs/003) | User | P1 |
| [011](011-shell-completion.md) | Shell Completion | Shell Integration (specs/003) | User | P2 |
| [012](012-alias-generation.md) | Alias Generation | Shell Integration (specs/003) | User | P3 |
| [013](013-standardized-default-prefix.md) | Standardized Default Prefix | Profile Env Prefix (specs/004) | User | P1 |
| [014](014-custom-prefix-configuration.md) | Custom Prefix Configuration | Profile Env Prefix (specs/004) | User | P2 |
| [015](015-xdg-configuration-discovery.md) | XDG Configuration Discovery | Profile Env Prefix (specs/004) | User | P3 |
| [016](016-sdk-integration-agent-cards.md) | SDK Integration & Agent Cards | A2A Protocol (specs/005) | User | P1 |
| [017](017-a2a-server.md) | A2A Server | A2A Protocol (specs/005) | User | P2 |
| [018](018-a2a-client.md) | A2A Client | A2A Protocol (specs/005) | User | P3 |
| [019](019-transport-adapters.md) | Transport Adapters | A2A Protocol (specs/005) | User | P4 |
| [020](020-a2a-cli-integration.md) | A2A CLI Integration | A2A Protocol (specs/005) | User | P5 |
| [021](021-stateless-run.md) | Stateless Run | Agent Protocol Adapter (specs/006) | External Client | P1 |
| [022](022-streaming-run.md) | Streaming Run | Agent Protocol Adapter (specs/006) | External Client | P1 |
| [023](023-run-cancellation.md) | Run Cancellation | Agent Protocol Adapter (specs/006) | External Client | P2 |
| [024](024-agent-discovery.md) | Agent Discovery | Agent Protocol Adapter (specs/006) | External Client | P2 |
| [025](025-thread-session-management.md) | Thread/Session Management | Agent Protocol Adapter (specs/006) | External Client | P3 |
| [026](026-install-capability.md) | Install Capability | Capability Management (specs/007) | User | P1 |
| [027](027-symlink-configuration.md) | Symlink Configuration | Capability Management (specs/007) | User | P1 |
| [028](028-smart-linking.md) | Smart Linking | Capability Management (specs/007) | User | P1 |
| [029](029-profile-assignment.md) | Profile Assignment | Capability Management (specs/007) | User | P2 |

## 🏷️ By Feature

### CLI Core (specs/001)
[001](001-profile-management.md), [002](002-command-execution.md), [003](003-action-execution.md), [004](004-webhook-server.md), [005](005-documentation-generation.md)

### E2E Tests (specs/002)
[006](006-profile-lifecycle-tests.md), [007](007-execution-environment-tests.md), [008](008-action-discovery-tests.md), [009](009-webhook-integration-tests.md)

### Shell Integration (specs/003)
[010](010-profile-shorthand-execution.md), [011](011-shell-completion.md), [012](012-alias-generation.md)

### Profile Env Prefix (specs/004)
[013](013-standardized-default-prefix.md), [014](014-custom-prefix-configuration.md), [015](015-xdg-configuration-discovery.md)

### A2A Protocol (specs/005)
[016](016-sdk-integration-agent-cards.md), [017](017-a2a-server.md), [018](018-a2a-client.md), [019](019-transport-adapters.md), [020](020-a2a-cli-integration.md)

### Agent Protocol Adapter (specs/006)
[021](021-stateless-run.md), [022](022-streaming-run.md), [023](023-run-cancellation.md), [024](024-agent-discovery.md), [025](025-thread-session-management.md)

### Capability Management (specs/007)
[026](026-install-capability.md), [027](027-symlink-configuration.md), [028](028-smart-linking.md), [029](029-profile-assignment.md)

## 👤 By Persona

### [User](../personas/user.md)
001-005, 010-020, 026-029

### [Maintainer](../personas/maintainer.md)
006-009

### [External Client](../personas/external-client.md)
021-025
