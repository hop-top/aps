package acp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/aps/internal/core/protocol"
)

// TestHandleSkillList tests the skill/list JSON-RPC method
func TestHandleSkillList(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()

	t.Setenv("APS_DATA_PATH", filepath.Join(tmpDir, "data", "aps"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))

	// Create test skills in the profile path. On macOS the global
	// skills path intentionally follows Application Support instead of
	// XDG_DATA_HOME, while profile paths stay under APS_DATA_PATH.
	profileSkillsDir := filepath.Join(tmpDir, "data", "aps", "profiles", "testagent", "skills")
	require.NoError(t, os.MkdirAll(profileSkillsDir, 0755))

	setupTestSkill(t, profileSkillsDir, "acp-skill-1")
	setupTestSkill(t, profileSkillsDir, "acp-skill-2")

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
			expectedMinCount: 2,
		},
		{
			name:             "list skills with default profile",
			params:           SkillListParams{},
			expectedError:    false,
			expectedMinCount: 2,
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

	t.Setenv("APS_DATA_PATH", filepath.Join(tmpDir, "data", "aps"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))

	// Create test skill
	profileSkillsDir := filepath.Join(tmpDir, "data", "aps", "profiles", "testagent", "skills")
	require.NoError(t, os.MkdirAll(profileSkillsDir, 0755))

	setupTestSkill(t, profileSkillsDir, "acp-skill")

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
			name: "get skill successfully",
			params: SkillGetParams{
				SkillID:   "acp-skill",
				ProfileID: "testagent",
			},
			expectedError: false,
			expectedSkill: "acp-skill",
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
	tmpDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", filepath.Join(tmpDir, "data", "aps"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))

	profileSkillsDir := filepath.Join(tmpDir, "data", "aps", "profiles", "testagent", "skills")
	require.NoError(t, os.MkdirAll(profileSkillsDir, 0755))
	setupTestSkill(t, profileSkillsDir, "test-skill")
	require.NoError(t, os.WriteFile(
		filepath.Join(profileSkillsDir, "test-skill", "scripts", "process.sh"),
		[]byte("#!/bin/sh\nprintf 'skill=%s\\n' \"$APS_SKILL_ID\"\nprintf 'script=%s\\n' \"$APS_SKILL_SCRIPT\"\nprintf 'args=%s\\n' \"$APS_SKILL_ARGS_JSON\"\n"),
		0755,
	))

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
		expectedCode  ErrorCode
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
			name: "unknown skill",
			params: SkillInvokeParams{
				SkillID:   "missing-skill",
				Script:    "process.sh",
				SessionID: "sess_123",
			},
			expectedError: true,
			expectedCode:  ErrCodeResourceNotFound,
		},
		{
			name: "unknown script",
			params: SkillInvokeParams{
				SkillID:   "test-skill",
				Script:    "missing.sh",
				SessionID: "sess_123",
			},
			expectedError: true,
			expectedCode:  ErrCodeResourceNotFound,
		},
		{
			name: "script traversal is rejected",
			params: SkillInvokeParams{
				SkillID:   "test-skill",
				Script:    "../SKILL.md",
				SessionID: "sess_123",
			},
			expectedError: true,
			expectedCode:  ErrCodeResourceNotFound,
		},
		{
			name: "missing skill ID",
			params: SkillInvokeParams{
				Script:    "process.sh",
				SessionID: "sess_123",
			},
			expectedError: true,
			expectedCode:  ErrCodeInvalidParams,
		},
		{
			name: "missing script",
			params: SkillInvokeParams{
				SkillID:   "test-skill",
				SessionID: "sess_123",
			},
			expectedError: true,
			expectedCode:  ErrCodeInvalidParams,
		},
		{
			name: "missing session ID",
			params: SkillInvokeParams{
				SkillID: "test-skill",
				Script:  "process.sh",
			},
			expectedError: true,
			expectedCode:  ErrCodeInvalidParams,
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
				if tt.expectedCode != 0 {
					assert.Equal(t, tt.expectedCode, resp.Error.Code)
				}
			} else {
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Result)

				resultMap, ok := resp.Result.(map[string]interface{})
				require.True(t, ok)

				assert.NotEmpty(t, resultMap["runId"])
				assert.NotContains(t, resultMap["runId"], "placeholder")
				assert.NotEmpty(t, resultMap["terminalId"])
				assert.Equal(t, "test-skill", resultMap["skillId"])
				assert.Equal(t, "process.sh", resultMap["script"])

				terminalID, ok := resultMap["terminalId"].(string)
				require.True(t, ok)
				exitCode, err := server.terminalManager.WaitForExit(terminalID)
				require.NoError(t, err)
				assert.Equal(t, 0, exitCode)
				output, err := server.terminalManager.GetOutput(terminalID)
				require.NoError(t, err)
				assert.Contains(t, output, "skill=test-skill")
				assert.Contains(t, output, "script=process.sh")
				assert.Contains(t, output, `"input":"test"`)
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
