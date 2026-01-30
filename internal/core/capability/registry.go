package capability

import (
	"fmt"
	"strings"
)

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
