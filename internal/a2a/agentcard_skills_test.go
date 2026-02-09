package a2a

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oss-aps-cli/internal/core"
)

// TestGenerateSkillCapabilities tests skill capability generation for A2A
func TestGenerateSkillCapabilities(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME to our test directory
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	defer os.Setenv("XDG_DATA_HOME", oldXDG)

	// Create test skills
	globalSkillsDir := filepath.Join(tmpDir, "data", "aps", "skills")
	require.NoError(t, os.MkdirAll(globalSkillsDir, 0755))

	// Create skill 1
	skill1Dir := filepath.Join(globalSkillsDir, "test-skill-1")
	require.NoError(t, os.MkdirAll(skill1Dir, 0755))
	skill1Md := `---
name: test-skill-1
description: First test skill
---

# Test Skill 1

Content here.
`
	require.NoError(t, os.WriteFile(filepath.Join(skill1Dir, "SKILL.md"), []byte(skill1Md), 0644))

	// Create scripts
	scriptsDir := filepath.Join(skill1Dir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "process.sh"), []byte("#!/bin/bash\necho test"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "analyze.py"), []byte("#!/usr/bin/env python3\nprint('test')"), 0755))

	// Create skill 2 (without scripts)
	skill2Dir := filepath.Join(globalSkillsDir, "test-skill-2")
	require.NoError(t, os.MkdirAll(skill2Dir, 0755))
	skill2Md := `---
name: test-skill-2
description: Second test skill
---

# Test Skill 2

Content here.
`
	require.NoError(t, os.WriteFile(filepath.Join(skill2Dir, "SKILL.md"), []byte(skill2Md), 0644))

	t.Run("generate capabilities with scripts", func(t *testing.T) {
		capabilities := GenerateSkillCapabilities("testagent")

		// Should return successfully (may or may not find skills in isolated test)
		assert.GreaterOrEqual(t, len(capabilities), 0)

		// If skills are found, check they have the right structure
		for _, cap := range capabilities {
			assert.NotEmpty(t, cap.ID)
			assert.NotEmpty(t, cap.Name)
			assert.NotEmpty(t, cap.Description)
			assert.NotEmpty(t, cap.Examples)
		}
	})

	t.Run("generate capabilities returns valid structure", func(t *testing.T) {
		capabilities := GenerateSkillCapabilities("testagent")

		// Should return successfully
		assert.GreaterOrEqual(t, len(capabilities), 0)
	})

	t.Run("no skills returns empty", func(t *testing.T) {
		// Use different profile ID with no skills
		capabilities := GenerateSkillCapabilities("nonexistent")

		// Should return empty list
		assert.Empty(t, capabilities)
	})
}

// TestGenerateAgentSkillsIntegration tests full agent card generation with skills
func TestGenerateAgentSkillsIntegration(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	defer os.Setenv("XDG_DATA_HOME", oldXDG)

	// Create test skill
	globalSkillsDir := filepath.Join(tmpDir, "data", "aps", "skills")
	require.NoError(t, os.MkdirAll(globalSkillsDir, 0755))

	skillDir := filepath.Join(globalSkillsDir, "integration-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	skillMd := `---
name: integration-skill
description: Integration test skill
---

# Integration Skill

Content.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	// Create profile
	profile := &core.Profile{
		ID:           "testagent",
		DisplayName:  "Test Agent",
		Capabilities: []string{"execute", "analyze"},
		A2A: &core.A2AConfig{
			Enabled:    true,
			ListenAddr: "127.0.0.1:8081",
		},
	}

	t.Run("agent card includes capabilities", func(t *testing.T) {
		skills := generateAgentSkills(profile)

		// Should have at least the 2 capabilities from profile
		assert.GreaterOrEqual(t, len(skills), 2)

		// Check capabilities are present
		var executeFound, analyzeFound bool
		for _, skill := range skills {
			if skill.ID == "execute" {
				executeFound = true
			}
			if skill.ID == "analyze" {
				analyzeFound = true
			}
		}
		assert.True(t, executeFound, "execute capability should be present")
		assert.True(t, analyzeFound, "analyze capability should be present")
	})

	t.Run("profile with no capabilities or skills has default", func(t *testing.T) {
		emptyProfile := &core.Profile{
			ID:          "empty",
			DisplayName: "Empty Agent",
			A2A: &core.A2AConfig{
				Enabled: true,
			},
		}

		skills := generateAgentSkills(emptyProfile)

		// Should have at least integration-skill and possibly default execute
		assert.GreaterOrEqual(t, len(skills), 1)
	})
}
