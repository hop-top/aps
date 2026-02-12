# User

## Description

The end-user of APS who creates agent profiles, executes commands and actions within profile contexts, configures shell integrations, and manages capabilities. This persona cares about productivity, ease of use, and the ability to maintain isolated identities and configurations for different agent workflows.

## Stories

- [001 - Profile Management](../stories/001-profile-management.md)
- [002 - Command Execution](../stories/002-command-execution.md)
- [003 - Action Execution](../stories/003-action-execution.md)
- [004 - Webhook Server](../stories/004-webhook-server.md)
- [005 - Documentation Generation](../stories/005-documentation-generation.md)
- [010 - Profile Shorthand Execution](../stories/010-profile-shorthand-execution.md)
- [011 - Shell Completion](../stories/011-shell-completion.md)
- [012 - Alias Generation](../stories/012-alias-generation.md)
- [013 - Standardized Default Prefix](../stories/013-standardized-default-prefix.md)
- [014 - Custom Prefix Configuration](../stories/014-custom-prefix-configuration.md)
- [015 - XDG Configuration Discovery](../stories/015-xdg-configuration-discovery.md)
- [016 - SDK Integration & Agent Cards](../stories/016-sdk-integration-agent-cards.md)
- [017 - A2A Server](../stories/017-a2a-server.md)
- [018 - A2A Client](../stories/018-a2a-client.md)
- [019 - Transport Adapters](../stories/019-transport-adapters.md)
- [020 - A2A CLI Integration](../stories/020-a2a-cli-integration.md)
- [026 - Install Capability](../stories/026-install-capability.md)
- [027 - Symlink Configuration](../stories/027-symlink-configuration.md)
- [028 - Smart Linking](../stories/028-smart-linking.md)
- [029 - Profile Assignment](../stories/029-profile-assignment.md)

## Related Stories

Stories where User functionality is validated by another persona:

- [006 - Profile Lifecycle Tests](../stories/006-profile-lifecycle-tests.md) (Maintainer validates [001](../stories/001-profile-management.md))
- [007 - Execution Environment Tests](../stories/007-execution-environment-tests.md) (Maintainer validates [002](../stories/002-command-execution.md))
- [008 - Action Discovery Tests](../stories/008-action-discovery-tests.md) (Maintainer validates [003](../stories/003-action-execution.md))
- [009 - Webhook Integration Tests](../stories/009-webhook-integration-tests.md) (Maintainer validates [004](../stories/004-webhook-server.md))
