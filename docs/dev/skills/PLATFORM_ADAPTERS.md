# Platform Adapters for Agent Skills

## ✅ Complete Cross-Platform Support

APS now provides **format adapters** for all major IDE/TDE platforms that support Agent Skills.

## 🌐 Supported Platforms

Based on research from [agentskills.io](https://agentskills.io), [VS Code docs](https://code.visualstudio.com/docs/copilot/customization/agent-skills), [Cursor docs](https://cursor.com/docs/context/skills), and [Gemini CLI](https://geminicli.com/docs/cli/skills/):

| Platform | Type | Format | Location Field |
|----------|------|--------|---------------|
| **Claude Code** | Filesystem | XML | ✅ Included |
| **Cursor** | Filesystem | XML | ✅ Included |
| **VS Code/Copilot** | Filesystem | XML | ✅ Included |
| **Gemini CLI** | Filesystem | XML | ✅ Included |
| **Zed** | Filesystem | XML | ✅ Included |
| **API Integrations** | Tool-based | JSON | ❌ Omitted |
| **A2A Protocol** | Tool-based | JSON | ❌ Omitted |
| **Agent Protocol** | Tool-based | JSON | ❌ Omitted |

---

## 🎯 Key Concepts

### Agent Types

**Filesystem-based agents:**
- Operate within a computer environment
- Skills activated via shell commands: `cat /path/to/skill/SKILL.md`
- Include `<location>` field in output
- Examples: Claude Code, Cursor, VS Code, Gemini CLI

**Tool-based agents:**
- No dedicated computer environment
- Skills triggered via tools/APIs
- Omit `<location>` field (content delivered via tools)
- Examples: HTTP APIs, A2A protocol, MCP servers

### Output Formats

1. **XML** - Recommended for Claude models, system prompts
2. **JSON** - API responses, programmatic access
3. **YAML** - Configuration files, exports

---

## 💻 Usage Examples

### Basic Adapter Usage

```go
import "github.com/IdeaCraftersLabs/oss-aps-cli/internal/skills"

// Create registry
registry := skills.NewRegistry(profileID, userPaths, true)
registry.Discover()

// Create adapter
adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

// Generate XML
xml := adapter.ToXML(nil)

// Generate JSON
json, _ := adapter.ToJSON(nil)

// Generate YAML
yaml, _ := adapter.ToYAML(nil)
```

### Platform-Specific Methods

```go
// For Claude Code
xmlClaude := adapter.ForClaude(nil)

// For Cursor
xmlCursor := adapter.ForCursor(nil)

// For VS Code
xmlVSCode := adapter.ForVSCode(nil)

// For Gemini CLI
xmlGemini := adapter.ForGeminiCLI(nil)

// For API integrations
jsonAPI, _ := adapter.ForAPI(nil)

// For A2A protocol
jsonA2A, _ := adapter.ForA2A(nil)

// For ACP protocol
xmlACP := adapter.ForACP(nil)
```

### With Filtering

```go
// Filter by platform
filter := &skills.SkillFilter{
    Protocol: "acp",
    Platform: "linux",
}

xml := adapter.ToXML(filter)
```

---

## 📤 Output Examples

### XML Format (Filesystem-based)

```xml
<available_skills>
  <skill>
    <name>pdf-processing</name>
    <description>Extract text and tables from PDF files</description>
    <location>/Users/user/.local/share/aps/skills/pdf-processing/SKILL.md</location>
  </skill>
  <skill>
    <name>data-analysis</name>
    <description>Analyze datasets with Python/pandas</description>
    <location>/Users/user/.local/share/aps/skills/data-analysis/SKILL.md</location>
  </skill>
</available_skills>
```

### JSON Format (Tool-based)

```json
{
  "skills": [
    {
      "name": "pdf-processing",
      "description": "Extract text and tables from PDF files",
      "license": "MIT",
      "metadata": {
        "author": "aps-team",
        "version": "1.0.0"
      }
    },
    {
      "name": "data-analysis",
      "description": "Analyze datasets with Python/pandas",
      "license": "Apache-2.0"
    }
  ],
  "count": 2
}
```

### YAML Format (Configuration)

```yaml
skills:
  - name: pdf-processing
    description: Extract text and tables from PDF files
    location: /Users/user/.local/share/aps/skills/pdf-processing/SKILL.md
    license: MIT
    metadata:
      author: aps-team
      version: "1.0.0"
  - name: data-analysis
    description: Analyze datasets with Python/pandas
    location: /Users/user/.local/share/aps/skills/data-analysis/SKILL.md
    license: Apache-2.0
count: 2
```

---

## 🔌 Protocol Integration

### Agent Protocol (HTTP REST)

```go
// internal/protocol/agent/skills_handler.go

func (h *SkillsHandler) GET_Skills(w http.ResponseWriter, r *http.Request) {
    registry := skills.NewRegistry(profileID, config.SkillSources, false)
    registry.Discover()

    adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

    // Return JSON for API
    json, err := adapter.ForAPI(nil)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(json))
}
```

### A2A Protocol (Agent Card)

```go
// internal/a2a/agentcard.go

func (ac *AgentCard) GetSkillCapabilities() (string, error) {
    registry := skills.NewRegistry(ac.ProfileID, config.SkillSources, false)
    registry.Discover()

    adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

    return adapter.ForA2A(nil)
}
```

### ACP Protocol (Editor Integration)

```go
// internal/acp/methods/skill.go

func (s *Session) HandleSkillList() (string, error) {
    registry := skills.NewRegistry(s.ProfileID, config.SkillSources, true)
    registry.Discover()

    adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

    // Return XML for editor (filesystem-based)
    return adapter.ForACP(nil), nil
}
```

---

## 🔍 Path Discovery

Our implementation **automatically discovers skills** from all platforms:

```go
// Already implemented in internal/skills/paths.go
func (sp *SkillPaths) DetectIDEPaths() []string {
    return []string{
        // Claude Code
        "~/.claude/skills",

        // Cursor
        "~/.cursor/skills",
        "~/Library/Application Support/Cursor/User/skills",  // macOS
        "~/.config/Cursor/User/skills",                      // Linux

        // VS Code / GitHub Copilot
        "~/.vscode/skills",
        "~/.config/Code/User/skills",                        // Linux
        "~/Library/Application Support/Code/User/skills",    // macOS
        ".github/skills",                                    // Project-level
        ".claude/skills",                                    // Legacy

        // Gemini CLI
        ".gemini/skills",                                    // Workspace
        "~/.gemini/skills",                                  // Personal

        // Zed
        "~/.config/zed/skills",

        // Windsurf
        "~/.windsurf/skills",
    }
}
```

---

## 🧪 CLI Commands

```bash
# List skills in XML format (for Claude)
aps skill list --format xml

# List skills in JSON format (for APIs)
aps skill list --format json

# List skills in YAML format (for export)
aps skill list --format yaml

# Platform-specific
aps skill list --for claude
aps skill list --for cursor
aps skill list --for vscode
aps skill list --for gemini

# With filtering
aps skill list --format xml --protocol acp
aps skill list --format json --platform linux
```

---

## 📊 Comparison Table

| Feature | Filesystem-based | Tool-based |
|---------|-----------------|------------|
| **Location field** | ✅ Included | ❌ Omitted |
| **Activation** | File read commands | Tool calls |
| **Context usage** | Lower (progressive disclosure) | Higher (content in tool) |
| **Platforms** | IDEs, CLIs | APIs, Protocols |
| **Format** | XML (recommended) | JSON (typical) |

---

## 🎯 Best Practices

### 1. Choose the Right Agent Type

```go
// For IDEs/editors (filesystem access)
adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

// For APIs/services (no filesystem access)
adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)
```

### 2. Use Platform-Specific Methods

```go
// Instead of:
xml := adapter.ToXML(nil)

// Use:
xml := adapter.ForClaude(nil)  // More explicit, future-proof
```

### 3. Apply Filters

```go
// Filter by protocol to only show compatible skills
filter := &skills.SkillFilter{
    Protocol: "acp",
}

xml := adapter.ToXML(filter)
```

### 4. Handle Errors

```go
json, err := adapter.ToJSON(filter)
if err != nil {
    log.Printf("Failed to generate JSON: %v", err)
    return defaultSkillsList()
}
```

---

## 📚 References

Sources:
- [VS Code Agent Skills Documentation](https://code.visualstudio.com/docs/copilot/customization/agent-skills)
- [Cursor Agent Skills Documentation](https://cursor.com/docs/context/skills)
- [Gemini CLI Skills Guide](https://geminicli.com/docs/cli/skills/)
- [Agent Skills Official Site](https://agentskills.io)
- [Agent Skills Integration Guide](https://agentskills.io/integrate-skills.md)
- [GitHub Copilot Agent Skills](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills)

---

## ✅ Summary

**YES**, APS provides complete adapter support for all major platforms:

1. ✅ **Filesystem-based agents** (Claude, Cursor, VS Code, Gemini) - XML with location
2. ✅ **Tool-based agents** (APIs, protocols) - JSON without location
3. ✅ **Multiple formats** - XML, JSON, YAML
4. ✅ **Auto-discovery** - Detects skills from all platforms
5. ✅ **Platform-specific methods** - `ForClaude()`, `ForCursor()`, etc.
6. ✅ **Filtering support** - By protocol, platform, isolation

**Implementation complete and production-ready!** 🚀
