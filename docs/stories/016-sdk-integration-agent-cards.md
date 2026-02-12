# SDK Integration & Agent Cards

**ID**: 016
**Feature**: A2A Protocol (specs/005)
**Persona**: [User](../personas/user.md)
**Priority**: P1

## Story

As a user, I want to integrate the a2a-go SDK and generate Agent Cards for my APS profiles so that other A2A-compliant agents can discover and interact with them.

## Acceptance Scenarios

1. **Given** a profile with A2A enabled, **When** I generate an Agent Card, **Then** the card is valid per the A2A specification.
2. **Given** a profile with capabilities configured, **When** the Agent Card is generated, **Then** it reflects the profile's capabilities and security schemes.

## Tests

### Unit
- `internal/a2a/agentcard_generation_test.go` — `TestGenerateAgentCardFromProfile_Enabled`, `TestGenerateAgentCardFromProfile_Disabled`, `TestGenerateAgentCardFromProfile_NoA2AConfig`, `TestGenerateAgentCardForProfile`, `TestGenerateAgentCardForProfile_InvalidID`
- `internal/a2a/agentcard_validation_test.go` — `TestValidateAgentCard_ValidCard`, `TestValidateAgentCard_MissingName`, `TestValidateAgentCard_MissingURL`, `TestValidateAgentCard_EmptySkills`
- `internal/a2a/config_test.go`
- `internal/a2a/cache_test.go`
