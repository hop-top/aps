# ADR-003: UUID v4 for Message IDs

**Status**: Accepted
**Date**: 2026-01-20

---

## Context

The A2A protocol requires unique identifiers for messages to enable correlation, deduplication, and tracking.

**Requirements**:
- Must be unique across all messages and profiles
- Must be globally unique (for network communication)
- Should be collision-resistant
- Should be standard and widely supported
- Should support ordering (for conversation history)

---

## Decision

**Use UUID v4 (random) for message identifiers**

### Implementation Details

- **Algorithm**: UUID v4 (random 122-bit + fixed 4-bit version + 2-bit variant)
- **Format**: `xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx` (hexadecimal)
- **Collision Probability**: ~1 in 5.3×10^36 (negligible)
- **Generation**: Standard library (e.g., `google/uuid` in Go)
- **Ordering**: Use separate `timestamp` field for ordering

### Examples

```
Message ID: "550e8400-e29b-41d4-a716-446655440000"
Message ID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
Message ID: "6fa459ea-ee8a-3ca4-894e-db77e160355e"
```

### Message Structure

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": 1705773600000000000,
  "from": "agent-a",
  ...
}
```

---

## Consequences

### Positive

- **Universally Unique**: Globally unique across all profiles and machines
- **Collision-Resistant**: Negligible collision probability
- **Standard Format**: Well-known, widely supported in all languages
- **No Coordination**: No centralized ID generation needed
- **No Metadata**: Doesn't reveal information about message (random)
- **Efficient**: Fast generation (~microseconds)
- **Portable**: Same format across platforms and languages

### Negative

- **Not Sortable**: UUIDs don't sort chronologically
- **Longer String**: 36 characters (vs. 19 for timestamp-based)
- **Requires Timestamp**: Need separate field for ordering

---

## Alternatives Considered

### 1. Timestamp-Based IDs ❌

**Format**: Unix timestamp (nanoseconds) + counter

**Example**: `1705773600000000000-001`

**Pros**:
- **Sortable**: Naturally chronological
- **Shorter**: 19-22 characters
- **Human-readable**: Easy to read and compare
- **Efficient**: No additional timestamp field needed

**Cons**:
- **Predictable**: Can guess future IDs (security risk)
- **Collision Risk**: Multiple messages in same nanosecond need counter
- **Clock Sync**: Requires synchronized clocks across machines
- **Monotonicity**: Clock skew can cause ordering issues
- **Coordination**: Needs counter coordination per profile

**Rejection**: Predictability is security concern, clock sync adds complexity

---

### 2. UUID v7 (Time-Sorted) ❌

**Format**: UUID v7 (timestamp + random)

**Example**: `0189a6d1-d591-7b8e-8000-2d6b9b6c8f5b`

**Pros**:
- **Sortable**: Chronologically ordered
- **Unique**: Same collision resistance as UUID v4
- **No Clock Sync**: Embeds timestamp in ID
- **Single Field**: No separate timestamp needed

**Cons**:
- **Not Standard**: UUID v7 is draft, not yet RFC standard
- **Less Support**: Fewer libraries implement UUID v7
- **Less Familiar**: Developers less familiar with format
- **Complex Generation**: Requires timestamp extraction + random generation
- **Newer**: Less battle-tested than UUID v4

**Rejection**: Not yet standard, fewer library implementations

---

### 3. UUID v1 (Time-Based) ❌

**Format**: UUID v1 (timestamp + MAC address + sequence)

**Example**: `6ba7b810-9dad-11d1-80b4-00c04fd430c8`

**Pros**:
- **Sortable**: Time-based component enables sorting
- **Standard**: RFC 4122 standard
- **Unique**: Guaranteed uniqueness (timestamp + MAC + sequence)

**Cons**:
- **Privacy Leak**: Exposes MAC address of generating machine
- **Requires Clock Sync**: Needs accurate system clock
- **Sequence Coordination**: Requires monotonically increasing counter
- **Complex Generation**: More complex than UUID v4
- **MAC Address**: Changes when hardware changes
- **Predictability**: Next IDs can be guessed (security risk)

**Rejection**: Privacy and security concerns with MAC address exposure

---

### 4. UUID v6 (Time-Sorted) ❌

**Format**: UUID v6 (reordered v1 timestamps)

**Example**: `1ec9414c-232a-6b00-b3d6-001bb62dc3f4`

**Pros**:
- **Sortable**: Time-based and sortable
- **Standard**: Draft standard
- **Better Privacy**: Doesn't expose MAC address

**Cons**:
- **Not Final**: Still draft, not yet final RFC
- **Complexity**: More complex generation than v4
- **Less Support**: Fewer libraries implement UUID v6
- **Same Issues**: Still requires clock and sequence

**Rejection**: Draft status, more complex than needed

---

### 5. NanoID ❌

**Format**: 21-character URL-safe random string

**Example**: `V1StGXR8_Z5jdHi6B-myT`

**Pros**:
- **Shorter**: 21 characters (vs. 36 for UUID)
- **URL-Safe**: Uses URL-safe alphabet
- **Customizable**: Can adjust length and alphabet
- **Collision-Resistant**: Configurable collision probability

**Cons**:
- **Non-Standard**: Custom format, not a standard
- **Less Portable**: Requires nanoID library
- **Collision Risk**: Higher than UUID if too short
- **No Versioning**: No built-in version/type information
- **Less Familiar**: Developers less familiar than UUID

**Rejection**: Non-standard, UUID is more widely adopted

---

### 6. ULID (Universally Unique Lexicographically Sortable ID) ❌

**Format**: 26-character base32 encoded timestamp + random

**Example**: `01FYZD2BSRQMTA7P8QYTS7A0H`

**Pros**:
- **Sortable**: Lexicographically sortable by time
- **URL-Safe**: Base32 encoding
- **Compact**: 26 characters (vs. 36 for UUID)
- **Standard**: Draft standard gaining adoption
- **Efficient**: Fast generation

**Cons**:
- **Less Standard**: Still gaining adoption vs. UUID
- **Less Support**: Fewer library implementations than UUID
- **Base32**: Less familiar than hex
- **Newer**: Less battle-tested than UUID v4

**Rejection**: UUID v4 is more standard and battle-tested

---

### 7. Snowflake (Twitter ID) ❌

**Format**: 64-bit integer (timestamp + worker ID + sequence)

**Example**: `1234567890123456789` (converted to string)

**Pros**:
- **Sortable**: Chronologically sortable
- **Compact**: 64-bit integer
- **Distributed**: Designed for distributed systems
- **Efficient**: Integer operations are fast

**Cons**:
- **Requires Coordination**: Needs worker ID assignment
- **Epoch Configuration**: Requires custom epoch timestamp
- **Sequence Management**: Needs sequence coordination per worker
- **Time Dependency**: Sensitive to clock skew
- **Integer Overflow**: Can overflow after ~69 years
- **Coordination Complexity**: Requires coordination service

**Rejection**: Too much coordination overhead, not needed for v1.0

---

### 8. Custom Format (Profile + Timestamp + Random) ❌

**Format**: `<profile-id>-<timestamp>-<random>`

**Example**: `agent-a-1705773600000000000-a1b2c3d4`

**Pros**:
- **Traceable**: Profile ID visible in message ID
- **Sortable**: Timestamp enables sorting
- **Customizable**: Can adjust format for use case

**Cons**:
- **Non-Standard**: Custom format
- **Privacy Leak**: Exposes profile ID
- **Coordination**: Requires coordination per profile
- **Collision Risk**: Needs random component
- **Complex Parsing**: Need to parse custom format

**Rejection**: Privacy concern, unnecessary complexity

---

## Comparison Summary

| Option | Length | Sortable | Standard | Collision Risk | Complexity |
|--------|---------|-----------|-----------|----------------|-------------|
| **UUID v4** ✅ | 36 | No | Yes | Negligible | Low |
| Timestamp + Counter | 19-22 | Yes | Custom | High | Medium |
| UUID v7 | 36 | Yes | Draft | Negligible | Medium |
| UUID v1 | 36 | Yes | Yes | Negligible | High |
| UUID v6 | 36 | Yes | Draft | Negligible | Medium |
| NanoID | 21 | No | Custom | Low | Low |
| ULID | 26 | Yes | Draft | Low | Low |
| Snowflake | 19 | Yes | Custom | Low | High |
| Custom | Variable | Yes | Custom | Low | Medium |

---

## Rationale for UUID v4

### Security
- **Unpredictable**: Random 122 bits cannot be guessed
- **No Information Leak**: Doesn't reveal profile, timestamp, or machine

### Simplicity
- **Standard**: Widely known and supported
- **Easy to Implement**: Single function call
- **No Coordination**: Distributed, no central service needed

### Performance
- **Fast**: Microsecond generation time
- **Efficient Storage**: 36 bytes (fixed length)
- **Efficient Comparison**: String comparison for equality

### Reliability
- **Collision-Resistant**: Negligible collision probability
- **Battle-Tested**: Used in production systems for decades
- **Portable**: Same behavior across platforms

### Ordering
- **Separate Field**: Use `timestamp` field for ordering (nanosecond precision)
- **Collision Handling**: If timestamps collide, order by `id` (UUID)
- **Sufficient**: Nanosecond precision handles high throughput

---

## Implementation

### Go Example

```go
import "github.com/google/uuid"

func GenerateMessageID() string {
    return uuid.New().String()
}

// Example: "550e8400-e29b-41d4-a716-446655440000"
```

### Python Example

```python
import uuid

def generate_message_id() -> str:
    return str(uuid.uuid4())

# Example: "550e8400-e29b-41d4-a716-446655440000"
```

### JavaScript Example

```javascript
import { v4 as uuidv4 } from 'uuid';

function generateMessageID() {
    return uuidv4();
}

// Example: "550e8400-e29b-41d4-a716-446655440000"
```

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (UUIDs work well with JSON)
- **ADR-007**: Per-conversation message ordering (uses timestamp + UUID)

---

## References

- **Specification**: `spec.md` - Field Definitions (ID field)
- **RFC 4122**: A Universally Unique IDentifier (UUID) URN Namespace
- **Decisions Document**: `decisions.md` - Question #2

---

## Revisions

- 2026-01-20: Initial decision - UUID v4 for message IDs
- 2026-01-20: Added detailed alternatives comparison
