# Architecture Decision Records (ADRs)

**Status**: Adoption Phase - Adopting Official A2A Protocol
**Last Updated**: 2026-01-21

---

## Overview

This directory contains Architecture Decision Records (ADRs) for A2A Protocol adoption in APS.

**Critical Decision**: APS adopts to **official A2A Protocol** (https://a2a-protocol.org) instead of creating a custom APS-specific A2A protocol.

---

## ADR Status

### Active ADRs (Official A2A Adoption)

**No active ADRs currently** - Official A2A Protocol documentation at https://a2a-protocol.org/latest/specification/ provides all necessary specifications.

### Archived ADRs (Custom Protocol)

The following ADRs relate to **archived custom APS A2A protocol** and are kept for historical reference only:

| ADR | Title | Status | Reference |
|------|-------|--------|-----------|
| 001 | JSON v1 Serialization | Resolved | [001-json-v1-serialization.md](001-json-v1-serialization.md) |
| 002 | Protobuf v2 Serialization | Resolved | [002-protobuf-v2-serialization.md](002-protobuf-v2-serialization.md) |
| 003 | UUID v4 Message IDs | Resolved | [003-uuid-v4-message-ids.md](003-uuid-v4-message-ids.md) |
| 004 | Transport Fallback Strategy | Resolved | [004-transport-fallback-strategy.md](004-transport-fallback-strategy.md) |
| 005 | Per-Message Compression | Resolved | [005-per-message-compression.md](005-per-message-compression.md) |
| 006 | Message References Large Payloads | Resolved | [006-message-references-large-payloads.md](006-message-references-large-payloads.md) |
| 007 | Message Expiration | Resolved | [007-message-expiration.md](007-message-expiration.md) |
| 008 | Per-Conversation Ordering | Resolved | [008-per-conversation-ordering.md](008-per-conversation-ordering.md) |
| 009 | Batch Messages v1.1 | Resolved | [009-batch-messages-v1-1.md](009-batch-messages-v1-1.md) |

**Note**: These ADRs document decisions for the **archived custom protocol**. They are **not** for implementation of official A2A Protocol.

---

## Decision: Adopt Official A2A Protocol

### Context

APS's A2A Protocol specification created a custom "A2A Protocol" that shares the same name as an existing, official standard.

### Decision

**Adopt official A2A Protocol** (https://a2a-protocol.org) instead of custom APS-specific A2A protocol.

### Rationale

- **Name Collision**: Custom protocol shares name with official standard
- **Zero Protocol Engineering**: Official protocol is battle-tested, production-ready
- **Ecosystem Interoperability**: APS agents can communicate with any A2A-compliant agent
- **Go SDK Available**: Official `github.com/a2aproject/a2a-go` SDK (v0.3.4, actively maintained)
- **Enterprise-Grade**: Built-in authentication, streaming, async operations
- **Community Support**: Open source, Apache 2.0 license, active development

### Consequences

**Positive**:
- APS gains access to A2A ecosystem and features immediately
- No protocol engineering effort required
- Official SDKs available in multiple languages
- Interoperability with A2A-compliant agents from other organizations

**Negative**:
- Existing custom protocol documentation archived (legacy)
- Learning curve for team unfamiliar with official A2A SDK
- Migration effort to update APS integration

**Neutral**:
- Legacy conversations remain read-only (no auto-migration)
- APS isolation tiers map to A2A security schemes
- CLI commands updated internally to use A2A SDK

### Alternatives Considered

| Alternative | Status | Reason Not Chosen |
|-------------|--------|------------------|
| Keep Custom A2A Protocol | Rejected | Name collision, no ecosystem benefit, maintenance burden |
| Hybrid (Custom + Official A2A) | Rejected | Increased complexity, unclear migration path |
| Fork Official A2A SDK | Rejected | Unnecessary divergence, lose ecosystem benefits |

### Related Decisions

See [decisions.md](../decisions.md) for complete list of resolved questions regarding official A2A adoption.

---

## Official A2A Protocol Structure

Official A2A Protocol documentation is organized into layers:

```
┌─────────────────────────────────────────┐
│  A2A Protocol Specification         │
└─────────────────────────────────────────┘
           │
    ┌──────┼──────┐
    │      │      │
    ▼      ▼      ▼
┌────────┐┌──────┐┌────────┐
│ Layer 1││Layer 2││Layer 3│
│  Data  ││  Ops  ││Binding │
│ Model  ││       ││        │
└────────┘└──────┘└────────┘
```

### Layer 1: Canonical Data Model
- Core data structures (Task, Message, AgentCard, etc.)
- Protocol Buffer schema (normative source)
- Transport-agnostic definitions

### Layer 2: Abstract Operations
- Core operations (SendMessage, GetTask, ListTasks, etc.)
- Task lifecycle management
- Semantics independent of binding

### Layer 3: Protocol Bindings
- JSON-RPC 2.0 binding
- gRPC binding
- HTTP+JSON/REST binding
- Method mappings and protocol-specific behavior

---

## APS Integration with Official A2A

### Architecture Mapping

| APS Component | A2A Concept | Description |
|---------------|--------------|-------------|
| Profile | Agent | Isolated execution environment |
| Profile Config | Agent Card | Discovery metadata and capabilities |
| Conversation | Task | Multi-turn communication context |
| Message | Message | Communication unit |
| Participant | Participant | Task participant profiles |

### Implementation Plan

See [plan.md](../plan.md) for complete 6-week adoption plan:

1. **Phase 1** (Week 1): Protocol Adoption
   - Archive custom protocol
   - Create adoption guide
   - Update documentation

2. **Phase 2** (Week 2-3): SDK Integration
   - Add `a2a-go` dependency
   - Implement A2A Server using `a2asrv`
   - Implement A2A Client using `a2aclient`
   - Create Agent Card generator

3. **Phase 3** (Week 4): Isolation Integration
   - Map APS isolation tiers to A2A security schemes
   - Implement IPC transport via A2A extensions
   - Configure HTTP/gRPC for network communication

4. **Phase 4** (Week 5): CLI Integration
   - Update CLI commands to use A2A SDK
   - Maintain CLI UX (users shouldn't see protocol change)
   - Add Agent Card discovery commands

5. **Phase 5** (Week 6): Testing & Validation
   - Unit and integration tests
   - Interoperability tests with external A2A agents
   - Performance benchmarks
   - Security audit

---

## Creating New ADRs

### When to Create ADRs

Create new ADRs for APS-specific A2A integration decisions:

- Custom extensions to official A2A Protocol
- APS-specific transport implementations
- Isolation tier mapping decisions
- Agent Card generation strategies
- Storage backend decisions
- Security scheme configurations

### ADR Template

Use [TEMPLATE.md](TEMPLATE.md) when creating new ADRs:

```markdown
# [Number] [Title]

**Status**: [Proposed/Accepted/Rejected/Superceded]
**Date**: [YYYY-MM-DD]
**Context**: [Background]
**Decision**: [What was decided]
**Rationale**: [Why this decision]
**Consequences**: [Impact]
**Alternatives**: [Other options considered]
```

### ADR Numbering

Continue from archived custom protocol ADRs:

- Next ADR: 010
- Format: `XXX-title.md` (kebab-separated, lowercase)

---

## References

### Official A2A Protocol
- **Specification**: https://a2a-protocol.org/latest/specification/
- **Documentation**: https://a2a-protocol.org/latest/
- **Go SDK**: https://github.com/a2aproject/a2a-go
- **GitHub**: https://github.com/a2aproject/A2A

### APS A2A Integration
- **Spec**: [spec.md](../spec.md) - APS integration with official A2A
- **Plan**: [plan.md](../plan.md) - 6-week adoption plan
- **Research**: [research.md](../research.md) - Research and discovery
- **Decisions**: [decisions.md](../decisions.md) - Design decisions
- **Quickstart**: [quickstart.md](../quickstart.md) - Getting started guide

### Legacy (Custom Protocol)
- **Custom Spec**: [legacy/custom-spec.md](../legacy/custom-spec.md) - Archived custom protocol
- **Archived ADRs**: Listed above in "Archived ADRs" section

---

## Summary

**Critical Decision**: APS adopts official A2A Protocol (https://a2a-protocol.org) instead of creating a custom protocol.

**Current Status**: Planning complete, ready for Phase 2 (SDK Integration)

**Timeline**: 6 weeks for complete adoption (see [plan.md](../plan.md))

**Ready for Implementation**: Yes

---

**Last Updated**: 2026-01-21
**A2A Protocol Version**: v0.3.4
**APS Integration Version**: v1.0
