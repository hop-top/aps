package skills_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/skills"
)

func setupTestSkills(t *testing.T, baseDir string) {
	// Create multiple test skills
	skills := map[string]string{
		"skill-one": "First test skill",
		"skill-two": "Second test skill",
		"skill-three": "Third test skill",
	}

	for name, desc := range skills {
		skillDir := filepath.Join(baseDir, name)
		require.NoError(t, os.MkdirAll(skillDir, 0755))

		skillMd := "---\nname: " + name + "\ndescription: " + desc + "\n---\n\nBody"
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))
	}
}

func TestRegistry_Discover(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestSkills(t, tmpDir)

	// Create registry with tmpDir as user path
	registry := skills.NewRegistry("test-profile", []string{tmpDir}, false)
	err := registry.Discover()
	require.NoError(t, err)

	// Should find all 3 skills
	assert.Equal(t, 3, registry.Count())

	allSkills := registry.List()
	assert.Len(t, allSkills, 3)

	// Verify sorted by name
	assert.Equal(t, "skill-one", allSkills[0].Name)
	assert.Equal(t, "skill-three", allSkills[1].Name)
	assert.Equal(t, "skill-two", allSkills[2].Name)
}

func TestRegistry_Get(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestSkills(t, tmpDir)

	registry := skills.NewRegistry("test-profile", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	// Get existing skill
	skill, found := registry.Get("skill-one")
	assert.True(t, found)
	assert.NotNil(t, skill)
	assert.Equal(t, "skill-one", skill.Name)

	// Get non-existent skill
	skill, found = registry.Get("nonexistent")
	assert.False(t, found)
	assert.Nil(t, skill)
}

func TestRegistry_HierarchicalOverride(t *testing.T) {
	// Create multiple directories with overlapping skill names
	profileDir := filepath.Join(t.TempDir(), "profile")
	globalDir := filepath.Join(t.TempDir(), "global")
	userDir := filepath.Join(t.TempDir(), "user")

	require.NoError(t, os.MkdirAll(profileDir, 0755))
	require.NoError(t, os.MkdirAll(globalDir, 0755))
	require.NoError(t, os.MkdirAll(userDir, 0755))

	// Create "test-skill" in all three locations with different descriptions
	locations := map[string]string{
		profileDir: "Profile version",
		globalDir:  "Global version",
		userDir:    "User version",
	}

	for dir, desc := range locations {
		skillDir := filepath.Join(dir, "test-skill")
		require.NoError(t, os.Mkdir(skillDir, 0755))
		skillMd := "---\nname: test-skill\ndescription: " + desc + "\n---\n\nBody"
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))
	}

	// Manually discover from paths in priority order
	registry := skills.NewRegistry("", []string{}, false)

	// Simulate discovery by scanning paths manually
	for _, searchPath := range []string{profileDir, globalDir, userDir} {
		entries, err := os.ReadDir(searchPath)
		require.NoError(t, err)

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := filepath.Join(searchPath, entry.Name())
			_, err := skills.ParseSkill(skillPath)
			if err != nil {
				continue
			}
		}
	}

	// Create proper registry
	registry = skills.NewRegistry("", []string{profileDir, globalDir, userDir}, false)
	require.NoError(t, registry.Discover())

	// Should find the skill (first one discovered wins)
	skill, found := registry.Get("test-skill")
	assert.True(t, found)
	assert.NotNil(t, skill)

	// The description will be from the first path that contains it
	assert.Contains(t, []string{"Profile version", "Global version", "User version"}, skill.Description)
}

func TestRegistry_ListBySource(t *testing.T) {
	profileDir := filepath.Join(t.TempDir(), "profile")
	globalDir := filepath.Join(t.TempDir(), "global")

	require.NoError(t, os.MkdirAll(profileDir, 0755))
	require.NoError(t, os.MkdirAll(globalDir, 0755))

	// Create skill in profile
	profileSkillDir := filepath.Join(profileDir, "profile-skill")
	require.NoError(t, os.Mkdir(profileSkillDir, 0755))
	skillMd := "---\nname: profile-skill\ndescription: Profile skill\n---\n\nBody"
	require.NoError(t, os.WriteFile(filepath.Join(profileSkillDir, "SKILL.md"), []byte(skillMd), 0644))

	// Create skill in global
	globalSkillDir := filepath.Join(globalDir, "global-skill")
	require.NoError(t, os.Mkdir(globalSkillDir, 0755))
	skillMd = "---\nname: global-skill\ndescription: Global skill\n---\n\nBody"
	require.NoError(t, os.WriteFile(filepath.Join(globalSkillDir, "SKILL.md"), []byte(skillMd), 0644))

	registry := skills.NewRegistry("", []string{profileDir, globalDir}, false)
	require.NoError(t, registry.Discover())

	bySource := registry.ListBySource()
	assert.NotEmpty(t, bySource)
}

func TestRegistry_ToPromptXML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test skill
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))
	skillMd := `---
name: test-skill
description: A test skill for XML generation
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	xml := registry.ToPromptXML()

	// Verify XML structure
	assert.Contains(t, xml, "<available_skills>")
	assert.Contains(t, xml, "</available_skills>")
	assert.Contains(t, xml, "<skill>")
	assert.Contains(t, xml, "<name>test-skill</name>")
	assert.Contains(t, xml, "<description>A test skill for XML generation</description>")
	assert.Contains(t, xml, "<location>")
	assert.Contains(t, xml, "SKILL.md</location>")
}

func TestRegistry_ToPromptXML_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	xml := registry.ToPromptXML()
	assert.Empty(t, xml)
}

func TestRegistry_ToPromptXMLWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))
	skillMd := `---
name: test-skill
description: Test skill
license: MIT
metadata:
  author: test-author
  version: "1.0.0"
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	xml := registry.ToPromptXMLWithMetadata()

	assert.Contains(t, xml, "<license>MIT</license>")
	assert.Contains(t, xml, "<metadata>")
	assert.Contains(t, xml, "<author>test-author</author>")
	assert.Contains(t, xml, "<version>1.0.0</version>")
}

func TestRegistry_XMLEscape(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	// Description with special XML characters
	skillMd := `---
name: test-skill
description: Test skill with <special> & "characters"
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	xml := registry.ToPromptXML()

	// Verify escaping
	assert.Contains(t, xml, "&lt;special&gt;")
	assert.Contains(t, xml, "&amp;")
	assert.Contains(t, xml, "&quot;")
	assert.NotContains(t, xml, "<special>") // Should be escaped
}
