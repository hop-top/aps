package core_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"hop.top/aps/internal/core/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolRegistry_Lookup(t *testing.T) {
	t.Run("Get existing tool - claude", func(t *testing.T) {
		tool, err := tools.GetTool("claude")
		require.NoError(t, err)
		require.Equal(t, "Claude Code", tool.Name)
		require.Equal(t, "Anthropic's AI coding assistant", tool.Description)
	})

	t.Run("Get existing tool - gemini", func(t *testing.T) {
		tool, err := tools.GetTool("gemini")
		require.NoError(t, err)
		require.Equal(t, "Google Gemini CLI", tool.Name)
	})

	t.Run("Get existing tool - codex", func(t *testing.T) {
		tool, err := tools.GetTool("codex")
		require.NoError(t, err)
		require.Equal(t, "OpenAI Codex", tool.Name)
	})

	t.Run("Get non-existent tool", func(t *testing.T) {
		_, err := tools.GetTool("nonexistent")
		require.Error(t, err)
	})
}

func TestToolRegistry_List(t *testing.T) {
	toolsList := tools.ListTools()

	require.NotEmpty(t, toolsList, "Tool list should not be empty")

	foundClaude := false
	foundGemini := false
	foundCodex := false

	for _, tool := range toolsList {
		if tool.Name == "Claude Code" {
			foundClaude = true
		}
		if tool.Name == "Google Gemini CLI" {
			foundGemini = true
		}
		if tool.Name == "OpenAI Codex" {
			foundCodex = true
		}
	}

	assert.True(t, foundClaude, "Claude Code should be in list")
	assert.True(t, foundGemini, "Gemini CLI should be in list")
	assert.True(t, foundCodex, "Codex should be in list")
}

func TestTool_IsInstalled(t *testing.T) {
	t.Run("Check installed tool - python3", func(t *testing.T) {
		tool := tools.ToolRegistry["python3"]
		installed := tools.IsToolInstalled(tool)
		if _, err := exec.LookPath("python3"); err == nil {
			assert.True(t, installed)
		} else {
			assert.False(t, installed)
		}
	})

	t.Run("Check installed tool - node", func(t *testing.T) {
		tool := tools.ToolRegistry["node"]
		installed := tools.IsToolInstalled(tool)
		if _, err := exec.LookPath("node"); err == nil {
			assert.True(t, installed)
		} else {
			assert.False(t, installed)
		}
	})

	t.Run("Check tool not in registry", func(t *testing.T) {
		tool := tools.Tool{Name: "nonexistent"}
		installed := tools.IsToolInstalled(tool)
		assert.False(t, installed)
	})
}

func TestTool_EnsureTool(t *testing.T) {
	t.Run("Ensure already installed tool", func(t *testing.T) {
		if _, err := exec.LookPath("python3"); err != nil {
			t.Skip("python3 not installed, skipping test")
		}

		err := tools.EnsureTool("python3", "")
		assert.NoError(t, err)
	})

	t.Run("Ensure tool with version", func(t *testing.T) {
		if _, err := exec.LookPath("python3"); err != nil {
			t.Skip("python3 not installed, skipping test")
		}

		err := tools.EnsureTool("python3", "3.10")
		if err == nil {
			t.Skip("python3 3.10 not installed, skipping version test")
		}
	})
}

func TestTool_ProfileScripts(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tempDir, ".local", "share"))
	t.Setenv("APS_DATA_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, ".config"))

	profileID := "tools-test-profile"
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: tools-test-profile
display_name: Tools Test Profile
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte(""), 0600)
	require.NoError(t, err)

	t.Run("Discover no scripts", func(t *testing.T) {
		scripts, err := tools.DiscoverProfileScripts(profileID)
		require.NoError(t, err)
		assert.Empty(t, scripts, "Should find no scripts when none exist")
	})

	t.Run("Discover shell scripts", func(t *testing.T) {
		toolsDir := filepath.Join(profileDir, "tools")
		err := os.MkdirAll(toolsDir, 0755)
		require.NoError(t, err)

		scriptContent := `#!/bin/sh
echo "Hello from custom tool"
`
		scriptPath := filepath.Join(toolsDir, "custom-tool.sh")
		err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		require.NoError(t, err)

		scripts, err := tools.DiscoverProfileScripts(profileID)
		require.NoError(t, err)
		assert.Len(t, scripts, 1, "Should find one script")
		assert.Equal(t, "custom-tool", scripts[0].Name)
	})

	t.Run("Discover python scripts", func(t *testing.T) {
		toolsDir := filepath.Join(profileDir, "tools")
		err := os.MkdirAll(toolsDir, 0755)
		require.NoError(t, err)

		scriptContent := `#!/usr/bin/env python3
print("Hello from Python tool")
`
		scriptPath := filepath.Join(toolsDir, "python-tool.py")
		err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		require.NoError(t, err)

		scripts, err := tools.DiscoverProfileScripts(profileID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(scripts), 1, "Should find at least one script")
	})
}

func TestTool_ExecuteProfileTool(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("USERPROFILE", tempDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tempDir, ".local", "share"))
	t.Setenv("APS_DATA_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, ".config"))

	profileID := "execute-tool-test"
	profileDir := filepath.Join(tempDir, ".local", "share", "aps", "profiles", profileID)
	err := os.MkdirAll(profileDir, 0755)
	require.NoError(t, err)

	profileContent := `id: execute-tool-test
display_name: Execute Tool Test
`
	profilePath := filepath.Join(profileDir, "profile.yaml")
	err = os.WriteFile(profilePath, []byte(profileContent), 0644)
	require.NoError(t, err)

	secretsPath := filepath.Join(profileDir, "secrets.env")
	err = os.WriteFile(secretsPath, []byte("TEST_VAR=test_value\n"), 0600)
	require.NoError(t, err)

	t.Run("Execute shell script tool", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell scripts not executable on Windows")
		}
		toolsDir := filepath.Join(profileDir, "tools")
		err := os.MkdirAll(toolsDir, 0755)
		require.NoError(t, err)

		scriptContent := `#!/bin/sh
echo "Tool executed successfully"
`
		scriptPath := filepath.Join(toolsDir, "test-tool.sh")
		err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
		require.NoError(t, err)

		err = tools.ExecuteProfileTool(profileID, "test-tool.sh", []string{})
		require.NoError(t, err)
	})

	t.Run("Execute non-existent tool", func(t *testing.T) {
		err := tools.ExecuteProfileTool(profileID, "nonexistent-tool.sh", []string{})
		require.Error(t, err)
	})
}
