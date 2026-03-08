package skills_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/skills"
)

func TestParseSkill_Valid(t *testing.T) {
	// Create temporary skill directory
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	// Write valid SKILL.md
	skillMd := `---
name: test-skill
description: A test skill for unit testing
license: MIT
metadata:
  author: test-author
  version: "1.0.0"
---

# Test Skill

This is the body content of the skill.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	// Parse skill
	skill, err := skills.ParseSkill(skillDir)
	require.NoError(t, err)
	assert.NotNil(t, skill)

	// Verify frontmatter
	assert.Equal(t, "test-skill", skill.Name)
	assert.Equal(t, "A test skill for unit testing", skill.Description)
	assert.Equal(t, "MIT", skill.License)
	assert.Equal(t, "test-author", skill.Metadata["author"])
	assert.Equal(t, "1.0.0", skill.Metadata["version"])

	// Verify body content
	assert.Contains(t, skill.BodyContent, "This is the body content")
	assert.Equal(t, skillDir, skill.BasePath)
}

func TestParseSkill_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "no-skill-md")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	// Try to parse non-existent SKILL.md
	skill, err := skills.ParseSkill(skillDir)
	assert.Error(t, err)
	assert.Nil(t, skill)
	assert.ErrorIs(t, err, skills.ErrMissingSkillFile)
}

func TestParseSkill_MissingFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "bad-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	// Write SKILL.md without frontmatter
	skillMd := "# Just a heading, no frontmatter"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	skill, err := skills.ParseSkill(skillDir)
	assert.Error(t, err)
	assert.Nil(t, skill)
	assert.ErrorIs(t, err, skills.ErrMissingFrontmatter)
}

func TestParseSkill_InvalidName(t *testing.T) {
	tests := []struct {
		name        string
		skillName   string
		expectError bool
	}{
		{"valid lowercase", "my-skill", false},
		{"valid with numbers", "skill-123", false},
		{"invalid uppercase", "My-Skill", true},
		{"invalid leading hyphen", "-my-skill", true},
		{"invalid trailing hyphen", "my-skill-", true},
		{"invalid consecutive hyphens", "my--skill", true},
		{"invalid underscore", "my_skill", true},
		{"empty name", "", true},
		{"too long", "this-is-a-very-long-skill-name-that-exceeds-the-maximum-allowed-length-of-64-characters", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			// Use skillName as directory name to avoid name mismatch
			dirName := tt.skillName
			if dirName == "" {
				dirName = "empty-skill"
			}
			skillDir := filepath.Join(tmpDir, dirName)
			require.NoError(t, os.Mkdir(skillDir, 0755))

			skillMd := "---\nname: " + tt.skillName + "\ndescription: Test skill\n---\n\nBody"
			require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

			skill, err := skills.ParseSkill(skillDir)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, skill)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, skill)
			}
		})
	}
}

func TestParseSkill_NameMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "actual-dir-name")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	// Skill name in frontmatter doesn't match directory name
	skillMd := `---
name: different-name
description: Test skill
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	skill, err := skills.ParseSkill(skillDir)
	assert.Error(t, err)
	assert.Nil(t, skill)
	assert.ErrorIs(t, err, skills.ErrNameMismatch)
}

func TestParseSkill_MissingDescription(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "no-desc")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	skillMd := `---
name: no-desc
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	skill, err := skills.ParseSkill(skillDir)
	assert.Error(t, err)
	assert.Nil(t, skill)
	assert.ErrorIs(t, err, skills.ErrInvalidDescription)
}

func TestSkill_HasScript(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755))

	skillMd := `---
name: test-skill
description: Test skill
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "scripts", "test.sh"), []byte("#!/bin/bash"), 0755))

	skill, err := skills.ParseSkill(skillDir)
	require.NoError(t, err)

	assert.True(t, skill.HasScript("test.sh"))
	assert.False(t, skill.HasScript("nonexistent.sh"))
}

func TestSkill_ListScripts(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "scripts"), 0755))

	skillMd := `---
name: test-skill
description: Test skill
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "scripts", "script1.sh"), []byte("#!/bin/bash"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "scripts", "script2.py"), []byte("#!/usr/bin/env python3"), 0755))

	skill, err := skills.ParseSkill(skillDir)
	require.NoError(t, err)

	scripts, err := skill.ListScripts()
	require.NoError(t, err)
	assert.Len(t, scripts, 2)
	assert.Contains(t, scripts, "script1.sh")
	assert.Contains(t, scripts, "script2.py")
}

func TestSkill_ListReferences(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "references"), 0755))

	skillMd := `---
name: test-skill
description: Test skill
---

Body`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "references", "REFERENCE.md"), []byte("# Reference"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "references", "guide.md"), []byte("# Guide"), 0644))

	skill, err := skills.ParseSkill(skillDir)
	require.NoError(t, err)

	refs, err := skill.ListReferences()
	require.NoError(t, err)
	assert.Len(t, refs, 2)
	assert.Contains(t, refs, "REFERENCE.md")
	assert.Contains(t, refs, "guide.md")
}
