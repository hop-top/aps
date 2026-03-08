package skills_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/skills"
)

func setupTestRegistry(t *testing.T) (*skills.Registry, string) {
	tmpDir := t.TempDir()

	// Create test skills
	testSkills := []struct {
		name        string
		description string
		license     string
		metadata    map[string]string
	}{
		{
			name:        "skill-one",
			description: "First test skill",
			license:     "MIT",
			metadata: map[string]string{
				"author":   "test-team",
				"version":  "1.0.0",
				"protocols": "acp,agent-protocol",
			},
		},
		{
			name:        "skill-two",
			description: "Second test skill",
			license:     "Apache-2.0",
			metadata: map[string]string{
				"author":   "another-team",
				"version":  "2.0.0",
				"protocols": "a2a",
			},
		},
	}

	for _, ts := range testSkills {
		skillDir := filepath.Join(tmpDir, ts.name)
		require.NoError(t, os.Mkdir(skillDir, 0755))

		metadataStr := ""
		if len(ts.metadata) > 0 {
			metadataStr = "\nmetadata:\n"
			for k, v := range ts.metadata {
				metadataStr += fmt.Sprintf("  %s: %s\n", k, v)
			}
		}

		skillMd := fmt.Sprintf(`---
name: %s
description: %s
license: %s%s---

# %s

Test skill body content.
`, ts.name, ts.description, ts.license, metadataStr, ts.name)

		require.NoError(t, os.WriteFile(
			filepath.Join(skillDir, "SKILL.md"),
			[]byte(skillMd),
			0644,
		))
	}

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	return registry, tmpDir
}

func TestNewSkillAdapter(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
	assert.NotNil(t, adapter)

	adapterTool := skills.NewSkillAdapter(registry, skills.AgentTypeTool)
	assert.NotNil(t, adapterTool)
}

func TestAdapter_ToXML_FilesystemBased(t *testing.T) {
	registry, tmpDir := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	xml := adapter.ToXML(nil)

	// Verify XML structure
	assert.Contains(t, xml, "<available_skills>")
	assert.Contains(t, xml, "</available_skills>")
	assert.Contains(t, xml, "<skill>")
	assert.Contains(t, xml, "<name>skill-one</name>")
	assert.Contains(t, xml, "<name>skill-two</name>")
	assert.Contains(t, xml, "<description>First test skill</description>")
	assert.Contains(t, xml, "<description>Second test skill</description>")

	// Verify location is included for filesystem-based
	assert.Contains(t, xml, "<location>")
	assert.Contains(t, xml, filepath.Join(tmpDir, "skill-one", "SKILL.md"))
	assert.Contains(t, xml, filepath.Join(tmpDir, "skill-two", "SKILL.md"))
}

func TestAdapter_ToXML_ToolBased(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

	xml := adapter.ToXML(nil)

	// Verify XML structure
	assert.Contains(t, xml, "<available_skills>")
	assert.Contains(t, xml, "<name>skill-one</name>")

	// Verify location is NOT included for tool-based
	assert.NotContains(t, xml, "<location>")
}

func TestAdapter_ToXML_WithFilter(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	// Filter by protocol
	filter := &skills.SkillFilter{
		Protocol: "acp",
	}

	xml := adapter.ToXML(filter)

	// Should include skill-one (has acp protocol)
	assert.Contains(t, xml, "<name>skill-one</name>")

	// Should NOT include skill-two (only has a2a protocol)
	assert.NotContains(t, xml, "<name>skill-two</name>")
}

func TestAdapter_ToJSON_FilesystemBased(t *testing.T) {
	registry, tmpDir := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	jsonStr, err := adapter.ToJSON(nil)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, result, "skills")
	assert.Contains(t, result, "count")
	assert.Equal(t, float64(2), result["count"])

	skillsList := result["skills"].([]interface{})
	assert.Len(t, skillsList, 2)

	// Verify first skill
	skill1 := skillsList[0].(map[string]interface{})
	assert.Equal(t, "skill-one", skill1["name"])
	assert.Equal(t, "First test skill", skill1["description"])
	assert.Equal(t, "MIT", skill1["license"])

	// Verify location is included for filesystem-based
	assert.Contains(t, skill1, "location")
	assert.Contains(t, skill1["location"], filepath.Join(tmpDir, "skill-one", "SKILL.md"))

	// Verify metadata
	metadata := skill1["metadata"].(map[string]interface{})
	assert.Equal(t, "test-team", metadata["author"])
	assert.Equal(t, "1.0.0", metadata["version"])
}

func TestAdapter_ToJSON_ToolBased(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

	jsonStr, err := adapter.ToJSON(nil)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	// Verify structure
	skillsList := result["skills"].([]interface{})
	skill1 := skillsList[0].(map[string]interface{})

	// Verify location is NOT included for tool-based
	assert.NotContains(t, skill1, "location")
}

func TestAdapter_ToJSON_WithFilter(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

	filter := &skills.SkillFilter{
		Protocol: "a2a",
	}

	jsonStr, err := adapter.ToJSON(filter)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	// Should only have 1 skill (skill-two with a2a protocol)
	assert.Equal(t, float64(1), result["count"])

	skillsList := result["skills"].([]interface{})
	assert.Len(t, skillsList, 1)

	skill := skillsList[0].(map[string]interface{})
	assert.Equal(t, "skill-two", skill["name"])
}

func TestAdapter_ToYAML_FilesystemBased(t *testing.T) {
	registry, tmpDir := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	yamlStr, err := adapter.ToYAML(nil)
	require.NoError(t, err)

	// Parse YAML
	var result map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &result)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, result, "skills")
	assert.Contains(t, result, "count")
	assert.Equal(t, 2, result["count"])

	skillsList := result["skills"].([]interface{})
	assert.Len(t, skillsList, 2)

	// Verify first skill
	skill1 := skillsList[0].(map[string]interface{})
	assert.Equal(t, "skill-one", skill1["name"])

	// Verify location is included for filesystem-based
	assert.Contains(t, skill1, "location")
	assert.Contains(t, skill1["location"], filepath.Join(tmpDir, "skill-one", "SKILL.md"))
}

func TestAdapter_ToYAML_ToolBased(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

	yamlStr, err := adapter.ToYAML(nil)
	require.NoError(t, err)

	// Parse YAML
	var result map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &result)
	require.NoError(t, err)

	skillsList := result["skills"].([]interface{})
	skill1 := skillsList[0].(map[string]interface{})

	// Verify location is NOT included for tool-based
	assert.NotContains(t, skill1, "location")
}

func TestAdapter_ToFormat(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	tests := []struct {
		format       skills.OutputFormat
		expectError  bool
		verifyString string
	}{
		{
			format:       skills.FormatXML,
			expectError:  false,
			verifyString: "<available_skills>",
		},
		{
			format:       skills.FormatJSON,
			expectError:  false,
			verifyString: `"skills"`,
		},
		{
			format:       skills.FormatYAML,
			expectError:  false,
			verifyString: "skills:",
		},
		{
			format:      "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			output, err := adapter.ToFormat(tt.format, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.verifyString)
			}
		})
	}
}

func TestAdapter_PlatformSpecificMethods(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	tests := []struct {
		name   string
		method func(*skills.SkillFilter) string
	}{
		{"ForClaude", adapter.ForClaude},
		{"ForCursor", adapter.ForCursor},
		{"ForVSCode", adapter.ForVSCode},
		{"ForGeminiCLI", adapter.ForGeminiCLI},
		{"ForACP", adapter.ForACP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.method(nil)

			// All filesystem-based methods should return XML with location
			assert.Contains(t, output, "<available_skills>")
			assert.Contains(t, output, "<location>")
			assert.Contains(t, output, "SKILL.md")
		})
	}
}

func TestAdapter_ForAPI(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	jsonStr, err := adapter.ForAPI(nil)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	// Verify it's tool-based (no location)
	skillsList := result["skills"].([]interface{})
	skill1 := skillsList[0].(map[string]interface{})
	assert.NotContains(t, skill1, "location")
}

func TestAdapter_ForA2A(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	jsonStr, err := adapter.ForA2A(nil)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	// Verify it's tool-based (no location)
	skillsList := result["skills"].([]interface{})
	skill1 := skillsList[0].(map[string]interface{})
	assert.NotContains(t, skill1, "location")
}

func TestAdapter_EmptyRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	// XML should be empty
	xml := adapter.ToXML(nil)
	assert.Empty(t, xml)

	// JSON should have empty skills array
	jsonStr, err := adapter.ToJSON(nil)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)
	assert.Equal(t, float64(0), result["count"])

	skillsList := result["skills"].([]interface{})
	assert.Len(t, skillsList, 0)
}

func TestAdapter_XMLEscaping(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	// Skill with special XML characters
	skillMd := `---
name: test-skill
description: Test skill with <special> & "characters"
---

Body`
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte(skillMd),
		0644,
	))

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)
	xml := adapter.ToXML(nil)

	// Verify XML escaping
	assert.Contains(t, xml, "&lt;special&gt;")
	assert.Contains(t, xml, "&amp;")
	assert.Contains(t, xml, "&quot;")
	assert.NotContains(t, xml, "<special>") // Should be escaped
}

func TestAdapter_LocationPathFormat(t *testing.T) {
	registry, tmpDir := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	xml := adapter.ToXML(nil)

	// Verify location paths are correctly formatted
	expectedPath := filepath.Join(tmpDir, "skill-one", "SKILL.md")
	assert.Contains(t, xml, expectedPath)

	// Verify path uses correct separator for OS
	if strings.Contains(xml, "\\") {
		// Windows path
		assert.Contains(t, xml, "\\")
	} else {
		// Unix path
		assert.Contains(t, xml, "/")
	}
}

func TestAdapter_FilterPropagation(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeFilesystem)

	filter := &skills.SkillFilter{
		Protocol: "acp",
	}

	// Test XML with filter
	xml := adapter.ToXML(filter)
	assert.Contains(t, xml, "skill-one")
	assert.NotContains(t, xml, "skill-two")

	// Test JSON with filter
	jsonStr, err := adapter.ToJSON(filter)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)
	assert.Equal(t, float64(1), result["count"])

	// Test YAML with filter
	yamlStr, err := adapter.ToYAML(filter)
	require.NoError(t, err)

	var yamlResult map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &yamlResult)
	require.NoError(t, err)
	assert.Equal(t, 1, yamlResult["count"])
}

func TestAdapter_MetadataInclusion(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

	jsonStr, err := adapter.ToJSON(nil)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	skillsList := result["skills"].([]interface{})
	skill1 := skillsList[0].(map[string]interface{})

	// Verify metadata is included
	assert.Contains(t, skill1, "metadata")
	metadata := skill1["metadata"].(map[string]interface{})
	assert.Contains(t, metadata, "author")
	assert.Contains(t, metadata, "version")
	assert.Contains(t, metadata, "protocols")
}

func TestAdapter_LicenseInclusion(t *testing.T) {
	registry, _ := setupTestRegistry(t)
	adapter := skills.NewSkillAdapter(registry, skills.AgentTypeTool)

	jsonStr, err := adapter.ToJSON(nil)
	require.NoError(t, err)

	// Parse JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err)

	skillsList := result["skills"].([]interface{})

	// Check first skill
	skill1 := skillsList[0].(map[string]interface{})
	assert.Equal(t, "MIT", skill1["license"])

	// Check second skill
	skill2 := skillsList[1].(map[string]interface{})
	assert.Equal(t, "Apache-2.0", skill2["license"])
}
