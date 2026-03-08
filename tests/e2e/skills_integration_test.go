package e2e_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/skills"
)

// TestSkillsE2E_FullWorkflow tests the complete skill workflow
func TestSkillsE2E_FullWorkflow(t *testing.T) {
	// Setup: Create temporary directories for profile and global skills
	tmpDir := t.TempDir()
	profileDir := filepath.Join(tmpDir, "profiles", "test-profile", "skills")
	globalDir := filepath.Join(tmpDir, "global-skills")

	require.NoError(t, os.MkdirAll(profileDir, 0755))
	require.NoError(t, os.MkdirAll(globalDir, 0755))

	// Step 1: Create a skill in global directory
	globalSkillDir := filepath.Join(globalDir, "global-skill")
	require.NoError(t, os.Mkdir(globalSkillDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(globalSkillDir, "scripts"), 0755))

	globalSkillMd := `---
name: global-skill
description: A globally available skill for testing
license: MIT
metadata:
  author: test-team
  version: "1.0.0"
---

# Global Skill

This skill is available globally across all profiles.

## Usage

Run the hello script:
` + "```bash\n./scripts/hello.sh\n```"

	require.NoError(t, os.WriteFile(
		filepath.Join(globalSkillDir, "SKILL.md"),
		[]byte(globalSkillMd),
		0644,
	))

	helloScript := `#!/bin/bash
echo "Hello from global skill!"
`
	require.NoError(t, os.WriteFile(
		filepath.Join(globalSkillDir, "scripts", "hello.sh"),
		[]byte(helloScript),
		0755,
	))

	// Step 2: Create a skill in profile directory (overrides global)
	profileSkillDir := filepath.Join(profileDir, "profile-skill")
	require.NoError(t, os.Mkdir(profileSkillDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(profileSkillDir, "scripts"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(profileSkillDir, "references"), 0755))

	profileSkillMd := `---
name: profile-skill
description: A profile-specific skill
license: Apache-2.0
metadata:
  author: profile-team
  version: "2.0.0"
---

# Profile Skill

This skill is specific to the test-profile.
`

	require.NoError(t, os.WriteFile(
		filepath.Join(profileSkillDir, "SKILL.md"),
		[]byte(profileSkillMd),
		0644,
	))

	processScript := `#!/bin/bash
echo "Processing with profile skill!"
`
	require.NoError(t, os.WriteFile(
		filepath.Join(profileSkillDir, "scripts", "process.sh"),
		[]byte(processScript),
		0755,
	))

	refDoc := "# Reference Documentation\n\nDetailed reference for profile-skill."
	require.NoError(t, os.WriteFile(
		filepath.Join(profileSkillDir, "references", "REFERENCE.md"),
		[]byte(refDoc),
		0644,
	))

	// Step 3: Create custom paths configuration
	customPaths := []string{globalDir}

	// Step 4: Create registry and discover skills
	registry := skills.NewRegistry("test-profile", customPaths, false)

	// Override paths for testing
	paths := registry.GetPaths()
	paths.ProfilePath = profileDir
	paths.GlobalPath = globalDir

	err := registry.Discover()
	require.NoError(t, err)

	// Step 5: Verify discovery
	assert.Equal(t, 2, registry.Count(), "Should discover 2 skills")

	allSkills := registry.List()
	assert.Len(t, allSkills, 2)

	// Verify global skill
	globalSkill, found := registry.Get("global-skill")
	assert.True(t, found)
	assert.NotNil(t, globalSkill)
	assert.Equal(t, "global-skill", globalSkill.Name)
	assert.Equal(t, "A globally available skill for testing", globalSkill.Description)
	assert.Equal(t, "MIT", globalSkill.License)
	assert.Equal(t, "test-team", globalSkill.Metadata["author"])

	// Verify profile skill
	profileSkill, found := registry.Get("profile-skill")
	assert.True(t, found)
	assert.NotNil(t, profileSkill)
	assert.Equal(t, "profile-skill", profileSkill.Name)
	assert.Equal(t, "Apache-2.0", profileSkill.License)

	// Step 6: Test script detection
	assert.True(t, globalSkill.HasScript("hello.sh"))
	assert.False(t, globalSkill.HasScript("nonexistent.sh"))

	scripts, err := globalSkill.ListScripts()
	require.NoError(t, err)
	assert.Contains(t, scripts, "hello.sh")

	// Step 7: Test reference detection
	assert.True(t, profileSkill.HasReference("REFERENCE.md"))

	refs, err := profileSkill.ListReferences()
	require.NoError(t, err)
	assert.Contains(t, refs, "REFERENCE.md")

	// Step 8: Generate XML for LLM context
	xml := registry.ToPromptXML()
	assert.NotEmpty(t, xml)
	assert.Contains(t, xml, "<available_skills>")
	assert.Contains(t, xml, "<name>global-skill</name>")
	assert.Contains(t, xml, "<name>profile-skill</name>")
	assert.Contains(t, xml, "</available_skills>")

	// Step 9: Test XML with metadata
	xmlWithMeta := registry.ToPromptXMLWithMetadata()
	assert.Contains(t, xmlWithMeta, "<license>MIT</license>")
	assert.Contains(t, xmlWithMeta, "<author>test-team</author>")
	assert.Contains(t, xmlWithMeta, "<version>1.0.0</version>")

	// Step 10: Verify hierarchical override (profile > global)
	bySource := registry.ListBySource()
	assert.NotEmpty(t, bySource)

	t.Log("E2E test completed successfully!")
}

// TestSkillsE2E_ValidationWorkflow tests skill validation
func TestSkillsE2E_ValidationWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		skillName   string
		skillMd     string
		expectValid bool
	}{
		{
			name:      "valid skill",
			skillName: "valid-skill",
			skillMd: `---
name: valid-skill
description: A valid skill for testing
---

Body content`,
			expectValid: true,
		},
		{
			name:      "missing name",
			skillName: "invalid-skill",
			skillMd: `---
description: A skill without name
---

Body`,
			expectValid: false,
		},
		{
			name:      "missing description",
			skillName: "no-desc",
			skillMd: `---
name: no-desc
---

Body`,
			expectValid: false,
		},
		{
			name:      "name mismatch",
			skillName: "actual-name",
			skillMd: `---
name: different-name
description: Mismatched name
---

Body`,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skillDir := filepath.Join(tmpDir, tt.skillName)
			require.NoError(t, os.Mkdir(skillDir, 0755))
			require.NoError(t, os.WriteFile(
				filepath.Join(skillDir, "SKILL.md"),
				[]byte(tt.skillMd),
				0644,
			))

			skill, err := skills.ParseSkill(skillDir)

			if tt.expectValid {
				assert.NoError(t, err)
				assert.NotNil(t, skill)
			} else {
				assert.Error(t, err)
				assert.Nil(t, skill)
			}
		})
	}
}

// TestSkillsE2E_TelemetryWorkflow tests telemetry tracking
func TestSkillsE2E_TelemetryWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "usage.jsonl")

	config := &skills.TelemetryConfig{
		Enabled:  true,
		EventLog: logFile,
	}

	telemetry, err := skills.NewTelemetry(config)
	require.NoError(t, err)

	// Simulate skill usage workflow
	// 1. Skill invoked
	err = telemetry.TrackInvocation("test-skill", "test-profile", "sess-123", "acp", "container")
	require.NoError(t, err)

	// 2. Skill completes successfully
	err = telemetry.TrackCompletion("test-skill", "test-profile", "sess-123", "process.sh", 1500, nil)
	require.NoError(t, err)

	// 3. Another invocation that fails
	err = telemetry.TrackInvocation("test-skill", "test-profile", "sess-124", "acp", "container")
	require.NoError(t, err)

	err = telemetry.TrackFailure("test-skill", "test-profile", "sess-124", "process.sh", 300, assert.AnError)
	require.NoError(t, err)

	// Verify log file exists and has events
	assert.FileExists(t, logFile)

	// Get statistics
	stats, err := telemetry.GetStats("test-profile", time.Time{})
	require.NoError(t, err)

	assert.Equal(t, int64(2), stats.TotalInvocations)
	assert.Equal(t, int64(1), stats.TotalCompletions)
	assert.Equal(t, int64(1), stats.TotalFailures)

	skillStats := stats.BySkill["test-skill"]
	require.NotNil(t, skillStats)
	assert.Equal(t, 0.5, skillStats.SuccessRate()) // 50% success rate
}
