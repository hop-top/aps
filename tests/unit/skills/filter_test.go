package skills_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"oss-aps-cli/internal/skills"
)

func TestNewSkillFilter(t *testing.T) {
	filter := skills.NewSkillFilter()

	assert.NotNil(t, filter)
	assert.Equal(t, runtime.GOOS, filter.Platform)
	assert.False(t, filter.CompatibleOnly)
}

func TestSkillFilter_PlatformFiltering(t *testing.T) {
	tests := []struct {
		name          string
		platform      string
		compatibility string
		shouldMatch   bool
	}{
		{
			name:          "linux skill on linux",
			platform:      "linux",
			compatibility: "Requires Linux",
			shouldMatch:   true,
		},
		{
			name:          "macos skill on darwin",
			platform:      "darwin",
			compatibility: "macOS only",
			shouldMatch:   true,
		},
		{
			name:          "windows skill on windows",
			platform:      "windows",
			compatibility: "Windows required",
			shouldMatch:   true,
		},
		{
			name:          "linux skill on darwin",
			platform:      "darwin",
			compatibility: "Linux only",
			shouldMatch:   false,
		},
		{
			name:          "no compatibility specified",
			platform:      "linux",
			compatibility: "",
			shouldMatch:   true,
		},
		{
			name:          "unix skill on linux",
			platform:      "linux",
			compatibility: "Unix systems",
			shouldMatch:   true,
		},
		{
			name:          "unix skill on darwin",
			platform:      "darwin",
			compatibility: "Unix systems",
			shouldMatch:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &skills.SkillFilter{
				Platform: tt.platform,
			}

			skill := &skills.Skill{
				Name:          "test-skill",
				Description:   "Test",
				Compatibility: tt.compatibility,
			}

			matches := filter.Matches(skill)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

func TestSkillFilter_ProtocolFiltering(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		metadata    map[string]string
		shouldMatch bool
	}{
		{
			name:     "skill supports acp",
			protocol: "acp",
			metadata: map[string]string{
				"protocols": "acp,agent-protocol",
			},
			shouldMatch: true,
		},
		{
			name:     "skill doesn't support a2a",
			protocol: "a2a",
			metadata: map[string]string{
				"protocols": "acp,agent-protocol",
			},
			shouldMatch: false,
		},
		{
			name:        "no protocol metadata",
			protocol:    "acp",
			metadata:    map[string]string{},
			shouldMatch: true, // Assume compatible if not specified
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &skills.SkillFilter{
				Protocol: tt.protocol,
			}

			skill := &skills.Skill{
				Name:        "test-skill",
				Description: "Test",
				Metadata:    tt.metadata,
			}

			matches := filter.Matches(skill)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

func TestSkillFilter_IsolationLevelFiltering(t *testing.T) {
	tests := []struct {
		name           string
		isolationLevel string
		metadata       map[string]string
		shouldMatch    bool
	}{
		{
			name:           "process skill in process isolation",
			isolationLevel: "process",
			metadata: map[string]string{
				"required_isolation": "process",
			},
			shouldMatch: true,
		},
		{
			name:           "container skill in container isolation",
			isolationLevel: "container",
			metadata: map[string]string{
				"required_isolation": "container",
			},
			shouldMatch: true,
		},
		{
			name:           "container skill in process isolation",
			isolationLevel: "process",
			metadata: map[string]string{
				"required_isolation": "container",
			},
			shouldMatch: false,
		},
		{
			name:           "platform skill in container isolation",
			isolationLevel: "container",
			metadata: map[string]string{
				"required_isolation": "platform",
			},
			shouldMatch: true, // Container can run platform-required skills
		},
		{
			name:           "no isolation requirement",
			isolationLevel: "process",
			metadata:       map[string]string{},
			shouldMatch:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &skills.SkillFilter{
				IsolationLevel: tt.isolationLevel,
			}

			skill := &skills.Skill{
				Name:        "test-skill",
				Description: "Test",
				Metadata:    tt.metadata,
			}

			matches := filter.Matches(skill)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

func TestSkillFilter_CombinedFiltering(t *testing.T) {
	filter := &skills.SkillFilter{
		Platform:       "linux",
		Protocol:       "acp",
		IsolationLevel: "container",
	}

	// Skill that matches all criteria
	matchingSkill := &skills.Skill{
		Name:          "matching-skill",
		Description:   "Matches all criteria",
		Compatibility: "Linux required",
		Metadata: map[string]string{
			"protocols":          "acp,agent-protocol",
			"required_isolation": "container",
		},
	}

	assert.True(t, filter.Matches(matchingSkill))

	// Skill that fails platform check
	platformMismatch := &skills.Skill{
		Name:          "platform-mismatch",
		Description:   "Wrong platform",
		Compatibility: "Windows only",
		Metadata: map[string]string{
			"protocols":          "acp",
			"required_isolation": "container",
		},
	}

	assert.False(t, filter.Matches(platformMismatch))

	// Skill that fails protocol check
	protocolMismatch := &skills.Skill{
		Name:          "protocol-mismatch",
		Description:   "Wrong protocol",
		Compatibility: "Linux",
		Metadata: map[string]string{
			"protocols":          "a2a",
			"required_isolation": "container",
		},
	}

	assert.False(t, filter.Matches(protocolMismatch))
}

func TestRegistry_ToPromptXMLFiltered(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple skills with different compatibilities
	testSkills := []struct {
		name          string
		compatibility string
		metadata      map[string]string
	}{
		{
			name:          "linux-skill",
			compatibility: "Linux only",
			metadata: map[string]string{
				"protocols": "acp",
			},
		},
		{
			name:          "macos-skill",
			compatibility: "macOS required",
			metadata: map[string]string{
				"protocols": "agent-protocol",
			},
		},
		{
			name:          "universal-skill",
			compatibility: "",
			metadata: map[string]string{
				"protocols": "acp,a2a,agent-protocol",
			},
		},
	}

	for _, s := range testSkills {
		skillDir := filepath.Join(tmpDir, s.name)
		require.NoError(t, os.Mkdir(skillDir, 0755))

		metadataStr := ""
		if len(s.metadata) > 0 {
			metadataStr = "\nmetadata:\n"
			for k, v := range s.metadata {
				metadataStr += fmt.Sprintf("  %s: %s\n", k, v)
			}
		}

		skillMd := fmt.Sprintf(`---
name: %s
description: Test skill for %s
compatibility: %s%s
---

Body`, s.name, s.name, s.compatibility, metadataStr)

		require.NoError(t, os.WriteFile(
			filepath.Join(skillDir, "SKILL.md"),
			[]byte(skillMd),
			0644,
		))
	}

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	// Test platform filtering
	filter := &skills.SkillFilter{
		Platform: "linux",
	}
	xml := registry.ToPromptXMLFiltered(filter)

	assert.Contains(t, xml, "linux-skill")
	assert.Contains(t, xml, "universal-skill")
	assert.NotContains(t, xml, "macos-skill") // Should be filtered out

	// Test protocol filtering
	filterACP := &skills.SkillFilter{
		Protocol: "acp",
	}
	xmlACP := registry.ToPromptXMLFiltered(filterACP)

	assert.Contains(t, xmlACP, "linux-skill")
	assert.Contains(t, xmlACP, "universal-skill")
	assert.NotContains(t, xmlACP, "macos-skill") // Only supports agent-protocol
}

func TestRegistry_ToPromptXMLForPlatform(t *testing.T) {
	tmpDir := t.TempDir()

	// Create platform-specific skill
	skillDir := filepath.Join(tmpDir, "darwin-skill")
	require.NoError(t, os.Mkdir(skillDir, 0755))

	skillMd := `---
name: darwin-skill
description: macOS specific skill
compatibility: Requires macOS
---

Body`
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte(skillMd),
		0644,
	))

	registry := skills.NewRegistry("", []string{tmpDir}, false)
	require.NoError(t, registry.Discover())

	// Get XML for macOS
	xmlDarwin := registry.ToPromptXMLForPlatform("darwin")
	assert.Contains(t, xmlDarwin, "darwin-skill")

	// Get XML for Linux (should not include darwin-specific skill)
	xmlLinux := registry.ToPromptXMLForPlatform("linux")
	assert.NotContains(t, xmlLinux, "darwin-skill")
}
