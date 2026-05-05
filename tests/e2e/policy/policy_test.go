// Package policy_e2e exercises the kit/runtime/policy wiring in the
// aps CLI (T-1292). Each test compiles the aps binary, sets
// APS_POLICY_FILE to the bundled default rules, and asserts deny / allow
// behaviour for the two trivial defaults shipped today:
//
//   - delete-session-requires-note
//   - delete-workspace-context-requires-note
//
// Tests intentionally bypass the bus token (no APS_BUS_TOKEN set) so
// the network adapter never runs; the policy engine still subscribes
// to the in-process memory bus, which is what fires pre_persisted on
// every CLI delete handler.
package policy_e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	collab "hop.top/aps/internal/core/collaboration"
)

var apsBinary string

func TestMain(m *testing.M) {
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "compile aps binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.Remove(apsBinary)
	os.Exit(code)
}

func compileBinary() error {
	binName := "aps-policy-e2e"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	apsBinary = filepath.Join(os.TempDir(), binName)

	rootDir, err := filepath.Abs("../../..")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runAPS runs the compiled binary against an isolated HOME with a
// fresh process bus. APS_BUS_TOKEN is set to a dummy value so the
// in-process memBus is initialised and the policy engine can subscribe
// (without one, internal/cli/bus.go skips bus init entirely). The
// network adapter still tries to dial but fails fast; the local
// memBus path is what powers the synchronous pre_persisted gate.
func runAPS(t *testing.T, home, policyFile string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)

	override := map[string]bool{
		"HOME":             true,
		"USERPROFILE":      true,
		"XDG_DATA_HOME":    true,
		"XDG_CONFIG_HOME":  true,
		"APS_DATA_PATH":    true,
		"APS_POLICY_FILE":  true,
		"APS_BUS_TOKEN":    true,
		"APS_BUS_ADDR":     true,
		"BUS_TOKEN":        true,
		"KIT_POLICY_DISABLE": true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"APS_DATA_PATH=" + filepath.Join(home, ".local", "share", "aps"),
		"APS_POLICY_FILE=" + policyFile,
		// Dummy token so internal/cli/bus.go init wires the memBus
		// (which the policy engine subscribes to). The websocket dial
		// to localhost:8080 fails fast; tests don't rely on hub
		// delivery.
		"APS_BUS_TOKEN=test-token-for-policy-e2e",
		"APS_BUS_ADDR=ws://127.0.0.1:1/unused",
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if override[key] {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// writeBundledPolicies writes the bundled aps default policies.yaml to
// the given path so APS_POLICY_FILE points at a known ruleset. The
// tests don't read internal/cli/embedded directly to keep this test
// package as a black-box e2e — every byte the binary loads is one the
// test has placed itself.
func writeBundledPolicies(t *testing.T, path string) {
	t.Helper()
	body := `policies:
  - name: delete-session-requires-note
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || !has(context.request_attrs.kind) || context.request_attrs.kind != "session" || context.note != ""'
    effect: allow
    otherwise: deny
    message: "deleting a session requires --note explaining why"

  - name: delete-workspace-context-requires-note
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || !has(context.request_attrs.kind) || context.request_attrs.kind != "workspace_context" || context.note != ""'
    effect: allow
    otherwise: deny
    message: "deleting a workspace context variable requires --note explaining why"
`
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write policies.yaml: %v", err)
	}
}

// writeRegistry writes a registry.json under the test home so the
// running binary's session.GetRegistry() loads fixture sessions on
// first call. Body is the raw JSON (caller composes the map).
func writeRegistry(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".local", "share", "aps", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir sessions dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write registry.json: %v", err)
	}
}

// seedContext writes context.json for a workspace under the data dir
// the binary loads from.
func seedContext(t *testing.T, home, wsID string, vars []collab.ContextVariable) {
	t.Helper()
	wsDir := filepath.Join(home, ".local", "share", "aps", "collaboration", wsID)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", wsDir, err)
	}
	body, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		t.Fatalf("marshal context: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "context.json"), body, 0o644); err != nil {
		t.Fatalf("write context.json: %v", err)
	}
}

const sessionFixture = `{
  "s-x": {
    "id": "s-x",
    "profile_id": "alice",
    "command": "shell",
    "pid": 100,
    "status": "active",
    "tier": "premium",
    "type": "",
    "workspace_id": "team-a",
    "created_at": "2026-01-01T10:00:00Z",
    "last_seen_at": "2026-01-01T10:05:00Z"
  }
}`

func TestSessionDelete_DenyWithoutNote(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, sessionFixture)
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	_, stderr, err := runAPS(t, home, policyFile, "session", "delete", "s-x", "--force")
	if err == nil {
		t.Fatalf("expected non-zero exit; stderr=%q", stderr)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError; got %T (%v)", err, err)
	}
	if got := exitErr.ExitCode(); got != 4 {
		t.Errorf("exit code = %d; want 4 (CONFLICT). stderr=%q", got, stderr)
	}
	if !strings.Contains(stderr, "delete-session-requires-note") {
		t.Errorf("stderr missing policy name; got: %s", stderr)
	}
}

func TestSessionDelete_AllowWithNote(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeRegistry(t, home, sessionFixture)
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	stdout, stderr, err := runAPS(t, home, policyFile,
		"session", "delete", "s-x", "--force", "--note", "fixture cleanup")
	if err != nil {
		t.Fatalf("delete with --note failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Session s-x deleted") {
		t.Errorf("expected delete confirmation in stdout; got: %s", stdout)
	}
}

func TestWorkspaceCtxDelete_DenyWithoutNote(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	_, stderr, err := runAPS(t, home, policyFile,
		"--profile", "noor",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit; stderr=%q", stderr)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError; got %T (%v)", err, err)
	}
	if got := exitErr.ExitCode(); got != 4 {
		t.Errorf("exit code = %d; want 4 (CONFLICT). stderr=%q", got, stderr)
	}
	if !strings.Contains(stderr, "delete-workspace-context-requires-note") {
		t.Errorf("stderr missing policy name; got: %s", stderr)
	}
}

func TestWorkspaceCtxDelete_AllowWithNote(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	stdout, stderr, err := runAPS(t, home, policyFile,
		"--profile", "noor",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "obsoleted by feature flag retirement",
	)
	if err != nil {
		t.Fatalf("delete with --note failed: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted 'feature.alpha'") {
		t.Errorf("expected delete confirmation; got: %s", stdout)
	}
}

func TestPolicyEngine_BadYAMLFailsLoud(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	policyFile := filepath.Join(home, "broken.yaml")
	if err := os.WriteFile(policyFile, []byte(`policies:
  - name: bad-rule
    on: not.a.valid.topic
    when: 'true'
    effect: allow
    otherwise: deny
`), 0o600); err != nil {
		t.Fatalf("write broken policy: %v", err)
	}

	_, stderr, err := runAPS(t, home, policyFile, "version")
	if err == nil {
		t.Fatalf("expected non-zero exit on misconfig; stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "policy") {
		t.Errorf("stderr should mention policy on misconfig; got: %s", stderr)
	}
}
