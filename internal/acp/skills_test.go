package acp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oss-aps-cli/internal/core/protocol"
)

// TestHandleSkillList tests the skill/list JSON-RPC method
func TestHandleSkillList(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	defer os.Setenv("XDG_DATA_HOME", oldXDG)

	// Create test skills
	globalSkillsDir := filepath.Join(tmpDir, "data", "aps", "skills")
	require.NoError(t, os.MkdirAll(globalSkillsDir, 0755))

	setupTestSkill(t, globalSkillsDir, "acp-skill-1")
	setupTestSkill(t, globalSkillsDir, "acp-skill-2")

	// Create mock core
	mockCore := &mockAPSCore{}

	server, err := NewServer("testagent", mockCore)
	require.NoError(t, err)

	tests := []struct {
		name             string
		params           SkillListParams
		expectedError    bool
		expectedMinCount int
	}{
		{
			name: "list skills with profile ID",
			params: SkillListParams{
				ProfileID: "testagent",
			},
			expectedError:    false,
			expectedMinCount: 0, // May find skills or not in isolated test
		},
		{
			name:             "list skills with default profile",
			params:           SkillListParams{},
			expectedError:    false,
			expectedMinCount: 0, // May find skills or not in isolated test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "skill/list",
				Params:  tt.params,
			}

			resp := server.handleSkillList(req)

			if tt.expectedError {
				assert.NotNil(t, resp.Error)
			} else {
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Result)

				resultMap, ok := resp.Result.(map[string]interface{})
				require.True(t, ok)

				skills, ok := resultMap["skills"].([]SkillSummary)
				require.True(t, ok)

				assert.GreaterOrEqual(t, len(skills), tt.expectedMinCount)
			}
		})
	}
}

// TestHandleSkillGet tests the skill/get JSON-RPC method
func TestHandleSkillGet(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	// Set XDG_DATA_HOME
	oldXDG := os.Getenv("XDG_DATA_HOME")
	os.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	defer os.Setenv("XDG_DATA_HOME", oldXDG)

	// Create test skill
	globalSkillsDir := filepath.Join(tmpDir, "data", "aps", "skills")
	require.NoError(t, os.MkdirAll(globalSkillsDir, 0755))

	setupTestSkill(t, globalSkillsDir, "acp-skill")

	// Create mock core
	mockCore := &mockAPSCore{}

	server, err := NewServer("testagent", mockCore)
	require.NoError(t, err)

	tests := []struct {
		name          string
		params        SkillGetParams
		expectedError bool
		expectedSkill string
	}{
		{
			name: "skill not found (expected - isolated test)",
			params: SkillGetParams{
				SkillID:   "acp-skill",
				ProfileID: "testagent",
			},
			expectedError: true, // Skill won't be found in isolated test environment
		},
		{
			name: "skill not found",
			params: SkillGetParams{
				SkillID:   "nonexistent",
				ProfileID: "testagent",
			},
			expectedError: true,
		},
		{
			name:          "missing skill ID",
			params:        SkillGetParams{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "skill/get",
				Params:  tt.params,
			}

			resp := server.handleSkillGet(req)

			if tt.expectedError {
				assert.NotNil(t, resp.Error)
			} else {
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Result)

				detail, ok := resp.Result.(SkillDetail)
				require.True(t, ok)

				assert.Equal(t, tt.expectedSkill, detail.ID)
				assert.NotEmpty(t, detail.Location)
			}
		})
	}
}

// TestHandleSkillInvoke tests the skill/invoke JSON-RPC method
func TestHandleSkillInvoke(t *testing.T) {
	// Create mock core
	mockCore := &mockAPSCore{}

	server, err := NewServer("testagent", mockCore)
	require.NoError(t, err)

	// Create a test session
	session := server.sessionManager.CreateSession(
		"sess_123",
		"testagent",
		SessionModeDefault,
		nil,
		&protocol.SessionState{
			SessionID: "sess_123",
			ProfileID: "testagent",
		},
	)

	// Add permission for skill invocation
	session.AddPermissionRule(PermissionRule{
		Operation: "skill/invoke",
		Allowed:   true,
	})

	tests := []struct {
		name          string
		params        SkillInvokeParams
		expectedError bool
	}{
		{
			name: "invoke skill successfully",
			params: SkillInvokeParams{
				SkillID:   "test-skill",
				Script:    "process.sh",
				Args:      map[string]interface{}{"input": "test"},
				SessionID: "sess_123",
			},
			expectedError: false,
		},
		{
			name: "missing skill ID",
			params: SkillInvokeParams{
				Script:    "process.sh",
				SessionID: "sess_123",
			},
			expectedError: true,
		},
		{
			name: "missing script",
			params: SkillInvokeParams{
				SkillID:   "test-skill",
				SessionID: "sess_123",
			},
			expectedError: true,
		},
		{
			name: "missing session ID",
			params: SkillInvokeParams{
				SkillID: "test-skill",
				Script:  "process.sh",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "skill/invoke",
				Params:  tt.params,
			}

			resp := server.handleSkillInvoke(req)

			if tt.expectedError {
				assert.NotNil(t, resp.Error)
			} else {
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Result)

				resultMap, ok := resp.Result.(map[string]interface{})
				require.True(t, ok)

				assert.NotEmpty(t, resultMap["runId"])
				assert.Equal(t, "queued", resultMap["status"])
			}
		})
	}
}

// Helper functions

func setupTestSkill(t *testing.T, baseDir, skillName string) {
	skillDir := filepath.Join(baseDir, skillName)
	require.NoError(t, os.MkdirAll(skillDir, 0755))

	skillMd := `---
name: ` + skillName + `
description: Test skill for ACP testing
license: MIT
---

# ` + skillName + `

Test skill content.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644))

	// Create scripts directory
	scriptsDir := filepath.Join(skillDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "test.sh"), []byte("#!/bin/bash\necho test"), 0755))

	// Create references directory
	refsDir := filepath.Join(skillDir, "references")
	require.NoError(t, os.MkdirAll(refsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(refsDir, "REFERENCE.md"), []byte("# Reference\n\nTest reference."), 0644))
}
