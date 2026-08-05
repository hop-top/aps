package org_e2e

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestOrgCheck_Clean: a consistent hierarchy yields exit 0.
func TestOrgCheck_Clean(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	_, stderr, err := runAPS(t, home, "org", "check")
	if err != nil {
		t.Fatalf("org check on clean hierarchy: %v\nstderr: %s", err, stderr)
	}
}

// TestOrgCheck_Cycle: a reporting cycle is a finding and exits non-zero.
func TestOrgCheck_Cycle(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedCycle(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "check")
	if err == nil {
		t.Fatalf("org check on cyclic hierarchy: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout, "cycle") {
		t.Errorf("stdout missing cycle finding:\n%s\nstderr: %s", stdout, stderr)
	}
	for _, id := range []string{"loopa", "loopb"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("stdout missing cycle member %q:\n%s", id, stdout)
		}
	}
}

// TestOrgCheck_Dangling: reports_to pointing at a missing profile is a
// finding.
func TestOrgCheck_Dangling(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "orphan", "id: orphan\ndisplay_name: Orphan\nreports_to: ghost\n")

	stdout, _, err := runAPS(t, home, "org", "check")
	if err == nil {
		t.Fatalf("org check with dangling reports_to: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout, "dangling_reports_to") {
		t.Errorf("stdout missing dangling_reports_to finding:\n%s", stdout)
	}
	if !strings.Contains(stdout, "ghost") {
		t.Errorf("stdout missing dangling target id:\n%s", stdout)
	}
}

// TestOrgCheck_UnknownType: a type outside {"", agent, human} is a
// finding.
func TestOrgCheck_UnknownType(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeProfile(t, home, "robby", "id: robby\ndisplay_name: Robby\ntype: robot\n")

	stdout, _, err := runAPS(t, home, "org", "check")
	if err == nil {
		t.Fatalf("org check with unknown type: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout, "unknown_type") {
		t.Errorf("stdout missing unknown_type finding:\n%s", stdout)
	}
}

// TestOrgCheck_UnloadableFile: a profile.yaml that fails to load must
// surface as a finding, not silently vanish (unlike ListProfilesFull).
func TestOrgCheck_UnloadableFile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)
	// ID mismatch between directory and content makes LoadProfile fail.
	writeProfile(t, home, "broken", "id: not-broken\ndisplay_name: Broken\n")

	stdout, _, err := runAPS(t, home, "org", "check")
	if err == nil {
		t.Fatalf("org check with unloadable profile: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout, "unloadable_profile") {
		t.Errorf("stdout missing unloadable_profile finding:\n%s", stdout)
	}
	if !strings.Contains(stdout, "broken") {
		t.Errorf("stdout missing unloadable profile id:\n%s", stdout)
	}
}

// TestOrgCheck_JSONRoundTrip: --format json emits a parseable array of
// findings with kind/profiles/message keys.
func TestOrgCheck_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedCycle(t, home)
	writeProfile(t, home, "orphan", "id: orphan\ndisplay_name: Orphan\nreports_to: ghost\n")

	stdout, stderr, err := runAPS(t, home, "org", "check", "--format", "json")
	if err == nil {
		t.Fatalf("org check with findings: want non-zero exit\nstdout: %s", stdout)
	}
	var rows []map[string]any
	if jsonErr := json.Unmarshal([]byte(stdout), &rows); jsonErr != nil {
		t.Fatalf("stdout is not a JSON array: %v\nstdout: %s\nstderr: %s", jsonErr, stdout, stderr)
	}
	kinds := map[string]bool{}
	for _, r := range rows {
		kind, _ := r["kind"].(string)
		kinds[kind] = true
		for _, key := range []string{"kind", "profiles", "message"} {
			if _, ok := r[key]; !ok {
				t.Errorf("finding row missing %q key: %v", key, r)
			}
		}
	}
	for _, want := range []string{"cycle", "dangling_reports_to"} {
		if !kinds[want] {
			t.Errorf("missing finding kind %q in %v", want, kinds)
		}
	}
}
