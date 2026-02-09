package skills

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentType represents the type of agent integration
type AgentType string

const (
	AgentTypeFilesystem AgentType = "filesystem" // Filesystem-based agents (Claude Code, Cursor, etc.)
	AgentTypeTool       AgentType = "tool"       // Tool-based agents (API integrations)
)

// OutputFormat represents the serialization format
type OutputFormat string

const (
	FormatXML  OutputFormat = "xml"
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
)

// SkillAdapter adapts skill registry output for different platforms
type SkillAdapter struct {
	registry  *Registry
	agentType AgentType
}

// NewSkillAdapter creates an adapter for a specific agent type
func NewSkillAdapter(registry *Registry, agentType AgentType) *SkillAdapter {
	return &SkillAdapter{
		registry:  registry,
		agentType: agentType,
	}
}

// ToFormat generates skill list in the specified format
func (a *SkillAdapter) ToFormat(format OutputFormat, filter *SkillFilter) (string, error) {
	switch format {
	case FormatXML:
		return a.ToXML(filter), nil
	case FormatJSON:
		return a.ToJSON(filter)
	case FormatYAML:
		return a.ToYAML(filter)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// ToXML generates XML format (Claude models, system prompts)
func (a *SkillAdapter) ToXML(filter *SkillFilter) string {
	skills := a.registry.List()
	if len(skills) == 0 {
		return ""
	}

	// Apply filter
	var filtered []*Skill
	for _, skill := range skills {
		if filter == nil || filter.Matches(skill) {
			filtered = append(filtered, skill)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")

	for _, skill := range filtered {
		sb.WriteString("  <skill>\n")
		sb.WriteString(fmt.Sprintf("    <name>%s</name>\n", xmlEscape(skill.Name)))
		sb.WriteString(fmt.Sprintf("    <description>%s</description>\n", xmlEscape(skill.Description)))

		// Include location only for filesystem-based agents
		if a.agentType == AgentTypeFilesystem {
			location := filepath.Join(skill.BasePath, "SKILL.md")
			sb.WriteString(fmt.Sprintf("    <location>%s</location>\n", xmlEscape(location)))
		}

		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}

// ToJSON generates JSON format (API responses, tool-based agents)
func (a *SkillAdapter) ToJSON(filter *SkillFilter) (string, error) {
	skills := a.registry.List()

	// Apply filter
	var filtered []*Skill
	for _, skill := range skills {
		if filter == nil || filter.Matches(skill) {
			filtered = append(filtered, skill)
		}
	}

	type SkillJSON struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Location    string            `json:"location,omitempty"`
		License     string            `json:"license,omitempty"`
		Metadata    map[string]string `json:"metadata,omitempty"`
	}

	type Response struct {
		Skills []SkillJSON `json:"skills"`
		Count  int         `json:"count"`
	}

	response := Response{
		Skills: make([]SkillJSON, 0, len(filtered)),
		Count:  len(filtered),
	}

	for _, skill := range filtered {
		skillJSON := SkillJSON{
			Name:        skill.Name,
			Description: skill.Description,
			License:     skill.License,
			Metadata:    skill.Metadata,
		}

		// Include location only for filesystem-based agents
		if a.agentType == AgentTypeFilesystem {
			skillJSON.Location = filepath.Join(skill.BasePath, "SKILL.md")
		}

		response.Skills = append(response.Skills, skillJSON)
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// ToYAML generates YAML format (configuration files, exports)
func (a *SkillAdapter) ToYAML(filter *SkillFilter) (string, error) {
	skills := a.registry.List()

	// Apply filter
	var filtered []*Skill
	for _, skill := range skills {
		if filter == nil || filter.Matches(skill) {
			filtered = append(filtered, skill)
		}
	}

	type SkillYAML struct {
		Name        string            `yaml:"name"`
		Description string            `yaml:"description"`
		Location    string            `yaml:"location,omitempty"`
		License     string            `yaml:"license,omitempty"`
		Metadata    map[string]string `yaml:"metadata,omitempty"`
	}

	type Response struct {
		Skills []SkillYAML `yaml:"skills"`
		Count  int         `yaml:"count"`
	}

	response := Response{
		Skills: make([]SkillYAML, 0, len(filtered)),
		Count:  len(filtered),
	}

	for _, skill := range filtered {
		skillYAML := SkillYAML{
			Name:        skill.Name,
			Description: skill.Description,
			License:     skill.License,
			Metadata:    skill.Metadata,
		}

		// Include location only for filesystem-based agents
		if a.agentType == AgentTypeFilesystem {
			skillYAML.Location = filepath.Join(skill.BasePath, "SKILL.md")
		}

		response.Skills = append(response.Skills, skillYAML)
	}

	yamlBytes, err := yaml.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// ForClaude generates XML optimized for Claude models (filesystem-based)
func (a *SkillAdapter) ForClaude(filter *SkillFilter) string {
	adapter := NewSkillAdapter(a.registry, AgentTypeFilesystem)
	return adapter.ToXML(filter)
}

// ForCursor generates format for Cursor IDE
func (a *SkillAdapter) ForCursor(filter *SkillFilter) string {
	// Cursor uses filesystem-based approach
	adapter := NewSkillAdapter(a.registry, AgentTypeFilesystem)
	return adapter.ToXML(filter)
}

// ForVSCode generates format for VS Code / GitHub Copilot
func (a *SkillAdapter) ForVSCode(filter *SkillFilter) string {
	// VS Code uses filesystem-based approach
	adapter := NewSkillAdapter(a.registry, AgentTypeFilesystem)
	return adapter.ToXML(filter)
}

// ForGeminiCLI generates format for Gemini CLI
func (a *SkillAdapter) ForGeminiCLI(filter *SkillFilter) string {
	// Gemini CLI uses filesystem-based approach
	adapter := NewSkillAdapter(a.registry, AgentTypeFilesystem)
	return adapter.ToXML(filter)
}

// ForAPI generates JSON format for API integrations
func (a *SkillAdapter) ForAPI(filter *SkillFilter) (string, error) {
	// APIs use tool-based approach (no location field)
	adapter := NewSkillAdapter(a.registry, AgentTypeTool)
	return adapter.ToJSON(filter)
}

// ForA2A generates format for A2A protocol
func (a *SkillAdapter) ForA2A(filter *SkillFilter) (string, error) {
	// A2A may use tool-based or filesystem-based depending on transport
	// Default to tool-based for wider compatibility
	adapter := NewSkillAdapter(a.registry, AgentTypeTool)
	return adapter.ToJSON(filter)
}

// ForACP generates format for ACP protocol
func (a *SkillAdapter) ForACP(filter *SkillFilter) string {
	// ACP uses filesystem-based approach (editor integration)
	adapter := NewSkillAdapter(a.registry, AgentTypeFilesystem)
	return adapter.ToXML(filter)
}
