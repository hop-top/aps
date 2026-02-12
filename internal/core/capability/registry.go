package capability

import (
	"fmt"
	"strings"
)

// BuiltinCapabilities defines all built-in capabilities
var BuiltinCapabilities = []BuiltinCapability{
	{Name: "a2a", Description: "Agent-to-Agent protocol"},
	{Name: "agent-protocol", Description: "Agent Protocol (IDE/stdin client)"},
	{Name: "webhooks", Description: "Webhook event server"},
}

// IsBuiltin returns true if the capability name is a builtin
func IsBuiltin(name string) bool {
	lower := strings.ToLower(name)
	for _, b := range BuiltinCapabilities {
		if strings.ToLower(b.Name) == lower {
			return true
		}
	}
	return false
}

// GetBuiltin returns a builtin capability by name
func GetBuiltin(name string) (BuiltinCapability, error) {
	lower := strings.ToLower(name)
	for _, b := range BuiltinCapabilities {
		if strings.ToLower(b.Name) == lower {
			return b, nil
		}
	}
	return BuiltinCapability{}, fmt.Errorf("unknown builtin capability: %s", name)
}

// ListBuiltins returns all builtin capabilities
func ListBuiltins() []BuiltinCapability {
	return BuiltinCapabilities
}

// Exists returns true if a capability exists (builtin or external)
func Exists(name string) bool {
	if IsBuiltin(name) {
		return true
	}
	_, err := LoadCapability(name)
	return err == nil
}

var SmartPatterns = []SmartPattern{
	{ToolName: "claude", DefaultPath: ".claude/commands/agent.md", Description: "Claude Code Agent"},
	{ToolName: "cursor", DefaultPath: ".cursor/commands/agent.md", Description: "Cursor Agent"},
	{ToolName: "roo", DefaultPath: ".roo/commands/agent.md", Description: "Roo Agent"},
	{ToolName: "augment", DefaultPath: ".augment/commands/agent.md", Description: "Augment Agent"},
	{ToolName: "opencode", DefaultPath: ".opencode/command/agent.md", Description: "OpenCode Agent"},
	{ToolName: "windsurf", DefaultPath: ".windsurf/workflows/agent.md", Description: "Windsurf Agent Workflow"},
	{ToolName: "copilot", DefaultPath: ".github/agents/agent.agent.md", Description: "GitHub Copilot Agent"},
	{ToolName: "gemini", DefaultPath: ".gemini/GEMINI.md", Description: "Gemini Agent"},
	{ToolName: "antigravity", DefaultPath: ".antigravity/ANTIGRAVITY.md", Description: "Antigravity Agent"},
	{ToolName: "crush", DefaultPath: ".crush/CRUSH.md", Description: "Crush Agent"},
}

func GetSmartPattern(toolName string) (SmartPattern, error) {
	lowerName := strings.ToLower(toolName)
	for _, p := range SmartPatterns {
		if strings.ToLower(p.ToolName) == lowerName {
			return p, nil
		}
	}
	return SmartPattern{}, fmt.Errorf("unknown tool pattern: %s", toolName)
}

func ListSmartPatterns() []SmartPattern {
	return SmartPatterns
}
