# ADR-002: Protocol Buffers for v2.0 Serialization

**Status**: Accepted (Future)
**Date**: 2026-01-20
**Related**: ADR-001 (JSON for v1.0)
**Target**: Protocol v2.0

---

## Context

After initial v1.0 implementation using JSON, performance optimization becomes important for high-throughput scenarios and production deployments.

**Challenges with v1.0 (JSON)**:
- Verbose message format (3-5x larger than binary)
- Slower parsing (5-10x slower than Protobuf)
- No compile-time type safety
- Not optimal for > 1000 messages/second

**Requirements for v2.0**:
- Improve serialization performance by 5-10x
- Reduce message size by 3-5x
- Add compile-time type safety
- Maintain backward compatibility with v1.0 clients
- Provide clear migration path

---

## Decision

**Adopt Protocol Buffers (Protobuf) for A2A Protocol v2.0 serialization**

### Implementation Details

#### v2.0 Protocol Schema (.proto)

```protobuf
syntax = "proto3";
package a2a.v2;

option go_package = "oss-aps-cli/internal/a2a/v2";

// Message envelope
message Message {
  string id = 1;                           // UUID v4
  string version = 2;                        // "2.0"
  string from = 3;
  repeated string to = 4;
  Pattern pattern = 5;
  string type = 6;
  oneof payload {
    TaskPayload task = 10;
    QueryPayload query = 11;
    EventPayload event = 12;
    BatchPayload batch = 13;
  }
  map<string, string> metadata = 7;
  int64 timestamp = 8;                      // Nanoseconds
  int64 expires_at = 9;                     // Optional
  string correl_id = 14;                     // For request/response
  string topic = 15;                         // For pub/sub
  string signature = 16;                     // HMAC-SHA256
}

// Message patterns
enum Pattern {
  REQUEST = 0;
  RESPONSE = 1;
  PUBLISH = 2;
  SUBSCRIBE = 3;
  BATCH = 4;
}

// Payload types
message TaskPayload {
  string command = 1;
  map<string, string> args = 2;
}

message QueryPayload {
  string query = 1;
  map<string, string> filters = 2;
}

message EventPayload {
  string event_name = 1;
  map<string, string> event_data = 2;
}

message BatchPayload {
  repeated Message messages = 1;
}
```

#### Version Negotiation

```
Request Headers:
  Accept: application/protobuf, application/json
  Content-Type: application/protobuf

Response Headers:
  Content-Type: application/protobuf  // If client accepts Protobuf
  Content-Type: application/json     // If client only accepts JSON
```

#### Dual-Support Period

- **Duration**: 6 months from v2.0 GA release
- **Behavior**:
  - v1.0 clients send/receive JSON only
  - v2.0 clients send/receive Protobuf by default
  - v2.0 servers support both JSON and Protobuf
  - Automatic negotiation based on `Accept` headers
- **Post-migration**: Deprecate JSON support in v2.1

---

## Consequences

### Positive

- **Performance**: 5-10x faster serialization/deserialization
- **Size**: 3-5x smaller message size
- **Type Safety**: Compile-time schema validation
- **Code Generation**: Auto-generated client/server stubs
- **Cross-language**: Generated code for Go, Python, JavaScript, etc.
- **Backward Compatibility**: Dual support period ensures smooth migration
- **Production Ready**: Battle-tested by Google, used at scale

### Negative

- **Complexity**: Requires protoc compiler and build toolchain
- **Breaking Change**: v1.0 clients need migration
- **Schema Management**: Need to maintain .proto files and versions
- **Binary Format**: Not human-readable (harder debugging)
- **Migration Effort**: 6-month dual-support period
- **Learning Curve**: Team needs to learn Protobuf

---

## Alternatives Considered

### Continue with JSON for v2.0 ❌

**Pros**:
- No breaking changes
- No migration effort
- Human-readable

**Cons**:
- No performance improvement
- Verbose format remains
- No type safety

**Rejection**: Performance requirements not met

---

### CBOR for v2.0 ❌

**Pros**:
- Binary format (compact)
- Self-describing (no schema)
- Faster than JSON

**Cons**:
- Less mature than Protobuf
- No code generation
- Limited tooling
- Still no type safety

**Rejection**: Protobuf offers better tooling and type safety

---

### FlatBuffers for v2.0 ❌

**Pros**:
- Zero-copy parsing (fastest)
- Schema evolution support
- Cross-language support

**Cons**:
- Less mature than Protobuf
- Smaller ecosystem
- Steeper learning curve

**Rejection**: Protobuf has better ecosystem and adoption

---

### Cap'n Proto for v2.0 ❌

**Pros**:
- Zero-copy parsing
- RPC support built-in
- Modern design

**Cons**:
- Smaller ecosystem than Protobuf
- Less adoption
- Less mature

**Rejection**: Protobuf has better ecosystem and industry adoption

---

## Migration Strategy

### Phase 1: Development (Months 1-2)
- Develop v2.0 with Protobuf support
- Implement dual JSON/Protobuf support in servers
- Create migration documentation
- Test backward compatibility

### Phase 2: Alpha/Beta (Months 3-4)
- Release v2.0-alpha with dual support
- Gather feedback on performance improvements
- Benchmark v1.0 vs v2.0
- Refine migration guide

### Phase 3: GA & Migration (Months 5-10)
- Release v2.0-GA
- Begin 6-month dual-support period
- Support both v1.0 and v2.0 clients
- Monitor migration metrics

### Phase 4: Deprecation (Months 11-12)
- Deprecate JSON support
- Release v2.1 (Protobuf-only)
- Provide final migration push
- Remove JSON code in v2.2

### Migration Example

```go
// v1.0 client (JSON)
client := a2a.NewClient(a2a.WithVersion("1.0"))
client.SendMessage(msg)  // Uses JSON

// v2.0 client (Protobuf)
client := a2a.NewClient(a2a.WithVersion("2.0"))
client.SendMessage(msg)  // Uses Protobuf

// v2.0 client with JSON fallback
client := a2a.NewClient(a2a.WithVersion("2.0"))
client.SetFallbackVersion("1.0")
client.SendMessage(msg)  // Uses Protobuf, falls back to JSON
```

---

## Performance Targets

### v1.0 (JSON)
- **Serialization**: ~1ms per message
- **Deserialization**: ~1ms per message
- **Message Size**: ~1 KB average

### v2.0 (Protobuf)
- **Serialization**: ~0.1ms per message (10x improvement)
- **Deserialization**: ~0.1ms per message (10x improvement)
- **Message Size**: ~300 bytes average (3x improvement)

### Benchmarks

Run benchmarks to verify:
```
BenchmarkJSONSerialize-8    1000000  1000 ns/op
BenchmarkProtobufSerialize-8 10000000   100 ns/op

BenchmarkJSONDeserialize-8  1000000  1000 ns/op
BenchmarkProtobufDeserialize-8 10000000   100 ns/op
```

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (baseline)
- **ADR-008**: Batch messages for v1.1 (related optimization)
- **ADR-005**: Message references for large payloads (complementary)

---

## References

- **Specification**: `spec.md` - Future Versions section
- **Protocol Buffers**: https://protobuf.dev/
- **Decisions Document**: `decisions.md` - Question #1

---

## Revisions

- 2026-01-20: Initial decision - Protobuf for v2.0
- 2026-01-20: Added migration strategy and performance targets
