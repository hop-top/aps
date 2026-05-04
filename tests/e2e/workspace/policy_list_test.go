package workspace_e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	collab "hop.top/aps/internal/core/collaboration"
)

// seedPolicyFixture writes a workspace with a default strategy and one
// override so the row builder produces 2 rows.
func seedPolicyFixture(t *testing.T, home, wsID string) {
	t.Helper()
	now := nowUTC()
	ws := &collab.Workspace{
		ID: wsID,
		Config: collab.WorkspaceConfig{
			Name:           wsID,
			OwnerProfileID: "noor",
			DefaultPolicy:  collab.StrategyPriority,
			MaxAgents:      10,
			HeartbeatInterval: 10 * time.Second,
			SessionTimeout:    30 * time.Second,
		},
		State: collab.StateActive,
		Agents: []collab.AgentInfo{
			{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
		},
		Policy: collab.PolicyConfig{
			Default: collab.StrategyPriority,
			Overrides: map[string]collab.ResolutionStrategy{
				"doc/intro": collab.StrategyKeepLast,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	seedWorkspace(t, home, ws)
}

// TestPolicyList_FormatJSON exercises the rich row + JSON path.
func TestPolicyList_FormatJSON(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedPolicyFixture(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "policy", wsID,
		"--format", "json",
	)
	if err != nil {
		t.Fatalf("policy list: %v\nstderr: %s", err, errStr)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("decode json: %v\nstdout: %s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows (default + 1 override), got %d: %s", len(rows), out)
	}

	byKey := map[string]map[string]any{}
	for _, r := range rows {
		byKey[r["key"].(string)] = r
	}
	def, ok := byKey["default"]
	if !ok {
		t.Fatalf("missing default row in: %s", out)
	}
	if def["scope"] != "workspace" {
		t.Errorf("default scope = %v, want workspace", def["scope"])
	}
	if def["resource"] != "*" {
		t.Errorf("default resource = %v, want *", def["resource"])
	}
	if def["strategy"] != string(collab.StrategyPriority) {
		t.Errorf("default strategy = %v, want %q", def["strategy"], collab.StrategyPriority)
	}

	override, ok := byKey["doc/intro"]
	if !ok {
		t.Fatalf("missing override row in: %s", out)
	}
	if override["scope"] != "override" {
		t.Errorf("override scope = %v, want override", override["scope"])
	}
	if override["strategy"] != string(collab.StrategyKeepLast) {
		t.Errorf("override strategy = %v, want %q", override["strategy"], collab.StrategyKeepLast)
	}
}

// TestPolicyList_TableHeaders confirms the default-format table emits
// the rich row headers.
func TestPolicyList_TableHeaders(t *testing.T) {
	home := t.TempDir()
	const wsID = "ws-team"
	seedPolicyFixture(t, home, wsID)

	out, errStr, err := runAPS(t, home,
		"workspace", "policy", wsID,
	)
	if err != nil {
		t.Fatalf("policy list table: %v\nstderr: %s", err, errStr)
	}
	for _, h := range []string{"KEY", "STRATEGY", "SCOPE", "RESOURCE", "UPDATED"} {
		if !strings.Contains(out, h) {
			t.Errorf("expected header %q in table, got: %s", h, out)
		}
	}
}
