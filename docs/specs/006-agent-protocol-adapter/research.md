# Research: Agent Protocol Adapter

## Agent Protocol Overview

**Source**: [LangChain Agent Protocol](https://github.com/langchain-ai/agent-protocol)

Agent Protocol is a framework-agnostic API specification for serving LLM agents in production. It standardizes how agents are deployed, executed, and managed.

### Core Concepts

| Concept | Description | APS Equivalent |
|---------|-------------|----------------|
| **Runs** | Execution instances of an agent | `RunAction()`, `RunCommand()` |
| **Threads** | Multi-turn interaction containers | `SessionRegistry` |
| **Store** | Long-term memory/storage | `Profile`, `secrets.env` |
| **Agents** | Discoverable agent definitions | `ListProfiles()`, `LoadActions()` |

### Execution Paradigms

1. **Stateless Runs** (`/runs/wait`, `/runs/stream`)
   - Ephemeral thread created, run executes, thread deleted
   - For one-shot interactions

2. **Background Runs** (`POST /runs`, `GET /runs/{id}`)
   - Run executes asynchronously
   - Client polls for status or reconnects to stream

3. **Thread-bound Runs** (`POST /threads/{id}/runs`)
   - Run executes within existing thread context
   - State persists across runs

### Key Design Decisions in AP

- **Single active run per thread**: Concurrent runs forbidden (HTTP 409)
- **Append-only history**: Thread state changes are logged, not overwritten
- **Stream modes**: `output` (final only), `steps` (intermediate), `events` (all)
- **Message primitives**: Compatible with OpenAI/Anthropic message formats

## Protocol Comparison

### Agent Protocol vs OpenAI Assistants API

| Feature | Agent Protocol | OpenAI Assistants |
|---------|---------------|-------------------|
| Threads | Yes | Yes |
| Runs | Yes | Yes |
| Streaming | SSE | SSE |
| Tools | Via agent implementation | Built-in (code interpreter, retrieval) |
| Store | Yes (generic) | No (files only) |
| Agent introspection | Yes (schemas) | Limited |
| Open spec | Yes | No (proprietary) |

### Agent Protocol vs A2A (Google)

| Feature | Agent Protocol | A2A |
|---------|---------------|-----|
| Focus | Client-to-agent | Agent-to-agent |
| Transport | HTTP REST | HTTP + SSE + Push |
| Discovery | `/agents/search` | Agent Cards |
| Execution | Runs/Threads | Tasks |
| Memory | Store API | Not specified |

**Conclusion**: A2A and AP are complementary. AP handles client-to-agent; A2A handles agent-to-agent.

## What Agent Protocol Does NOT Cover

These are APS-specific features that AP doesn't address:

| APS Feature | Why Not in AP |
|-------------|---------------|
| Profile isolation | AP is transport-layer, not execution |
| Git identity injection | Tool-specific, not agent-generic |
| Secrets management | Security model varies by implementation |
| Shell detection | Execution environment is opaque |
| Webhook authentication | Assumes trusted client |
| Filesystem sandboxing | Implementation detail |

## Implementation References

### Official Implementations

1. **Python Server Stubs** (FastAPI, Pydantic V2)
   - Auto-generated from OpenAPI spec
   - Reference implementation

2. **LangGraph.js API**
   - Open-source, in-memory storage
   - Production-ready

3. **LangGraph Platform**
   - Commercial, superset of protocol
   - Adds deployment, monitoring

### Third-Party Implementations

- [Goose (Block)](https://github.com/block/goose/issues/6282) - Considering adoption
- Various AI agent frameworks evaluating compatibility

## Streaming Implementation

### Server-Sent Events (SSE)

Agent Protocol uses SSE for streaming. Key considerations:

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: output
data: {"chunk": "Hello"}

event: output
data: {"chunk": " World"}

event: done
data: {"status": "completed", "output": "Hello World"}
```

### Go SSE Implementation Pattern

```go
func (a *Adapter) handleStreamRun(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "SSE not supported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    run, err := a.core.ExecuteRun(r.Context(), input)
    if err != nil {
        // Send error event
        return
    }

    for chunk := range run.Output {
        fmt.Fprintf(w, "event: output\ndata: %s\n\n", chunk)
        flusher.Flush()
    }

    fmt.Fprintf(w, "event: done\ndata: {\"status\": \"completed\"}\n\n")
    flusher.Flush()
}
```

## Adapter Pattern Rationale

### Why Abstract?

1. **Protocol churn**: AI agent standards are evolving rapidly
2. **Multi-client support**: Different clients may prefer different protocols
3. **Testing**: Mock core, test adapters independently
4. **Clean separation**: Protocol concerns don't leak into business logic

### Interface Design Principles

1. **Minimal surface**: Only expose what adapters need
2. **Async-first**: Streaming and cancellation are first-class
3. **Context propagation**: All methods take `context.Context`
4. **Error semantics**: Errors map cleanly to HTTP status codes

## Open Questions

1. **Store scope**: Should Store be profile-scoped or global?
   - Recommendation: Profile-scoped namespaces (`{profile_id}/{key}`)

2. **Action discovery**: How to generate JSON Schema for actions?
   - Option A: Parse action manifests
   - Option B: Require explicit schema files
   - Recommendation: Start with manifest parsing, allow overrides

3. **Session persistence**: In-memory or disk?
   - Recommendation: In-memory with optional disk persistence

4. **Authentication**: How to secure the API?
   - AP spec doesn't specify auth
   - Recommendation: Optional Bearer token, similar to webhook HMAC

## References

- [Agent Protocol GitHub](https://github.com/langchain-ai/agent-protocol)
- [Agent Protocol Announcement](https://www.blog.langchain.com/agent-protocol-interoperability-for-llm-agents/)
- [LangGraph Agent Server Docs](https://docs.langchain.com/langsmith/agent-server)
- [SSE Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
