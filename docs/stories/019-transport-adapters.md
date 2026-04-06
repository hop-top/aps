# Transport Adapters

**ID**: 019
**Feature**: A2A Protocol
**Persona**: [User](../personas/user.md)
**Priority**: P4

## Story

As a user, I want A2A communication to use the appropriate transport (IPC, HTTP, gRPC) based on the isolation tier so that agents can communicate efficiently across different deployment contexts.

## Acceptance Scenarios

1. **Given** two profiles in the same isolation tier, **When** they communicate via A2A, **Then** IPC transport is used for efficiency.
2. **Given** two profiles in different isolation tiers, **When** they communicate via A2A, **Then** HTTP or gRPC transport is used with appropriate authentication.

## Tests

### Unit
- `tests/unit/a2a/transport/ipc_test.go` — `TestNewIPCTransport_ValidConfig`, `TestIPCTransport_SendMessage`, `TestIPCTransport_ReceiveMessage`, `TestIPCTransport_StartStop`
- `tests/unit/a2a/transport/http_test.go` — `TestNewHTTPTransport_ValidConfig`, `TestNewHTTPTransport_SendMessage`, `TestNewHTTPTransport_IsHealthy`
- `tests/unit/a2a/transport/grpc_test.go` — `TestNewGRPCTransport_ValidConfig`, `TestNewGRPCTransport_SendMessage`, `TestNewGRPCTransport_IsHealthy`
- `tests/unit/a2a/transport/selector_test.go` — `TestSelectTransport_ProcessTier`, `TestSelectTransport_PlatformTier`, `TestSelectTransport_ContainerTier`, `TestGetFallbackTransport_IPCToHTTP`

### E2E
- `tests/e2e/a2a_transport_test.go` — `TestCrossTierCommunication_IPC`, `TestCrossTierCommunication_TransportSelection`, `TestCrossTierCommunication_Fallback`
