package workspace_e2e

import (
	"strings"
	"testing"
	"time"

	collab "hop.top/aps/internal/core/collaboration"
)

// seedListWorkspaces populates a home with three fixture workspaces
// covering each filter dimension (T-0430).
func seedListWorkspaces(t *testing.T, home string) {
	t.Helper()
	now := nowUTC()

	seedWorkspace(t, home, &collab.Workspace{
		ID: "ws-alpha",
		Config: collab.WorkspaceConfig{
			Name:           "alpha",
			OwnerProfileID: "alice",
		},
		State: collab.StateActive,
		Agents: []collab.AgentInfo{
			{ProfileID: "alice", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
			{ProfileID: "bob", Role: collab.RoleContributor, JoinedAt: now, LastSeen: now, Status: "online"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	seedWorkspace(t, home, &collab.Workspace{
		ID: "ws-beta",
		Config: collab.WorkspaceConfig{
			Name:           "beta",
			OwnerProfileID: "bob",
		},
		State: collab.StateActive,
		Agents: []collab.AgentInfo{
			{ProfileID: "bob", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
		},
		CreatedAt: now.Add(time.Hour),
		UpdatedAt: now.Add(time.Hour),
	})

	seedWorkspace(t, home, &collab.Workspace{
		ID: "ws-gamma",
		Config: collab.WorkspaceConfig{
			Name:           "gamma",
			OwnerProfileID: "alice",
		},
		State: collab.StateArchived,
		Agents: []collab.AgentInfo{
			{ProfileID: "alice", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "offline"},
		},
		CreatedAt: now.Add(2 * time.Hour),
		UpdatedAt: now.Add(2 * time.Hour),
	})
}

func TestWorkspaceList_RichDefault(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedListWorkspaces(t, home)

	stdout, stderr, err := runAPS(t, home, "workspace", "list")
	if err != nil {
		t.Fatalf("workspace list: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{"ID", "NAME", "STATUS", "OWNER", "MEMBERS", "alpha", "beta", "gamma"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("workspace list output missing %q\nstdout: %s", want, stdout)
		}
	}
}

func TestWorkspaceList_FilterMember(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedListWorkspaces(t, home)

	stdout, _, err := runAPS(t, home, "workspace", "list", "--member", "bob")
	if err != nil {
		t.Fatalf("workspace list --member: %v", err)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha (member=bob): %s", stdout)
	}
	if !strings.Contains(stdout, "beta") {
		t.Errorf("expected beta (member=bob): %s", stdout)
	}
	if strings.Contains(stdout, "gamma") {
		t.Errorf("did not expect gamma (no bob): %s", stdout)
	}

	// Zero-match
	stdout, _, err = runAPS(t, home, "workspace", "list", "--member", "no-such")
	if err != nil {
		t.Fatalf("workspace list --member bogus: %v", err)
	}
	for _, n := range []string{"alpha", "beta", "gamma"} {
		if strings.Contains(stdout, n) {
			t.Errorf("expected zero rows for bogus member, found %q in: %s", n, stdout)
		}
	}
}

func TestWorkspaceList_FilterOwner(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedListWorkspaces(t, home)

	stdout, _, err := runAPS(t, home, "workspace", "list", "--owner", "alice")
	if err != nil {
		t.Fatalf("workspace list --owner: %v", err)
	}
	if !strings.Contains(stdout, "alpha") || !strings.Contains(stdout, "gamma") {
		t.Errorf("expected alpha + gamma (owner=alice): %s", stdout)
	}
	if strings.Contains(stdout, "beta") {
		t.Errorf("did not expect beta (owner=bob): %s", stdout)
	}
}

func TestWorkspaceList_FilterArchived(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedListWorkspaces(t, home)

	stdout, _, err := runAPS(t, home, "workspace", "list", "--archived")
	if err != nil {
		t.Fatalf("workspace list --archived: %v", err)
	}
	if !strings.Contains(stdout, "gamma") {
		t.Errorf("expected gamma (archived): %s", stdout)
	}
	if strings.Contains(stdout, "alpha") || strings.Contains(stdout, "beta") {
		t.Errorf("did not expect non-archived in --archived: %s", stdout)
	}

	// Negation: --archived=false should hide gamma but show alpha + beta.
	stdout, _, err = runAPS(t, home, "workspace", "list", "--archived=false")
	if err != nil {
		t.Fatalf("workspace list --archived=false: %v", err)
	}
	if strings.Contains(stdout, "gamma") {
		t.Errorf("did not expect gamma under --archived=false: %s", stdout)
	}
	if !strings.Contains(stdout, "alpha") || !strings.Contains(stdout, "beta") {
		t.Errorf("expected alpha+beta under --archived=false: %s", stdout)
	}
}
