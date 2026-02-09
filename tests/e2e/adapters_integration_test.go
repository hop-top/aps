package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"oss-aps-cli/internal/skills"
)

// TestAdaptersE2E_CrossPlatformWorkflow tests adapters in a complete workflow
func TestAdaptersE2E_CrossPlatformWorkflow(t *testing.T) {
	// Setup: Create skills directory structure simulating multiple platforms
	tmpDir := t.TempDir()

	// Create Claude Code skills
	claudeDir := filepath.Join(tmpDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	createSkill(t, claudeDir, "claude-skill", "Claude-specific skill", "acp")

	// Create Cursor skills
	cursorDir := filepath.Join(tmpDir, ".cursor", "skills")
	require.NoError(t, os.MkdirAll(cursorDir, 0755))
	createSkill(t, cursorDir, "cursor-skill", "Cursor-specific skill", "acp")

	// Create VS Code skills
	vscodeDir := filepath.Join(tmpDir, ".vscode", "skills")
	require.NoError(t, os.MkdirAll(vscodeDir, 0755))
	createSkill(t, vscodeDir, "vscode-skill", "VS Code skill", "agent-protocol")

	// Create global APS skills
	globalDir := filepath.Join(tmpDir, "global-skills")
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	createSkill(t, globalDir, "universal-skill", "Works everywhere", "acp,a2a,agent-protocol")

	// Create registry with all paths
	registry := skills.NewRegistry("", []string{claudeDir, cursorDir, vscodeDir, globalDir}, false)
	require.NoError(t, registry.Discover())

	// Verify discovery
	assert.Equal(t, 4, registry.Count())

	// Test 1: Generate XML for Claude Code (filesystem-based)
	t.Run("Claude Code XML", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
		xml := adapter.ForClaude(nil)

		assert.Contains(t, xml, "<available_skills>")
		assert.Contains(t, xml, "<name>claude-skill</name>")
		assert.Contains(t, xml, "<location>")
		assert.Contains(t, xml, "SKILL.md</location>")
	})

	// Test 2: Generate XML for Cursor (filesystem-based)
	t.Run("Cursor XML", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
		xml := adapter.ForCursor(nil)

		assert.Contains(t, xml, "<available_skills>")
		assert.Contains(t, xml, "<name>cursor-skill</name>")
		assert.Contains(t, xml, "<location>")
	})

	// Test 3: Generate JSON for API (tool-based)
	t.Run("API JSON", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
		jsonStr, err := adapter.ForAPI(nil)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &result)
		require.NoError(t, err)

		assert.Equal(t, float64(4), result["count"])

		// Verify no location field for tool-based
		skillsList := result["skills"].([]interface{})
		skill := skillsList[0].(map[string]interface{})
		assert.NotContains(t, skill, "location")
	})

	// Test 4: Filter by protocol (ACP only)
	t.Run("ACP Protocol Filter", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

		filter := &skills.SkillFilter{
			Protocol: "acp",
		}

		xml := adapter.ToXML(filter)

		// Should include ACP-compatible skills
		assert.Contains(t, xml, "claude-skill")
		assert.Contains(t, xml, "cursor-skill")
		assert.Contains(t, xml, "universal-skill")

		// Should NOT include agent-protocol-only skill
		assert.NotContains(t, xml, "vscode-skill")
	})

	// Test 5: Generate YAML for export
	t.Run("YAML Export", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
		yamlStr, err := adapter.ToYAML(nil)
		require.NoError(t, err)

		var result map[string]interface{}
		err = yaml.Unmarshal([]byte(yamlStr), &result)
		require.NoError(t, err)

		assert.Equal(t, 4, result["count"])

		skillsList := result["skills"].([]interface{})
		assert.Len(t, skillsList, 4)
	})
}

// TestAdaptersE2E_ProtocolIntegration tests adapters in protocol contexts
func TestAdaptersE2E_ProtocolIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test skills with different protocol support
	testSkills := []struct {
		name      string
		desc      string
		protocols string
	}{
		{"api-skill", "HTTP API skill", "agent-protocol"},
		{"editor-skill", "Editor integration", "acp"},
		{"orchestration-skill", "Agent orchestration", "a2a"},
		{"universal-skill", "All protocols", "agent-protocol,a2a,acp"},
	}

	for _, s := range testSkills {
		createSkill(t, tmpDir, s.name, s.desc, s.protocols)
	}

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	// Test Agent Protocol endpoint simulation
	t.Run("Agent Protocol Endpoint", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

		filter := &skills.SkillFilter{
			Protocol: "agent-protocol",
		}

		jsonStr, err := adapter.ForAPI(filter)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &result)
		require.NoError(t, err)

		// Should have 2 skills (api-skill + universal-skill)
		assert.Equal(t, float64(2), result["count"])
	})

	// Test A2A Agent Card generation
	t.Run("A2A Agent Card", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

		filter := &skills.SkillFilter{
			Protocol: "a2a",
		}

		jsonStr, err := adapter.ForA2A(filter)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &result)
		require.NoError(t, err)

		// Should have 2 skills (orchestration-skill + universal-skill)
		assert.Equal(t, float64(2), result["count"])

		// Verify structure is suitable for Agent Card
		skillsList := result["skills"].([]interface{})
		skill := skillsList[0].(map[string]interface{})
		assert.Contains(t, skill, "name")
		assert.Contains(t, skill, "description")
		assert.NotContains(t, skill, "location") // Tool-based
	})

	// Test ACP session
	t.Run("ACP Session", func(t *testing.T) {
		adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

		filter := &skills.SkillFilter{
			Protocol: "acp",
		}

		xml := adapter.ForACP(filter)

		// Should have 2 skills (editor-skill + universal-skill)
		assert.Contains(t, xml, "editor-skill")
		assert.Contains(t, xml, "universal-skill")
		assert.NotContains(t, xml, "api-skill")
		assert.NotContains(t, xml, "orchestration-skill")

		// Should include location for filesystem-based
		assert.Contains(t, xml, "<location>")
	})
}

// TestAdaptersE2E_FormatConversion tests converting between formats
func TestAdaptersE2E_FormatConversion(t *testing.T) {
	tmpDir := t.TempDir()
	createSkill(t, tmpDir, "test-skill", "Test skill", "acp,a2a")

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	// Convert to all formats
	xml := adapter.ToXML(nil)
	jsonStr, err := adapter.ToJSON(nil)
	require.NoError(t, err)
	yamlStr, err := adapter.ToYAML(nil)
	require.NoError(t, err)

	// Verify all contain the skill
	assert.Contains(t, xml, "test-skill")
	assert.Contains(t, jsonStr, "test-skill")
	assert.Contains(t, yamlStr, "test-skill")

	// Verify XML format
	assert.Contains(t, xml, "<available_skills>")
	assert.Contains(t, xml, "<name>test-skill</name>")

	// Verify JSON format
	var jsonResult map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &jsonResult)
	require.NoError(t, err)
	assert.Contains(t, jsonResult, "skills")

	// Verify YAML format
	var yamlResult map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &yamlResult)
	require.NoError(t, err)
	assert.Contains(t, yamlResult, "skills")
}

// TestAdaptersE2E_RealWorldPaths tests with realistic path structures
func TestAdaptersE2E_RealWorldPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate realistic directory structure
	paths := []string{
		// Project-level
		filepath.Join(tmpDir, ".github", "skills"),
		filepath.Join(tmpDir, ".claude", "skills"),

		// User-level
		filepath.Join(tmpDir, ".config", "Code", "User", "skills"),
		filepath.Join(tmpDir, ".cursor", "skills"),
	}

	for i, path := range paths {
		require.NoError(t, os.MkdirAll(path, 0755))
		createSkill(t, path, fmt.Sprintf("skill-%d", i), fmt.Sprintf("Skill at path %d", i), "acp")
	}

	registry := skills.NewRegistry("", paths, false)
	require.NoError(t, registry.Discover())

	// Should discover all skills
	assert.Equal(t, 4, registry.Count())

	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
	xml := adapter.ToXML(nil)

	// Verify all skills are included
	for i := 0; i < 4; i++ {
		assert.Contains(t, xml, fmt.Sprintf("skill-%d", i))
	}
}

// Helper function to create a test skill
func createSkill(t *testing.T, baseDir, name, description, protocols string) {
	skillDir := filepath.Join(baseDir, name)
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillMd := fmt.Sprintf(`---
name: %s
description: %s
license: MIT
metadata:
  protocols: %s
  version: "1.0.0"
---

# %s

Test skill body content for %s.
`, name, description, protocols, name, name)

	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte(skillMd),
		0644,
	))
}
