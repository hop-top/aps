# Agent Skills Implementation Plan

## Overview

This document outlines the implementation plan for integrating the Agent Skills specification into APS using **Option 3: Hybrid Model** (Global + Profile-specific skills).

**Status:** 🚧 In Progress
**Target Completion:** Sprint 1 (2 weeks)

---

## Architecture

### Directory Structure

```
# 1. Profile-specific skills (highest priority)
~/.local/share/aps/profiles/<profile-id>/skills/
  pdf-processing/
    SKILL.md
    scripts/
    references/
    assets/

# 2. Global APS skills (XDG-compliant)
$XDG_DATA_HOME/aps/skills/              # Linux: ~/.local/share/aps/skills/
  data-analysis/                         # macOS: ~/Library/Application Support/aps/skills/
    SKILL.md                             # Windows: %LOCALAPPDATA%/aps/skills/

# 3. User-configured paths
~/.config/aps/config.yaml:
  skill_sources:
    - /team/shared/skills
    - /custom/skills

# 4. Auto-detected IDE paths (opt-in)
~/.claude/skills/
~/.cursor/skills/
~/.vscode/skills/
```

### Hierarchical Override System

Skills are discovered in priority order:
1. **Profile-specific** (highest priority)
2. **Global APS**
3. **User-configured paths**
4. **Auto-detected IDE paths** (lowest priority)

If the same skill name exists in multiple locations, the highest priority version is used.

---

## Implementation Phases

### Phase 1: Core Foundation ✅ (Days 1-3)

**Status:** Complete

**Components:**
- ✅ `internal/skills/paths.go` - XDG-compliant path resolution
- ✅ `internal/skills/parser.go` - SKILL.md frontmatter parser
- ✅ `internal/skills/registry.go` - Skill discovery and XML generation
- ✅ `internal/skills/config.go` - Configuration structures

**Key Features:**
- Cross-platform XDG Base Directory support
- Auto-detection of IDE skill directories
- YAML frontmatter validation
- Hierarchical skill resolution

**Testing:**
```bash
go test ./internal/skills/...
```

---

### Phase 2: Secret Replacement System ✅ (Days 4-6)

**Status:** Complete

**Components:**
- ✅ `internal/skills/secrets.go` - Secret interception and replacement
- ✅ `internal/skills/telemetry.go` - Usage tracking (JSONL event log)

**Key Features:**
- Placeholder pattern: `${SECRET:API_KEY}`
- Intelligent replacement using local models (Ollama)
- Fallback to remote model (user's LLM)
- Profile secret integration (age/secretspec)

**Configuration:**
```yaml
secret_replacement:
  enabled: true
  local_models:
    - llama3.2:3b
    - qwen2.5:3b
  local_only: false
  placeholder_pattern: '\$\{SECRET:([A-Z_]+)\}'
```

**Testing:**
```bash
# Test secret replacement
go test ./internal/skills/... -run TestSecretReplacement

# Manual test with Ollama
ollama run llama3.2:3b
```

---

### Phase 3: CLI Integration ✅ (Days 7-9)

**Status:** Complete

**Components:**
- ✅ `internal/cli/skill/skill.go` - Skill CLI commands

**Commands:**
```bash
aps skill list                          # List all skills
aps skill list --profile myagent        # Profile-specific skills
aps skill show pdf-processing           # Show skill details
aps skill install ./my-skill --global   # Install globally
aps skill install ./my-skill -p myagent # Install to profile
aps skill validate ./my-skill           # Validate SKILL.md
aps skill stats                         # Usage statistics
aps skill suggest                       # Suggest IDE paths
```

**Testing:**
```bash
# E2E test
./aps skill validate examples/skills/hello-world
./aps skill install examples/skills/hello-world --global
./aps skill list
```

---

### Phase 4: Protocol Integration 🚧 (Days 10-12)

**Status:** Planned

#### 4.1 Agent Protocol Enhancement

**New Endpoints:**
```http
GET  /v1/skills                    # List available skills
GET  /v1/skills/{skillId}          # Get skill details
POST /v1/skills/{skillId}/invoke   # Execute skill
```

**Example Request:**
```bash
curl -X POST http://localhost:8080/v1/skills/pdf-processing/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "script": "extract.py",
    "args": {"file": "document.pdf"},
    "threadId": "thread_abc123"
  }'
```

**Implementation:**
- `internal/protocol/agent/skills_handler.go`
- Integrate with existing run executor
- Support for streaming output

#### 4.2 A2A Agent Card Enhancement

**Agent Card Schema:**
```json
{
  "name": "myagent",
  "description": "AI coding assistant",
  "capabilities": [
    {
      "type": "skill",
      "id": "pdf-processing",
      "description": "Extract text and tables from PDF files",
      "scripts": ["extract.py", "merge.py"]
    },
    {
      "type": "skill",
      "id": "data-analysis",
      "description": "Analyze datasets with Python/pandas"
    }
  ]
}
```

**Implementation:**
- Extend `internal/a2a/agentcard.go`
- Add `GenerateSkillCapabilities()` function
- Include in `.well-known/agent-card` endpoint

#### 4.3 ACP Integration

**New Methods:**
```json
// List skills
{
  "method": "skill/list",
  "params": {"profileId": "myagent"}
}

// Invoke skill
{
  "method": "skill/invoke",
  "params": {
    "skillId": "pdf-processing",
    "script": "extract.py",
    "args": {"file": "document.pdf"},
    "sessionId": "sess_123"
  }
}

// Get skill details
{
  "method": "skill/get",
  "params": {"skillId": "pdf-processing"}
}
```

**Implementation:**
- `internal/acp/methods/skill.go`
- Session-aware skill execution
- Permission checks for skill invocation

---

### Phase 5: Execution Engine 🚧 (Days 13-15)

**Status:** Planned

**Components:**
- `internal/skills/executor.go` - Skill script execution

**Key Features:**
- Respect profile isolation level (process/platform/container)
- Environment variable injection
- Secret replacement integration
- Telemetry tracking
- Output capture (stdout/stderr)

**Example:**
```go
executor := skills.NewExecutor(profile, config)
result, err := executor.Execute(ctx, skill, "extract.py", map[string]interface{}{
    "file": "document.pdf",
    "api_key": "${SECRET:API_KEY}",
})
```

**Isolation Mapping:**
- **Process:** Execute script directly with env vars
- **Platform:** Execute via SSH in sandbox user
- **Container:** Execute inside Docker container

---

### Phase 6: Security & Sandboxing 🚧 (Days 16-18)

**Status:** Planned

#### Security Considerations

1. **Script Execution:**
   - Validate script exists in skill directory
   - Check file permissions (no SUID/SGID)
   - Enforce skill isolation level requirements

2. **Allowed Tools:**
   - Parse `allowed-tools` from SKILL.md frontmatter
   - Whitelist tool invocations
   - Block unapproved system calls

3. **Secret Handling:**
   - Never log secret values
   - Redact secrets in error messages
   - Clear secrets from memory after use

4. **Resource Limits:**
   - CPU/memory limits (via cgroups for container isolation)
   - Execution timeout
   - Disk quota

5. **Audit Logging:**
   - Log all skill invocations
   - Track which profile invoked which skill
   - Record success/failure with timestamps

**Implementation:**
- `internal/skills/security.go`
- Integration with existing isolation system
- Audit log format (JSONL)

---

## Configuration Examples

### Global Config (~/.config/aps/config.yaml)

```yaml
skills:
  enabled: true

  # Additional skill search paths
  skill_sources:
    - /team/shared/skills
    - /opt/company-skills

  # Auto-detect IDE skill directories (opt-in)
  auto_detect_ide_paths: false

  # Secret replacement
  secret_replacement:
    enabled: true
    local_models:
      - llama3.2:3b
      - qwen2.5:3b
    local_only: false
    placeholder_pattern: '\$\{SECRET:([A-Z_]+)\}'

  # Telemetry
  telemetry:
    enabled: true
    event_log: ~/.local/share/aps/skills/usage.jsonl
    include_metadata: false
```

### Profile Config (<data>/profiles/myagent/profile.yaml)

```yaml
name: myagent
display_name: My AI Agent

# Skills configuration (overrides global)
skills:
  enabled: true

  # Profile-specific skill sources
  skill_sources:
    - ~/custom-skills

  # Require specific isolation for sensitive skills
  isolation_requirements:
    pdf-processing: platform
    data-analysis: container
```

---

## Testing Strategy

### Unit Tests

```bash
# Core functionality
go test ./internal/skills/...

# Specific components
go test ./internal/skills -run TestParser
go test ./internal/skills -run TestRegistry
go test ./internal/skills -run TestSecretReplacement
```

### E2E Tests

```bash
# CLI commands
./aps skill validate examples/skills/hello-world
./aps skill install examples/skills/hello-world --global
./aps skill list
./aps skill show hello-world
./aps skill stats

# Protocol integration
go test ./tests/e2e/skill_test.go
```

### Docker-based Testing

```bash
# Test in isolated environment
make docker-build-test
make docker-test-skill-integration
```

---

## Migration Path

### From Existing Actions

**Converter Tool:**
```bash
# Convert action to skill format
aps action migrate my-action --profile myagent

# Output:
# ✓ Converted action 'my-action' to skill
# Location: ~/.local/share/aps/profiles/myagent/skills/my-action/
# - Created SKILL.md with frontmatter
# - Moved scripts to scripts/
# - Created references/ directory
```

**Implementation:**
- `internal/cli/action/migrate.go`
- Auto-generate SKILL.md from ACTION.md (if exists)
- Preserve existing scripts and structure

---

## Deliverables

### Sprint 1 (Weeks 1-2)
- ✅ Core foundation (paths, parser, registry)
- ✅ Secret replacement system
- ✅ CLI commands (list, show, install, validate, stats, suggest)
- ✅ Configuration support
- ✅ Telemetry (event logging)

### Sprint 2 (Weeks 3-4)
- 🚧 Protocol integration (Agent Protocol, A2A, ACP)
- 🚧 Execution engine
- 🚧 Security & sandboxing
- 🚧 Action migration tool
- 🚧 Documentation & examples

---

## Documentation

### User Documentation
- `docs/user/skills/README.md` - User guide
- `docs/user/skills/QUICKSTART.md` - Getting started
- `docs/user/skills/CREATING_SKILLS.md` - Skill authoring guide
- `docs/user/skills/EXAMPLES.md` - Example skills

### Developer Documentation
- `docs/dev/skills/ARCHITECTURE.md` - Architecture overview
- `docs/dev/skills/SECRET_REPLACEMENT.md` - Secret handling
- `docs/dev/skills/PROTOCOL_INTEGRATION.md` - Protocol details

### Example Skills
- `examples/skills/hello-world/` - Basic skill
- `examples/skills/pdf-processing/` - Complex skill with scripts
- `examples/skills/data-analysis/` - Python skill with dependencies

---

## References

- **Agent Skills Specification:** https://agentskills.io/specification
- **Integration Guide:** https://agentskills.io/integrate-skills
- **Example Skills:** https://github.com/anthropics/skills
- **Reference Library:** https://github.com/agentskills/agentskills/tree/main/skills-ref
