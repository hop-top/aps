# Maintainer

## Description

A developer or maintainer of the APS project who writes and runs automated tests to ensure the system works correctly across changes. This persona cares about test coverage, regression prevention, and the ability to verify all CLI subcommands and integrations through automated E2E tests.

## Stories

- [006 - Profile Lifecycle Tests](../stories/006-profile-lifecycle-tests.md)
- [007 - Execution Environment Tests](../stories/007-execution-environment-tests.md)
- [008 - Action Discovery Tests](../stories/008-action-discovery-tests.md)
- [009 - Webhook Integration Tests](../stories/009-webhook-integration-tests.md)
- [031 - Platform Sandbox](../stories/031-platform-sandbox.md)

## Related Stories

User stories that Maintainer test stories validate:

- [001 - Profile Management](../stories/001-profile-management.md) (tested by [006](../stories/006-profile-lifecycle-tests.md))
- [002 - Command Execution](../stories/002-command-execution.md) (tested by [007](../stories/007-execution-environment-tests.md))
- [003 - Action Execution](../stories/003-action-execution.md) (tested by [008](../stories/008-action-discovery-tests.md))
- [004 - Webhook Server](../stories/004-webhook-server.md) (tested by [009](../stories/009-webhook-integration-tests.md))
