# ADR-006: Message References for Large Payloads

**Status**: Accepted
**Date**: 2026-01-20

---

## Context

A2A protocol has a maximum message size limit (1 MB) for inline payloads. Some use cases require transferring larger files or data.

**Use Cases for Large Data**:
- Transferring log files (tens of MB)
- Deploying application packages (hundreds of MB)
- Sharing datasets (GBs)
- Transferring backups (GBs)

**Requirements**:
- Support data > 1 MB
- Maintain message size limits
- Minimize data duplication
- Support multiple storage backends
- Provide secure transfer mechanism
- Maintain protocol simplicity

---

## Decision

**Implement message references for payloads > 1 MB**

### Implementation Details

#### Inline Payload (< 1 MB)

```json
{
  "id": "msg-001",
  "payload": {"command": "deploy"}  // Inline, direct
}
```

#### Message Reference (> 1 MB)

```json
{
  "id": "msg-002",
  "version": "1.0",
  "from": "agent-a",
  "to": ["agent-b"],
  "pattern": "request",
  "type": "file_transfer",
  "payload": {
    "reference": true,           // Flag indicating reference
    "uri": "file:///shared/large-file.zip",
    "size": 52428800,           // 50 MB
    "checksum": "sha256:abc123def456...",  // SHA-256 checksum
    "metadata": {
      "filename": "large-file.zip",
      "content_type": "application/zip"
    }
  },
  "timestamp": 1705773600000000000
}
```

#### Reference Payload Schema

```go
type ReferencePayload struct {
    Reference bool              `json:"reference"`       // Required
    URI       string            `json:"uri"`              // Required
    Size      int64             `json:"size"`             // Bytes
    Checksum  string            `json:"checksum"`          // Required (format: "sha256:<hex>")
    Metadata  map[string]string `json:"metadata"`         // Optional (filename, content_type, etc.)
}
```

---

## URI Schemes

### 1. `file://` - Shared Filesystem

**Format**: `file:///absolute/path/to/file`

**Example**:
```
uri: "file:///Users/Shared/aps-user1/large-file.zip"
```

**Use Case**: Local communication, shared directory

**Implementation**:
```go
func ReadFileReference(uri string) ([]byte, error) {
    // Parse URI
    u, err := url.Parse(uri)
    if err != nil {
        return nil, err
    }

    if u.Scheme != "file" {
        return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
    }

    // Read file
    return os.ReadFile(u.Path)
}
```

**Permissions**:
- File must be readable by receiving profile
- Path must be accessible within isolation tier
- ACLs applied for platform sandbox

---

### 2. `http://` / `https://` - HTTP Download

**Format**: `http://host/path/to/file` or `https://host/path/to/file`

**Example**:
```
uri: "https://storage.example.com/large-file.zip"
uri: "http://10.0.0.1:8080/files/large-file.zip"
```

**Use Case**: Network communication, external storage

**Implementation**:
```go
func ReadHTTPReference(uri string) ([]byte, error) {
    resp, err := http.Get(uri)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    return io.ReadAll(resp.Body)
}
```

**Authentication**:
- Basic Auth: `https://user:pass@host/file`
- Headers: Negotiated via metadata
- Custom: Application-specific

---

### 3. `s3://` - S3 Object Reference

**Format**: `s3://bucket-name/object-key`

**Example**:
```
uri: "s3://my-bucket/deployments/large-file.zip"
```

**Use Case**: Cloud storage, S3-compatible services

**Implementation**:
```go
func ReadS3Reference(uri string) ([]byte, error) {
    // Parse URI
    // format: s3://bucket/key
    parts := strings.SplitN(uri, "://", 2)
    bucketKey := strings.SplitN(parts[1], "/", 2)

    bucket := bucketKey[0]
    key := bucketKey[1]

    // Use AWS SDK
    svc := s3.New(session.New())
    result, err := svc.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })

    if err != nil {
        return nil, err
    }
    defer result.Body.Close()

    return io.ReadAll(result.Body)
}
```

**Credentials**:
- Stored in `secrets.env`: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- Profile-specific configuration
- Optional (public buckets)

---

### 4. `data://` - Inline Base64 (Medium-Sized)

**Format**: `data://<base64-encoded-data>`

**Example**:
```
uri: "data://SGVsbG8gV29ybGQ="  // Base64 of "Hello World"
```

**Use Case**: Medium-sized payloads (100 KB - 1 MB)

**Implementation**:
```go
func ReadDataURL(uri string) ([]byte, error) {
    // Parse URI
    u, err := url.Parse(uri)
    if err != nil {
        return nil, err
    }

    if u.Scheme != "data" {
        return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
    }

    // Decode base64
    return base64.StdEncoding.DecodeString(u.Opaque)
}
```

**Usage**:
- For payloads too large for inline JSON but small enough for base64
- Alternative to compression
- No external storage needed

---

## Checksums

### Purpose

- **Integrity**: Verify data integrity
- **Security**: Detect tampering
- **Trust**: Ensure received data matches sent

### Format

```
checksum: "sha256:abc123def456..."
         |       |             |
         |       |             |Hex encoded
         |       |Algorithm
         |Prefix (allows multiple algorithms)
```

### Supported Algorithms

- `sha256`: SHA-256 (default, required)
- `sha512`: SHA-512 (future)
- `md5`: MD5 (legacy, not recommended)

### Implementation

```go
func VerifyChecksum(data []byte, checksum string) error {
    // Parse checksum
    parts := strings.SplitN(checksum, ":", 2)
    if len(parts) != 2 {
        return fmt.Errorf("invalid checksum format")
    }

    algorithm := parts[0]
    expected := parts[1]

    // Compute checksum
    var computed []byte
    switch algorithm {
    case "sha256":
        hash := sha256.Sum256(data)
        computed = hash[:]
    default:
        return fmt.Errorf("unsupported algorithm: %s", algorithm)
    }

    // Compare
    computedHex := hex.EncodeToString(computed)
    if computedHex != expected {
        return fmt.Errorf("checksum mismatch: expected %s, got %s",
            expected, computedHex)
    }

    return nil
}
```

---

## Consequences

### Positive

- **Unlimited Size**: Supports GBs+ of data
- **Decoupled**: Message metadata separate from large data
- **Efficient**: Large data not duplicated across messages
- **Flexible**: Multiple storage backends (filesystem, HTTP, S3)
- **Secure**: Checksums ensure integrity
- **Standard**: Uses standard URI formats

### Negative

- **Additional Complexity**: Senders must upload data first
- **Data Management**: Cleanup of referenced data (orphaned files)
- **Access Control**: Must manage permissions for shared resources
- **Network Dependency**: HTTP/S3 references require network
- **Synchronous?**: May require blocking on download
- **No Streaming**: Data must be downloaded before processing

---

## Alternatives Considered

### 1. Increase Message Size Limit ❌

**Approach**: Remove or increase 1 MB limit

**Pros**:
- **Simple**: No changes to message format
- **Transparent**: Senders don't need to change behavior

**Cons**:
- **Large Memory**: Must buffer large messages in memory
- **Performance**: Slower parsing/serialization for large messages
- **Transport Limitations**: Some transports have practical limits
  ```
  IPC: File size OK, but JSON parsing slow for 100 MB
  HTTP: May timeout on large uploads
  WebSocket: Frame size limits (may need fragmentation)
  ```
- **Storage**: Large message files on filesystem
- **Memory**: Profiles must buffer entire message
- **Unnecessary**: Most messages are small, rare to need > 1 MB

**Example Issues**:
```
100 MB message:
  JSON parsing: 5-10 seconds
  Memory: 100 MB+ buffer
  Network: May timeout (default 30s)
  Storage: 100 MB per message (disk usage)
```

**Rejection**: Performance and memory issues, overkill for rare use case

---

### 2. Chunking (Split Large Messages) ❌

**Approach**: Split large payload into multiple smaller messages

**Implementation**:
```json
// Message 1/3
{
  "id": "chunk-1",
  "type": "chunk",
  "payload": {
    "chunk_index": 1,
    "total_chunks": 3,
    "chunk_id": "file-transfer-001",
    "data": "<base64 of chunk 1>"
  }
}

// Message 2/3
{
  "id": "chunk-2",
  "type": "chunk",
  "payload": {
    "chunk_index": 2,
    "total_chunks": 3,
    "chunk_id": "file-transfer-001",
    "data": "<base64 of chunk 2>"
  }
}

// Message 3/3
{
  "id": "chunk-3",
  "type": "chunk",
  "payload": {
    "chunk_index": 3,
    "total_chunks": 3,
    "chunk_id": "file-transfer-001",
    "data": "<base64 of chunk 3>",
    "complete": true  // Last chunk
  }
}
```

**Pros**:
- **Standard Protocol**: Uses existing message format
- **Incremental**: Can stream chunks
- **No External Storage**: Self-contained in protocol

**Cons**:
- **Complex Reassembly**: Recipient must track and reassemble
- **Ordering**: Must receive chunks in order
- **Partial Failure**: Missing chunk = unusable transfer
- **Memory**: Must buffer all chunks in memory
- **Stateful**: Need to track chunk state
- **Base64 Overhead**: 33% overhead per chunk
- **Timeout Risk**: Long transfer may timeout

**Example Issues**:
```
100 MB file (256 KB chunks):
  Total chunks: 400
  Base64 overhead: 341 KB per chunk
  Total messages: 400 messages
  Reassembly: Must buffer all 400 chunks
  Memory: 136 MB+ (341 KB * 400)
  Timeout: High risk (400 messages, potential gaps)
```

**Rejection**: Too complex, stateful, high failure risk

---

### 3. Binary Protocol for Large Payloads ❌

**Approach**: Use binary protocol (separate from A2A) for large data

**Implementation**:
```
1. Send A2A message with metadata (reference to binary transfer)
2. Open binary connection (custom protocol)
3. Stream binary data
4. Close binary connection
5. Send A2A message (transfer complete)
```

**Pros**:
- **Efficient**: Binary streaming, no base64 overhead
- **Streaming**: Can stream continuously
- **Low Memory**: Don't buffer entire message

**Cons**:
- **New Protocol**: Custom binary protocol
- **Two-Step**: Requires coordination between A2A and binary
- **Complex**: Must implement binary protocol
- **Port Management**: May need separate ports
- **Limited**: Not transport-agnostic

**Rejection**: Too complex, new protocol, not transport-agnostic

---

### 4. WebSocket Binary Frames for Large Payloads ❌

**Approach**: Use WebSocket binary frames for large data

**Implementation**:
```
1. Send A2A metadata message (JSON)
2. Send binary frame(s) with large data
3. Recipient associates binary frames with metadata message
```

**Pros**:
- **Efficient**: Binary frames, no base64 overhead
- **Streaming**: Can stream continuously
- **Built-in**: WebSocket supports binary frames

**Cons**:
- **WebSocket-Only**: Doesn't work with IPC or HTTP
- **Two-Step**: Coordination between JSON and binary
- **Association Complexity**: Must correlate metadata with binary frames
- **Ordering**: Binary frames may arrive out of order

**Rejection**: WebSocket-only, doesn't work for other transports

---

### 5. No Support for Large Payloads ❌

**Approach**: Reject messages > 1 MB

**Pros**:
- **Simple**: No implementation
- **Predictable**: Clear limits
- **Safe**: Prevents resource exhaustion

**Cons**:
- **Limited**: Can't transfer large files
- **Workaround**: Users must implement external transfer mechanism
- **Fragmented**: User experience split between A2A and external tools

**Example Workflow**:
```
1. User uploads file to external storage (S3, FTP)
2. User manually generates URL/share
3. User sends URL in A2A message
4. Recipient downloads file
5. User processes file
```

**Rejection**: Poor user experience, manual workflow

---

## URI Scheme Comparison

| Scheme | Storage | Network | Complexity | Use Case |
|--------|---------|---------|------------|----------|
| **file://** ✅ | Filesystem | No | Low | Local, shared directory |
| **http://** ✅ | Server | Yes | Medium | Network, external storage |
| **https://** ✅ | Server | Yes | Medium | Network, secure storage |
| **s3://** ✅ | Cloud | Yes | Medium | Cloud, S3-compatible |
| **data://** ✅ | Inline | No | Low | Medium-sized payloads |

---

## Security Considerations

### Access Control

**Filesystem References**:
- Profile must have read access to file
- ACLs applied for platform sandbox
- Directory permissions: 0600 for profile data

**HTTP References**:
- TLS recommended for HTTPS
- Authentication via metadata or URL
- Optional: Signed URLs (time-limited)

**S3 References**:
- IAM roles or access keys
- Bucket policies
- Signed URLs (optional)

### Checksums

- **Required**: All references must include checksum
- **Verification**: Recipient must verify before processing
- **Algorithm**: SHA-256 (required), SHA-512 (optional)

---

## Related Decisions

- **ADR-001**: JSON for v1.0 serialization (max 1 MB limit)
- **ADR-005**: Per-message compression (complementary approach)

---

## References

- **Specification**: `spec.md` - Message Size and Compression section
- **RFC 3986**: Uniform Resource Identifier (URI): Generic Syntax
- **AWS S3**: https://docs.aws.amazon.com/AmazonS3/
- **Decisions Document**: `decisions.md` - Question #5

---

## Revisions

- 2026-01-20: Initial decision - Message references for large payloads
- 2026-01-20: Added detailed alternatives comparison
- 2026-01-20: Added URI scheme comparison
