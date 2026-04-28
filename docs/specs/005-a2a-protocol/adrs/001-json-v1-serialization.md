# ADR-001: JSON for v1.0 Serialization

**Status**: Accepted
**Date**: 2026-01-20
**Related**: ADR-002 (Protocol Buffers for v2.0)

---

## Context

The A2A protocol requires a message serialization format. Multiple options exist, each with trade-offs between performance, complexity, and usability.

**Options Considered**:
- JSON: Human-readable, widely supported, verbose
- Protocol Buffers: Fast, compact, binary, requires schema
- CBOR: Fast, compact, binary, self-describing

**Requirements**:
- Must be easy to implement and debug
- Must be transport-agnostic
- Should support versioning and evolution
- Should balance performance with usability for v1.0

---

## Decision

**Use JSON for A2A Protocol v1.0 serialization**

### Implementation Details

- All A2A messages are UTF-8 encoded JSON objects
- Canonical sorting: Object keys sorted alphabetically for signatures
- No external dependencies (standard library JSON parsing)
- Max message size: 1 MB for inline payloads

### Example Message Format

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "task",
  "payload": {"command": "deploy"},
  "timestamp": 1705773600000000000
}
```

---

## Consequences

### Positive

- **Human-readable**: Easy to debug and inspect messages manually
- **No external dependencies**: JSON parsing available in all major languages
- **Wide adoption**: Developers familiar with JSON, low learning curve
- **Simple tooling**: Standard JSON validators, formatters, diff tools work
- **Flexible**: Schema-less (for v1.0), easy evolution
- **Transport-agnostic**: JSON works over IPC, HTTP, WebSocket equally well

### Negative

- **Verbose**: JSON is larger than binary formats (3-5x overhead)
- **Slower parsing**: JSON parsing slower than Protobuf (5-10x)
- **No type safety**: No compile-time schema validation
- **Limited large-scale performance**: Not optimal for high-throughput (>10k msg/s)

---

## Alternatives Considered

### Protocol Buffers for v1.0 ❌

**Pros**:
- Fast serialization/deserialization (5-10x JSON)
- Compact size (3-5x smaller)
- Strong typing and code generation

**Cons**:
- Requires `.proto` schema files
- Not human-readable
- External dependency (protoc compiler)
- More complex toolchain
- Overkill for v1.0 requirements

**Rejection**: Too complex for initial implementation, planned for v2.0

---

### CBOR (Concise Binary Object Representation) ❌

**Pros**:
- Binary format (compact)
- Self-describing (no schema required)
- Faster than JSON

**Cons**:
- Less widely adopted than JSON
- More complex implementation
- Binary format harder to debug
- Limited tooling compared to JSON

**Rejection**: Not enough benefit over JSON for v1.0

---

### MessagePack ❌

**Pros**:
- Binary format (compact)
- JSON-compatible schema
- Faster than JSON

**Cons**:
- Less widely adopted than JSON
- Binary format harder to debug
- Less mature tooling

**Rejection**: Not enough benefit over JSON for v1.0

---

## Migration Path

### v1.0 → v1.1
- Continue using JSON (no breaking changes)
- Add optional fields to message schema

### v1.1 → v2.0
- Introduce Protocol Buffers support
- Dual-support period (JSON + Protobuf) for 6 months
- v2.0+ clients default to Protobuf
- v1.0 clients continue using JSON
- Automatic negotiation via `Accept` and `Content-Type` headers

### Example v2.0 Migration

```
v1.0 Client → v2.0 Server:
  Accept: application/json
  → Server responds with JSON (backward compatible)

v2.0 Client → v2.0 Server:
  Accept: application/protobuf, application/json
  Content-Type: application/protobuf
  → Server responds with Protobuf (optimal)

v2.0 Client → v1.0 Server:
  Accept: application/protobuf, application/json
  Content-Type: application/protobuf
  → Server returns error (v1.0 doesn't support Protobuf)
  → Client falls back to JSON
```

---

## Related Decisions

- **ADR-002**: Protocol Buffers for v2.0 (performance optimization)
- **ADR-005**: Message references for large payloads (mitigates JSON verbosity)
- **ADR-004**: Per-message compression (mitigates JSON verbosity)

---

## References

- **Specification**: `spec.md` - Message Format section
- **Decisions Document**: `decisions.md` - Question #1

---

## Revisions

- 2026-01-20: Initial decision - JSON for v1.0
- 2026-01-20: Added migration path to v2.0
