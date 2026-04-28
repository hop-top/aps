# ADR-005: Per-Message Optional Compression

**Status**: Accepted
**Date**: 2026-01-20

---

## Context

A2A protocol messages may contain large payloads (e.g., logs, documents). Compression can reduce message size and improve performance.

**Requirements**:
- Support compression without breaking compatibility
- Allow senders to choose when to compress
- Allow recipients to handle compressed messages
- Support multiple compression algorithms
- Maintain transparency for uncompressed messages
- Provide flexibility for different use cases

---

## Decision

**Implement per-message optional compression via metadata header**

### Implementation Details

#### Compression Metadata

```json
{
  "id": "msg-001",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "log_data",
  "metadata": {
    "compression": "gzip",
    "uncompressed_size": 524288  // Bytes before compression
  },
  "payload": "H4sIAAAAAAAA//8z+3KMTIyND...",  // Base64-encoded compressed data
  "timestamp": 1705773600000000000
}
```

#### Metadata Fields

| Field | Type | Required | Description |
|-------|------|-----------|-------------|
| `metadata.compression` | string | No | Compression algorithm (gzip, deflate, identity) |
| `metadata.uncompressed_size` | integer | No | Size in bytes before compression (required if compressed) |

#### Supported Algorithms

- **gzip**: RFC 1952 GZIP compression (default, widely supported)
- **deflate**: RFC 1951 DEFLATE compression (faster, similar ratio)
- **identity**: No compression (explicit, same as omitting field)

#### Uncompressed Message

```json
{
  "metadata": {},  // No compression field
  "payload": {"data": "uncompressed payload"}
}
```

#### Compression Process

```go
func CompressPayload(payload []byte, algorithm string) (string, int, error) {
    // 1. Measure uncompressed size
    uncompressedSize := len(payload)

    // 2. Compress based on algorithm
    var compressed []byte
    var err error

    switch algorithm {
    case "gzip":
        compressed, err = compressGzip(payload)
    case "deflate":
        compressed, err = compressDeflate(payload)
    case "identity":
        compressed = payload
    default:
        return "", 0, fmt.Errorf("unsupported algorithm: %s", algorithm)
    }

    if err != nil {
        return "", 0, err
    }

    // 3. Encode as base64
    base64Encoded := base64.StdEncoding.EncodeToString(compressed)

    return base64Encoded, uncompressedSize, nil
}
```

#### Decompression Process

```go
func DecompressPayload(payload string, metadata Metadata) ([]byte, error) {
    // 1. Check if compressed
    if metadata.Compression == "" || metadata.Compression == "identity" {
        return []byte(payload), nil  // Already uncompressed
    }

    // 2. Decode base64
    compressed, err := base64.StdEncoding.DecodeString(payload)
    if err != nil {
        return nil, fmt.Errorf("base64 decode failed: %w", err)
    }

    // 3. Decompress based on algorithm
    var uncompressed []byte

    switch metadata.Compression {
    case "gzip":
        uncompressed, err = decompressGzip(compressed)
    case "deflate":
        uncompressed, err = decompressDeflate(compressed)
    default:
        return nil, fmt.Errorf("unsupported algorithm: %s", metadata.Compression)
    }

    if err != nil {
        return nil, fmt.Errorf("decompression failed: %w", err)
    }

    // 4. Verify size
    if metadata.UncompressedSize > 0 && len(uncompressed) != metadata.UncompressedSize {
        return nil, fmt.Errorf("size mismatch: expected %d, got %d",
            metadata.UncompressedSize, len(uncompressed))
    }

    return uncompressed, nil}
```

---

## Consequences

### Positive

- **Flexibility**: Senders choose compression per message
- **Optional**: Uncompressed messages work without overhead
- **Transparent**: Recipients handle automatically based on metadata
- **Multiple Algorithms**: Support for gzip and deflate
- **Size Reduction**: 3-10x size reduction for text data
- **Standard**: gzip and deflate are widely available
- **Backward Compatible**: Uncompressed messages unaffected

### Negative

- **Metadata Overhead**: Adds fields for compression info
- **Compression Overhead**: CPU overhead for compressing/decompressing
- **Base64 Expansion**: Compressed data base64 encoded (33% expansion)
- **Small Message Overhead**: Small messages may become larger after compression+base64
- **Complexity**: Compression/decompression logic adds code complexity
- **Error Handling**: Must handle compression/decompression errors

---

## Alternatives Considered

### 1. Built-in Compression for All Messages ❌

**Approach**: Always compress all messages

**Implementation**:
```json
{
  "payload": "H4sIAAAAAAAA//8z+3KMTIyND..."  // Always compressed
  // No metadata compression field
}
```

**Pros**:
- **Transparent**: No user decision needed
- **Consistent**: All messages handled the same way
- **Simple**: No metadata fields, always compressed

**Cons**:
- **Small Messages Overhead**: Small messages become larger
  ```
  Original: {"msg":"hi"} → 13 bytes
  Compressed: ~30 bytes (gzip + base64)
  ```
- **Unnecessary**: Binary data doesn't compress well
- **CPU Overhead**: Always compress/decompress (even when wasteful)
- **No Choice**: Can't disable for small messages
- **Base64 Overhead**: Always adds 33% overhead

**Example Overhead**:
```
Message: {"command":"ping"}
  Size: 23 bytes
  Compressed: ~50 bytes (2x larger!)
  Result: Compression is harmful
```

**Rejection**: Small messages become larger, wasteful overhead

---

### 2. Content-Aware Auto-Compression ❌

**Approach**: System automatically compresses based on content type

**Implementation**:
```go
func AutoCompress(msg *Message) {
    contentType := detectContentType(msg.Payload)

    switch contentType {
    case "text/plain", "application/json", "text/xml":
        msg.Metadata.Compression = "gzip"  // Compress text
    case "application/octet-stream", "image/png", "video/mp4":
        // Don't compress binary (already compressed)
    default:
        msg.Metadata.Compression = ""  // Don't compress unknown
    }

    return msg}
```

**Pros**:
- **Automatic**: No user decision
- **Smart**: Avoids compressing already-compressed data
- **Optimal**: Good compression for text, none for binary

**Cons**:
- **Complex**: Requires content type detection
- **Heuristic**: May misclassify content
- **Unpredictable**: Users can't control behavior
- **False Positives**: May compress binary (harmful)
- **False Negatives**: May not compress compressible text

**Example Failure**:
```
Payload: {"data":"large text log..."}
  Detected as: "text/plain"
  Compressed: GOOD

Payload: {"data":"<base64 encoded zip>"}
  Detected as: "text/plain" (WRONG)
  Compressed: BAD (double compression, larger)
```

**Rejection**: Too complex, unpredictable behavior

---

### 3. Threshold-Based Auto-Compression ❌

**Approach**: Compress if payload size > threshold

**Implementation**:
```go
const COMPRESSION_THRESHOLD = 100 * 1024  // 100 KB

func AutoCompressBySize(msg *Message) {
    payloadSize := len(msg.Payload)

    if payloadSize > COMPRESSION_THRESHOLD {
        msg.Metadata.Compression = "gzip"
        msg.Payload = compress(msg.Payload)
    }

    return msg}
```

**Pros**:
- **Automatic**: No user decision
- **Smart**: Avoids compression overhead on small messages
- **Simple**: Easy to implement
- **Predictable**: Clear threshold

**Cons**:
- **One Size Fits All**: Doesn't consider content type
- **Binary Overhead**: May compress already-compressed binary
- **Fixed Threshold**: May not fit all use cases
- **No Choice**: Can't disable for specific cases

**Example Failure**:
```
Payload: 150 KB PNG image (already compressed)
  Size: 153600 bytes
  Threshold: >100 KB → Compress
  Compressed: 180000 bytes (larger!)
```

**Rejection**: Doesn't consider content type, may be harmful

---

### 4. Client-Side Compression Only ❌

**Approach**: Client always compresses, server always decompresses

**Implementation**:
```go
// Client
func SendMessage(msg Message) error {
    compressed := compress(msg.Payload)
    msg.Payload = base64Encode(compressed)
    return transport.Send(msg)}

// Server
func ReceiveMessage(msg Message) error {
    decompressed := decompress(base64Decode(msg.Payload))
    msg.Payload = decompressed
    return handler.Process(msg)}
```

**Pros**:
- **Transparent**: Server doesn't need metadata
- **Consistent**: All messages compressed
- **Simple**: No conditional logic

**Cons**:
- **No Choice**: Can't disable compression
- **Small Message Overhead**: Small messages become larger
- **Binary Overhead**: May compress already-compressed data
- **No Negotiation**: Can't agree on algorithm

**Rejection**: Same issues as built-in compression, no choice

---

### 5. Compression Negotiation ❌

**Approach**: Client and server negotiate compression support

**Implementation**:
```go
// Client request
type Negotiation struct {
    SupportedAlgorithms []string  // ["gzip", "deflate", "identity"]
}

// Server response
type NegotiationResponse struct {
    PreferredAlgorithm string  // "gzip"
}
```

**Pros**:
- **Flexible**: Can agree on best algorithm
- **Backward Compatible**: Can fallback to identity
- **Explicit**: Clear negotiation

**Cons**:
- **Complex**: Requires negotiation protocol
- **Extra Round-Trip**: Adds latency before actual message
- **Overkill**: gzip and deflate are universally supported
- **Unnecessary**: Metadata approach is simpler

**Rejection**: Adds complexity without significant benefit

---

### 6. No Compression ❌

**Approach**: Don't support compression at all

**Pros**:
- **Simplest**: No compression logic
- **Fast**: No compression/decompression overhead
- **Predictable**: No size variations

**Cons**:
- **Large Messages**: Large text logs/documents transmitted uncompressed
- **Bandwidth**: Higher network usage
- **Storage**: Larger message storage
- **Latency**: Slower transmission for large messages

**Example**:
```
1 MB text log:
  Uncompressed: 1 MB
  Compressed (gzip): 100 KB
  Savings: 900 KB (90% reduction)
```

**Rejection**: Significant performance and cost implications

---

## Compression Algorithm Comparison

| Algorithm | Compression Ratio | Speed | CPU Usage | Universal Support |
|-----------|------------------|--------|-----------|------------------|
| **gzip** ✅ | 3-10x | Fast | Low | Yes |
| **deflate** ✅ | 3-10x | Faster | Lower | Yes |
| **brotli** ❌ | 5-15x | Slower | Higher | Limited |
| **lz4** ❌ | 2-5x | Fastest | Lowest | Limited |
| **zstd** ❌ | 5-10x | Fast | Low | Limited |

**Rationale for gzip/deflate**:
- **Universal Support**: Available in all major languages
- **Good Ratio**: 3-10x compression
- **Fast**: Minimal CPU overhead
- **Standard**: RFC specifications

---

## When to Use Compression

### Recommended Use Cases

**Compress (> 100 KB text data)**:
- Large text logs
- JSON documents
- XML documents
- Code files
- Configuration files

**Compression Benefits**:
```
Text Log (1 MB):
  Original: 1,048,576 bytes
  Gzipped: 102,400 bytes
  Base64: 136,533 bytes
  Savings: 912,043 bytes (87% reduction)
```

### Don't Compress (< 100 KB OR binary data)**:
- Small messages (< 100 KB)
- Already-compressed files (zip, gzip, png, jpeg, mp4)
- Encrypted data (already randomized, can't compress)
- Binary executables

**Compression Harmful**:
```
PNG Image (50 KB):
  Original: 51,200 bytes
  Gzipped: 51,500 bytes (larger!)
  Base64: 68,533 bytes
  Result: 17,333 bytes larger (34% overhead)
```

---

## Threshold Recommendations

**Message Size Threshold**: 100 KB

**Reasoning**:
- Small messages (< 100 KB): Compression overhead > savings
- Medium messages (100 KB - 1 MB): Good compression ratio
- Large messages (> 1 MB): Excellent compression ratio

**Implementation**:
```go
const COMPRESSION_THRESHOLD = 100 * 1024  // 100 KB

func ShouldCompress(payload []byte) bool {
    return len(payload) > COMPRESSION_THRESHOLD}
```

---

## Base64 Overhead Consideration

**Base64 Encoding**: 33% size increase

```
Compressed: 102,400 bytes (100 KB)
Base64 Encoded: 136,533 bytes (133 KB)
Overhead: 34,133 bytes (33%)
```

**Net Compression**:
```
Original: 1,048,576 bytes (1 MB)
Compressed: 102,400 bytes (100 KB)
Base64: 136,533 bytes (133 KB)
Net Savings: 912,043 bytes (87% reduction)
```

**Conclusion**: Even with base64 overhead, compression still provides significant savings for large messages.

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (compresses well)
- **ADR-005**: Message references for large payloads (alternative approach)

---

## References

- **Specification**: `spec.md` - Message Size and Compression section
- **RFC 1951**: DEFLATE Compressed Data Format Specification
- **RFC 1952**: GZIP File Format Specification
- **Decisions Document**: `decisions.md` - Question #4

---

## Revisions

- 2026-01-20: Initial decision - Per-message optional compression
- 2026-01-20: Added detailed alternatives comparison
- 2026-01-20: Added threshold recommendations
