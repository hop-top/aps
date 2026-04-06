# Agent Skills Implementation - Agent Documentation

> Context for AI agents working on the APS codebase

## Overview

APS implements the [Agent Skills specification](https://agentskills.io) - an open standard for extending AI agent capabilities with reusable, discoverable skills.

**Status:** ✅ Phase 1-3 Complete (Core, Secrets, CLI)
**Next:** Phase 4 (Protocol Integration)

---

## Architecture

### Core Components

```
internal/skills/
├── parser.go         # SKILL.md parsing & validation
├── registry.go       # Discovery & management
├── paths.go          # Cross-platform path resolution
├── config.go         # Configuration structures
├── secrets.go        # Secret placeholder replacement
├── telemetry.go      # Usage tracking (JSONL)
├── filter.go         # Platform/protocol/isolation filtering
└── adapters.go       # Format adapters (XML/JSON/YAML)
```

### Key Design Decisions

1. **Hierarchical Override:** Profile > Global > User > IDE-detected
2. **XDG Compliant:** Cross-platform path resolution
3. **Progressive Disclosure:** Load metadata cheaply, full content on-demand
4. **Dual Agent Types:** Filesystem-based vs tool-based
5. **Format Adapters:** XML/JSON/YAML for different platforms
6. **Local-First Secrets:** Ollama for intelligent replacement

---

## Implementation Guide

### Adding a New Feature

**Example: Adding skill version management**

```go
// 1. Update Skill struct (internal/skills/parser.go)
type Skill struct {
    Name        string
    Description string
    Version     string  // Add version field
    // ...
}

// 2. Update parser to extract version from metadata
func ParseSkill(skillPath string) (*Skill, error) {
    // ... existing parsing ...

    if version, ok := skill.Metadata["version"]; ok {
        skill.Version = version
    }

    return skill, nil
}

// 3. Add tests (tests/unit/skills/parser_test.go)
func TestParseSkill_WithVersion(t *testing.T) {
    // Test version parsing
}

// 4. Update adapters to include version in output
```

### Integrating with Protocols

**Agent Protocol (HTTP REST):**

```go
// internal/protocol/agent/skills_handler.go

func (h *SkillsHandler) GetAvailableSkills(w http.ResponseWriter, r *http.Request) {
    registry := skills.NewRegistry(profileID, config.SkillSources, false)
    registry.Discover()

    adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)
    json, _ := adapter.ForAPI(nil)

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(json))
}
```

**A2A Protocol (Agent Card):**

```go
// internal/a2a/agentcard.go

func (ac *AgentCard) AddSkillCapabilities() error {
    registry := skills.NewRegistry(ac.ProfileID, config.SkillSources, false)
    registry.Discover()

    for _, skill := range registry.List() {
        ac.Capabilities = append(ac.Capabilities, Capability{
            Type:        "skill",
            ID:          skill.Name,
            Description: skill.Description,
        })
    }

    return nil
}
```

**ACP Protocol (Editor Integration):**

```go
// internal/acp/methods/skill.go

func (s *Session) HandleSkillList(params map[string]interface{}) (string, error) {
    registry := skills.NewRegistry(s.ProfileID, config.SkillSources, true)
    registry.Discover()

    adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
    return adapter.ForACP(nil), nil
}
```

---

## Code Patterns

### Pattern 1: Creating a Registry

```go
// Basic registry
registry := skills.NewRegistry(profileID, userPaths, autoDetect)
err := registry.Discover()

// With auto-detection
registry := skills.NewRegistry("myprofile", []string{}, true)
registry.Discover()

// Custom paths only
registry := skills.NewRegistry("", []string{"/custom/path"}, false)
```

### Pattern 2: Filtering Skills

```go
// By protocol
filter := &skills.SkillFilter{
    Protocol: "acp",
}
xml := registry.ToPromptXMLFiltered(filter)

// By platform (OS)
filter := &skills.SkillFilter{
    Platform: runtime.GOOS,
    CompatibleOnly: true,
}

// Combined filtering
filter := &skills.SkillFilter{
    Protocol:       "acp",
    Platform:       "linux",
    IsolationLevel: "container",
}
```

### Pattern 3: Format Adapters

```go
adapter := skills.NewSkillAdapter(registry, agentType)

// Generate different formats
xml := adapter.ToXML(filter)
json, _ := adapter.ToJSON(filter)
yaml, _ := adapter.ToYAML(filter)

// Platform-specific
xmlClaude := adapter.ForClaude(nil)
jsonAPI, _ := adapter.ForAPI(nil)
```

### Pattern 4: Secret Replacement

```go
config := &skills.SecretReplacementConfig{
    Enabled:     true,
    LocalModels: []string{"llama3.2:3b"},
    LocalOnly:   false,
}

replacer, _ := skills.NewSecretReplacer(config, secretStore)
replaced, _ := replacer.InterceptToolCall(ctx, "tool-name", args)
```

### Pattern 5: Telemetry

```go
telemetry, _ := skills.NewTelemetry(config.Telemetry)

// Track invocation
telemetry.TrackInvocation("skill-name", "profile-id", "session-id", "acp", "container")

// Track completion
telemetry.TrackCompletion("skill-name", "profile-id", "session-id", "script.sh", 1500, metadata)

// Get stats
stats, _ := telemetry.GetStats("profile-id", time.Time{})
```

---

## Testing Strategy

### Test Hierarchy

```
tests/
├── unit/skills/          # Unit tests (fast, isolated)
│   ├── parser_test.go
│   ├── registry_test.go
│   ├── paths_test.go
│   ├── secrets_test.go
│   ├── telemetry_test.go
│   ├── filter_test.go
│   └── adapters_test.go
│
└── e2e/                  # Integration tests (realistic workflows)
    ├── skills_integration_test.go
    └── adapters_integration_test.go
```

### Writing Tests

```go
// Unit test template
func TestComponent_Feature(t *testing.T) {
    // Setup with t.TempDir()
    tmpDir := t.TempDir()

    // Create test data
    setupTestSkill(t, tmpDir, "test-skill")

    // Test
    result := Component.Method(input)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}

// E2E test template
func TestE2E_Workflow(t *testing.T) {
    // Setup realistic environment
    // Test complete workflow
    // Verify end-to-end behavior
}
```

### Running Tests

```bash
# Unit tests
go test ./internal/skills/... -v
go test ./tests/unit/skills/... -v

# E2E tests
go test ./tests/e2e/skills_integration_test.go -v

# Coverage
go test ./tests/unit/skills/... -cover
```

---

## Common Tasks

### Task 1: Add a New CLI Command

```go
// internal/cli/skill/skill.go

func newMyNewCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mynew",
        Short: "Description",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
    return cmd
}

// Add to NewSkillCmd()
func NewSkillCmd() *cobra.Command {
    // ...
    cmd.AddCommand(newMyNewCmd())
    return cmd
}
```

### Task 2: Add a New Output Format

```go
// internal/skills/adapters.go

func (a *SkillAdapter) ToCustomFormat(filter *SkillFilter) (string, error) {
    skills := a.registry.List()

    // Apply filter
    var filtered []*Skill
    for _, skill := range skills {
        if filter == nil || filter.Matches(skill) {
            filtered = append(filtered, skill)
        }
    }

    // Generate custom format
    // ...

    return output, nil
}
```

### Task 3: Add IDE Path Detection

```go
// internal/skills/paths.go

func (sp *SkillPaths) DetectIDEPaths() []string {
    candidates := []string{
        // Add new IDE
        filepath.Join(homeDir, ".new-ide", "skills"),
        // ...
    }

    // Filter existing
    var detected []string
    for _, path := range candidates {
        if info, err := os.Stat(path); err == nil && info.IsDir() {
            detected = append(detected, path)
        }
    }

    return detected
}
```

---

## Troubleshooting

### Issue: Skills Not Discovered

**Cause:** Path misconfiguration or invalid SKILL.md

**Debug:**
```go
registry := skills.NewRegistry(profileID, userPaths, true)
err := registry.Discover()
if err != nil {
    log.Printf("Discovery failed: %v", err)
}

paths := registry.GetPaths()
log.Printf("Searching paths: %v", paths.AllPaths())

log.Printf("Found %d skills", registry.Count())
```

### Issue: XML Generation Empty

**Cause:** Filter too restrictive or no skills match

**Debug:**
```go
// Check without filter
xml := registry.ToPromptXML()
log.Printf("Unfiltered XML: %s", xml)

// Check with filter
filter := &skills.SkillFilter{Protocol: "acp"}
filteredXML := registry.ToPromptXMLFiltered(filter)
log.Printf("Filtered XML: %s", filteredXML)
```

### Issue: Secret Replacement Failing

**Cause:** SecretStore not configured or Ollama unavailable

**Debug:**
```go
// Check secret store
val, err := secretStore.Get("API_KEY")
if err != nil {
    log.Printf("Secret not found: %v", err)
}

// Check Ollama availability
cmd := exec.Command("ollama", "list")
if err := cmd.Run(); err != nil {
    log.Printf("Ollama not available: %v", err)
}
```

---

## Performance Considerations

### Discovery Performance

- **Fast:** Registry caches skills after discovery
- **Lazy:** Only parses frontmatter, not full SKILL.md
- **Efficient:** Uses filepath.Walk, not recursive reads

**Optimization:**
```go
// Don't re-discover on every request
var globalRegistry *skills.Registry

func GetRegistry() *skills.Registry {
    if globalRegistry == nil {
        globalRegistry = skills.NewRegistry(...)
        globalRegistry.Discover()
    }
    return globalRegistry
}
```

### XML Generation Performance

- **Lightweight:** ~50-100 tokens per skill
- **Cached:** Generate once, reuse multiple times
- **Filtered:** Only generate for relevant skills

### Telemetry Performance

- **Async:** Write events to JSONL asynchronously
- **Low overhead:** Simple append operations
- **No network:** Local file-based tracking

---

## Future Work

### Phase 4: Protocol Integration (Next)
- [ ] Agent Protocol endpoints (`/v1/skills`)
- [ ] A2A Agent Card enhancement
- [ ] ACP method integration (`skill/list`, `skill/invoke`)

### Phase 5: Execution Engine
- [ ] Script execution with isolation
- [ ] Environment injection
- [ ] Output capture

### Phase 6: Security & Polish
- [ ] Resource limits (CPU, memory, timeout)
- [ ] `allowed-tools` whitelist
- [ ] Audit logging

---

## References

### Internal Documentation
- `docs/dev/skills/IMPLEMENTATION_PLAN.md` - Full implementation plan
- `docs/dev/skills/TESTING.md` - Testing guide
- `docs/dev/skills/PLATFORM_ADAPTERS.md` - Adapter details
- `docs/user/skills/README.md` - User documentation

### External Resources
- [Agent Skills Spec](https://agentskills.io/specification)
- [Integration Guide](https://agentskills.io/integrate-skills)
- [Example Skills](https://github.com/anthropics/skills)

### Related Code
- `internal/a2a/` - A2A protocol implementation
- `internal/acp/` - ACP protocol implementation
- `internal/protocol/agent/` - Agent Protocol implementation

---

## Quick Reference

### Import Paths
```go
import "github.com/IdeaCraftersLabs/oss-aps-cli/internal/skills"
```

### Key Types
```go
type Skill struct { Name, Description, License string; Metadata map[string]string }
type Registry struct { ... }
type SkillFilter struct { Protocol, Platform, IsolationLevel string }
type SkillAdapter struct { ... }
```

### Key Functions
```go
skills.NewRegistry(profileID, userPaths, autoDetect) *Registry
registry.Discover() error
registry.List() []*Skill
registry.ToPromptXML() string
skills.NewSkillAdapter(registry, agentType) *SkillAdapter
adapter.ForClaude(filter) string
```

---

**Last Updated:** 2026-02-08
**Status:** Phase 1-3 Complete
**Next Phase:** Protocol Integration
