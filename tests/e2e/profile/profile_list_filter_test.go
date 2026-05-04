package profile_e2e

import (
	"strings"
	"testing"
)

// seedFilterProfiles populates a home with profiles that exercise
// every filter dimension exposed by `aps profile list`.
func seedFilterProfiles(t *testing.T, home string) {
	t.Helper()

	// alpha: capability=webhooks, role=owner, squad=core, workspace=team-a,
	// has identity, has secrets, tone=neutral
	writeProfile(t, home, "alpha", `id: alpha
display_name: Alpha
roles:
  - owner
capabilities:
  - webhooks
squads:
  - core
workspace:
  name: team-a
  scope: shared
identity:
  did: did:example:alpha
persona:
  tone: neutral
`)
	writeSecrets(t, home, "alpha", "GITHUB_TOKEN=tok\n")

	// beta: capability=github, role=auditor, squad=ops, workspace=team-b,
	// no identity, no real secrets, tone=warm
	writeProfile(t, home, "beta", `id: beta
display_name: Beta
roles:
  - auditor
capabilities:
  - github
squads:
  - ops
workspace:
  name: team-b
  scope: shared
persona:
  tone: warm
`)

	// gamma: no overlap with the above on filtered dimensions
	writeProfile(t, home, "gamma", `id: gamma
display_name: Gamma
`)
}

func TestProfileList_FilterCapability(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, stderr, err := runAPS(t, home, "profile", "list", "--capability", "webhooks")
	if err != nil {
		t.Fatalf("profile list --capability: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "beta") || strings.Contains(stdout, "gamma") {
		t.Errorf("unexpected non-match in output: %s", stdout)
	}

	// Zero-match: bogus capability value should produce no profile rows.
	stdout, _, err = runAPS(t, home, "profile", "list", "--capability", "does-not-exist")
	if err != nil {
		t.Fatalf("profile list --capability bogus: %v", err)
	}
	for _, id := range []string{"alpha", "beta", "gamma"} {
		if strings.Contains(stdout, id) {
			t.Errorf("expected no rows for bogus capability, found %q in: %s", id, stdout)
		}
	}
}

func TestProfileList_FilterRole(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, _, err := runAPS(t, home, "profile", "list", "--role", "owner")
	if err != nil {
		t.Fatalf("profile list --role: %v", err)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha (role=owner): %s", stdout)
	}
	if strings.Contains(stdout, "beta") {
		t.Errorf("did not expect beta (role=auditor): %s", stdout)
	}
}

func TestProfileList_FilterSquad(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, _, err := runAPS(t, home, "profile", "list", "--squad", "ops")
	if err != nil {
		t.Fatalf("profile list --squad: %v", err)
	}
	if !strings.Contains(stdout, "beta") {
		t.Errorf("expected beta (squad=ops): %s", stdout)
	}
	if strings.Contains(stdout, "alpha") {
		t.Errorf("did not expect alpha (squad=core): %s", stdout)
	}
}

func TestProfileList_FilterWorkspace(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, _, err := runAPS(t, home, "profile", "list", "--workspace", "team-a")
	if err != nil {
		t.Fatalf("profile list --workspace: %v", err)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha (workspace=team-a): %s", stdout)
	}
	if strings.Contains(stdout, "beta") {
		t.Errorf("did not expect beta (workspace=team-b): %s", stdout)
	}
}

func TestProfileList_FilterTone(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, _, err := runAPS(t, home, "profile", "list", "--tone", "warm")
	if err != nil {
		t.Fatalf("profile list --tone: %v", err)
	}
	if !strings.Contains(stdout, "beta") {
		t.Errorf("expected beta (tone=warm): %s", stdout)
	}
	if strings.Contains(stdout, "alpha") {
		t.Errorf("did not expect alpha (tone=neutral): %s", stdout)
	}
}

func TestProfileList_FilterHasIdentity(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, _, err := runAPS(t, home, "profile", "list", "--has-identity")
	if err != nil {
		t.Fatalf("profile list --has-identity: %v", err)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha (has identity): %s", stdout)
	}
	if strings.Contains(stdout, "beta") {
		t.Errorf("did not expect beta (no identity): %s", stdout)
	}

	// Negation: --has-identity=false should match beta + gamma but
	// NOT alpha.
	stdout, _, err = runAPS(t, home, "profile", "list", "--has-identity=false")
	if err != nil {
		t.Fatalf("profile list --has-identity=false: %v", err)
	}
	if strings.Contains(stdout, "alpha") {
		t.Errorf("did not expect alpha (has identity) under =false: %s", stdout)
	}
	if !strings.Contains(stdout, "beta") {
		t.Errorf("expected beta under --has-identity=false: %s", stdout)
	}
}

func TestProfileList_FilterHasSecrets(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedFilterProfiles(t, home)

	stdout, _, err := runAPS(t, home, "profile", "list", "--has-secrets")
	if err != nil {
		t.Fatalf("profile list --has-secrets: %v", err)
	}
	if !strings.Contains(stdout, "alpha") {
		t.Errorf("expected alpha (has secrets): %s", stdout)
	}
	if strings.Contains(stdout, "beta") {
		t.Errorf("did not expect beta (no secrets): %s", stdout)
	}
}
