package org_e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wantTreeAll is the exact tree rendering of the seedTree fixture.
const wantTreeAll = `Chief (ceo) (human)
  VP Eng (vp)
    Eng One (eng1)
      Intern (intern)
    Eng Two (eng2)
`

// wantMermaidAll is the exact mermaid rendering of the seedTree
// fixture: nodes sorted by id, then edges sorted by manager, report.
const wantMermaidAll = `flowchart TD
    ceo["Chief (human)"]
    eng1["Eng One"]
    eng2["Eng Two"]
    intern["Intern"]
    vp["VP Eng"]
    ceo --> vp
    eng1 --> intern
    vp --> eng1
    vp --> eng2
`

// TestOrgSnapshot_TreeAll: default scope (--all) + default format
// (tree) renders the exact ASCII forest, humans marked.
func TestOrgSnapshot_TreeAll(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot")
	if err != nil {
		t.Fatalf("org snapshot: %v\nstderr: %s", err, stderr)
	}
	if stdout != wantTreeAll {
		t.Errorf("tree output mismatch\nwant:\n%s\ngot:\n%s", wantTreeAll, stdout)
	}
}

// TestOrgSnapshot_Mermaid: --snapshot-format mermaid renders the exact
// flowchart, no timestamp.
func TestOrgSnapshot_Mermaid(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot", "--snapshot-format", "mermaid")
	if err != nil {
		t.Fatalf("org snapshot --snapshot-format mermaid: %v\nstderr: %s", err, stderr)
	}
	if stdout != wantMermaidAll {
		t.Errorf("mermaid output mismatch\nwant:\n%s\ngot:\n%s", wantMermaidAll, stdout)
	}
}

// TestOrgSnapshot_RootScope: --root limits the snapshot to the subtree.
func TestOrgSnapshot_RootScope(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot", "--root", "vp")
	if err != nil {
		t.Fatalf("org snapshot --root vp: %v\nstderr: %s", err, stderr)
	}
	want := `VP Eng (vp)
  Eng One (eng1)
    Intern (intern)
  Eng Two (eng2)
`
	if stdout != want {
		t.Errorf("subtree output mismatch\nwant:\n%s\ngot:\n%s", want, stdout)
	}
}

// TestOrgSnapshot_RootScopeUnknown: unknown --root id errors.
func TestOrgSnapshot_RootScopeUnknown(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot", "--root", "nobody")
	if err == nil {
		t.Fatalf("org snapshot --root nobody: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stderr, "nobody") {
		t.Errorf("stderr missing unknown root id:\n%s", stderr)
	}
}

// TestOrgSnapshot_SquadScope: --squad selects membership read off disk.
func TestOrgSnapshot_SquadScope(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot", "--squad", "core-team")
	if err != nil {
		t.Fatalf("org snapshot --squad core-team: %v\nstderr: %s", err, stderr)
	}
	// eng1 and eng2 are members; their manager vp is out of scope, so
	// both render as roots of the forest.
	want := `Eng One (eng1)
Eng Two (eng2)
`
	if stdout != want {
		t.Errorf("squad scope output mismatch\nwant:\n%s\ngot:\n%s", want, stdout)
	}
}

// TestOrgSnapshot_SquadScopeEmpty: zero members is a clear error.
func TestOrgSnapshot_SquadScopeEmpty(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot", "--squad", "ghost")
	if err == nil {
		t.Fatalf("org snapshot --squad ghost: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stderr, `no profiles are members of squad "ghost"`) {
		t.Errorf("stderr missing zero-member error:\n%s", stderr)
	}
}

// TestOrgSnapshot_ScopeFlagsMutuallyExclusive: --root and --squad
// cannot combine.
func TestOrgSnapshot_ScopeFlagsMutuallyExclusive(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	stdout, _, err := runAPS(t, home, "org", "snapshot", "--root", "vp", "--squad", "core-team")
	if err == nil {
		t.Fatalf("org snapshot --root + --squad: want non-zero exit\nstdout: %s", stdout)
	}
}

// TestOrgSnapshot_JSONDeterminism: identical state yields identical
// JSON bytes except the captured_at line.
func TestOrgSnapshot_JSONDeterminism(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)

	first, stderr, err := runAPS(t, home, "org", "snapshot", "--snapshot-format", "json")
	if err != nil {
		t.Fatalf("org snapshot --snapshot-format json: %v\nstderr: %s", err, stderr)
	}
	second, stderr, err := runAPS(t, home, "org", "snapshot", "--snapshot-format", "json")
	if err != nil {
		t.Fatalf("org snapshot --snapshot-format json (second run): %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(first, "captured_at") {
		t.Errorf("json output missing captured_at:\n%s", first)
	}
	if stripCapturedAt(first) != stripCapturedAt(second) {
		t.Errorf("json output not deterministic modulo captured_at\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	for _, want := range []string{`"scope"`, `"nodes"`, `"edges"`} {
		if !strings.Contains(first, want) {
			t.Errorf("json output missing %s key:\n%s", want, first)
		}
	}
}

// stripCapturedAt drops lines carrying the capture timestamp.
func stripCapturedAt(s string) string {
	var kept []string
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, "captured_at") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// TestOrgSnapshot_OutputFile: --output writes the rendering to the
// file instead of stdout.
func TestOrgSnapshot_OutputFile(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedTree(t, home)
	out := filepath.Join(home, "org.tree")

	stdout, stderr, err := runAPS(t, home, "org", "snapshot", "--output", out)
	if err != nil {
		t.Fatalf("org snapshot --output: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stdout, "Chief") {
		t.Errorf("tree must not be duplicated on stdout with --output:\n%s", stdout)
	}
	data, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatalf("read --output file: %v", readErr)
	}
	if string(data) != wantTreeAll {
		t.Errorf("--output file mismatch\nwant:\n%s\ngot:\n%s", wantTreeAll, string(data))
	}
}

// TestOrgSnapshot_CyclicTerminates: cyclic data must terminate, still
// render, warn, and exit non-zero.
func TestOrgSnapshot_CyclicTerminates(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedCycle(t, home)

	stdout, stderr, err := runAPS(t, home, "org", "snapshot")
	if err == nil {
		t.Fatalf("org snapshot on cyclic data: want non-zero exit\nstdout: %s", stdout)
	}
	if !strings.Contains(stdout, "loopa") || !strings.Contains(stdout, "loopb") {
		t.Errorf("cyclic members must still render:\n%s", stdout)
	}
	if !strings.Contains(stdout+stderr, "cycle") {
		t.Errorf("missing cycle warning\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

// TestOrgSnapshot_CyclicMermaidStillEmits: json/yaml/mermaid emit the
// finite node/edge sets even on cyclic data (non-zero exit).
func TestOrgSnapshot_CyclicMermaidStillEmits(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	seedCycle(t, home)

	stdout, _, err := runAPS(t, home, "org", "snapshot", "--snapshot-format", "mermaid")
	if err == nil {
		t.Fatalf("org snapshot mermaid on cyclic data: want non-zero exit\nstdout: %s", stdout)
	}
	for _, want := range []string{"flowchart TD", "loopa --> loopb", "loopb --> loopa"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("mermaid output missing %q:\n%s", want, stdout)
		}
	}
}
