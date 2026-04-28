# A2A Protocol for APS

**Status**: Adopting Official Standard
**Official A2A Protocol**: https://a2a-protocol.org
**Go SDK**: https://github.com/a2aproject/a2a-go

---

## Overview

This directory contains documentation for APS's integration with the **official A2A (Agent2Agent) Protocol**.

**Critical Decision**: APS adopts the official A2A Protocol instead of creating a custom APS-specific protocol.

**Why Official A2A?**
- Zero protocol engineering effort
- Ecosystem interoperability with any A2A-compliant agent
- Official Go SDK available and production-ready
- Enterprise-grade features (authentication, streaming, async)
- Active community support and maintenance

---

## Directory Structure

```
specs/005-a2a-protocol/
├── spec.md              # APS integration with official A2A protocol
├── plan.md              # 6-week adoption plan
├── research.md          # Research on protocols and discovery of official A2A
├── decisions.md         # Design decisions and resolutions
├── quickstart.md        # Getting started guide
├── legacy/              # Archived custom protocol documentation
│   └── custom-spec.md   # Original custom APS A2A protocol (archived)
└── adrs/               # Architecture Decision Records
    └── ...              # Legacy ADRs for custom protocol
```

---

## Key Documents

### spec.md
**Purpose**: APS integration specification with official A2A Protocol

**Contents**:
- A2A Protocol summary
- APS architecture mapping
- A2A Server/Client implementation
- Agent Card generation
- Isolation tier integration
- Storage integration
- CLI command mapping
- Security model
- Compliance requirements

**Read**: First - Understand how APS integrates with A2A

---

### plan.md
**Purpose**: 6-week implementation plan for adopting official A2A

**Contents**:
- Executive summary (critical discovery)
- Protocol comparison (custom vs. official)
- Updated architecture diagrams
- 6-phase adoption plan:
  1. Protocol Adoption (Week 1)
  2. SDK Integration (Week 2-3)
  3. Isolation Integration (Week 4)
  4. CLI Integration (Week 5)
  5. Testing & Validation (Week 6)

**Read**: Second - Understand implementation timeline and phases

---

### research.md
**Purpose**: Research on existing protocols and official A2A discovery

**Contents**:
- Critical discovery of official A2A Protocol
- Comparison with other protocols (ACL, JSON-RPC, MQTT, etc.)
- Communication patterns analysis
- Security considerations
- Transport layer research
- Storage options
- Best practices
- Recommendation to adopt official A2A

**Read**: Third - Understand why official A2A was chosen

---

### decisions.md
**Purpose**: Design decisions and resolved questions

**Contents**:
- Critical decision: Adopt official A2A Protocol
- 10 resolved technical questions:
  1. Protocol adoption
  2. Message format
  3. Conversation model
  4. Communication patterns
  5. Transport bindings
  6. Agent discovery
  7. Authentication
  8. Storage backend
  9. CLI commands
  10. Legacy data migration
- Architectural decisions
- Design principles
- Alternatives not chosen

**Read**: Fourth - Understand all design decisions

---

### quickstart.md
**Purpose**: Getting started guide for using A2A in APS

**Contents**:
- Quick setup (enable A2A on profile)
- Create first A2A task
- List and view tasks
- Agent Card discovery
- Common scenarios (task delegation, streaming, etc.)
- Profile configuration examples
- Transport selection
- Security configuration
- Debugging and monitoring

**Read**: Fifth - Learn how to use A2A in APS

---

### legacy/custom-spec.md
**Purpose**: Archived original custom APS A2A protocol

**Status**: **ARCHIVED - Not for implementation**

**Contents**:
- Custom JSON message format
- IPC/HTTP/WebSocket transports
- Conversation-based messaging
- Custom request/response and pub/sub patterns

**Read**: **Only for reference** - Do not implement

---

## Official A2A Protocol Resources

### Documentation
- **Specification**: https://a2a-protocol.org/latest/specification/
- **Documentation Home**: https://a2a-protocol.org/latest/
- **What is A2A?**: https://a2a-protocol.org/latest/topics/what-is-a2a/

### Go SDK
- **GitHub**: https://github.com/a2aproject/a2a-go
- **Go Doc Reference**: https://pkg.go.dev/github.com/a2aproject/a2a-go
- **README**: Examples and usage

### Ecosystem
- **Samples**: https://github.com/a2aproject/a2a-samples
- **Other SDKs**:
  - Python: https://github.com/a2aproject/a2a-python
  - JavaScript: https://github.com/a2aproject/a2a-js
  - Java: https://github.com/a2aproject/a2a-java
  - C#/.NET: https://github.com/a2aproject/a2a-dotnet

---

## Related Standards

### ACP (Agent Client Protocol)
**Purpose**: Editor ↔ Agent communication
**URL**: https://agentclientprotocol.com
**Relation**: A2A complements ACP (different layer)

### MCP (Model Context Protocol)
**Purpose**: Agent ↔ Tools/Data communication
**URL**: https://modelcontextprotocol.io
**Relation**: A2A uses MCP patterns for agent capabilities

### Protocol Stack
```
User ↔ Editor (ACP)
         ↓
      Agent (MCP ← Tools/Data)
         ↓
      Other Agents (A2A)
```

---

## Quick Links

| What | Where |
|------|--------|
| Adopt A2A in APS | [plan.md](plan.md) |
| Understand integration | [spec.md](spec.md) |
| Get started guide | [quickstart.md](quickstart.md) |
| Research findings | [research.md](research.md) |
| Design decisions | [decisions.md](decisions.md) |
| Official A2A spec | https://a2a-protocol.org/latest/specification/ |
| A2A Go SDK | https://github.com/a2aproject/a2a-go |

---

## Migration Path

### From Custom Protocol

**Status**: Custom protocol archived in `legacy/custom-spec.md`

**Adoption**: Official A2A Protocol (v0.3.4)

**Timeline**: 6 weeks (see `plan.md`)

**Legacy Data**:
- Remains in `legacy/` directory (read-only)
- Optional migration tool provided (user-controlled)
- New tasks use official A2A from day 1

---

## Status

**Current Phase**: Planning Complete ✅
**Next Phase**: SDK Integration (Week 2-3)
**Target Date**: Complete adoption in 6 weeks

**Ready for Implementation**: Yes

---

## Contributing

When making changes to this directory:

1. **Primary Docs**: Update spec.md, plan.md, decisions.md, research.md, quickstart.md
2. **Legacy Docs**: Do NOT modify legacy/custom-spec.md (archived)
3. **ADR Updates**: Update adrs/ only if adopting new official A2A features
4. **Alignment**: Ensure all docs reference official A2A Protocol

---

## Questions?

**Official A2A Support**:
- Documentation: https://a2a-protocol.org/latest/
- GitHub Issues: https://github.com/a2aproject/A2A/issues
- Go SDK Issues: https://github.com/a2aproject/a2a-go/issues

**APS A2A Integration**:
- See `plan.md` for implementation phases
- See `decisions.md` for design rationale
- See `quickstart.md` for usage examples

---

**Last Updated**: 2026-01-21
**A2A Protocol Version**: v0.3.4
**APS Integration Version**: v1.0
