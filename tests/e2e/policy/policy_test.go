// Package policy_e2e exercises the kit/runtime/policy wiring in the
// aps CLI (T-1292). Each test compiles the aps binary, sets
// APS_POLICY_FILE to the bundled default rules, and asserts deny / allow
// behaviour for the trivial defaults shipped today:
//
//   - delete-session-requires-note
//   - delete-workspace-context-requires-note
//   - cross-agent-context-delete-requires-owner (T-1302)
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

  - name: cross-agent-context-delete-requires-owner
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || !has(context.request_attrs.kind) || context.request_attrs.kind != "workspace_context" || !has(context.request_attrs.visibility) || context.request_attrs.visibility != "shared" || principal.role == "owner"'
    effect: allow
    otherwise: deny
    message: "deleting a shared workspace context variable requires workspace owner role"
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
	// T-1302 — the bundled cross-agent-context-delete-requires-owner
	// rule denies shared deletes unless principal.role == "owner". Seed
	// noor as workspace owner so the role gate passes; the test asserts
	// the --note rule, not the role rule.
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
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

// seedWorkspace writes a minimal manifest.yaml + state.json for a
// workspace under the same data root LoadWorkspace reads from. agents
// is the membership list seeded into state.json. T-1308 e2e tests
// exercise the workspace-aware principal resolver against real on-disk
// fixtures; this helper keeps them readable.
func seedWorkspace(t *testing.T, home, wsID string, agents []collab.AgentInfo) {
	t.Helper()
	wsDir := filepath.Join(home, ".local", "share", "aps", "collaboration", wsID)
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", wsDir, err)
	}
	manifest := []byte("name: " + wsID + "\nowner_profile_id: noor\n")
	if err := os.WriteFile(filepath.Join(wsDir, "manifest.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest.yaml: %v", err)
	}
	state := map[string]any{
		"id":         wsID,
		"state":      "active",
		"agents":     agents,
		"policy":     map[string]any{"default": "priority"},
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal workspace state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "state.json"), body, 0o644); err != nil {
		t.Fatalf("write state.json: %v", err)
	}
}

// writeRolePolicy writes a policy file that exercises principal.role
// composition with the default note rules. Adds an observer-only-deny
// rule on workspace_context deletes; combined with deny-overrides this
// tests T-1308's workspace-aware principal resolver end-to-end.
func writeRolePolicy(t *testing.T, path string) {
	t.Helper()
	body := `policies:
  - name: delete-workspace-context-requires-note
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || !has(context.request_attrs.kind) || context.request_attrs.kind != "workspace_context" || context.note != ""'
    effect: allow
    otherwise: deny
    message: "deleting a workspace context variable requires --note explaining why"

  - name: workspace-context-write-requires-owner
    on: kit.runtime.entity.pre_persisted
    when: 'payload.Op != "delete" || !has(context.request_attrs.kind) || context.request_attrs.kind != "workspace_context" || principal.role == "owner"'
    effect: allow
    otherwise: deny
    message: "only role:owner may delete workspace context variables"
`
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write policies.yaml: %v", err)
	}
}

// TestWorkspaceCtxDelete_OwnerAllowed — T-1308 acceptance: an owner
// session passes the role gate.
func TestWorkspaceCtxDelete_OwnerAllowed(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
	})
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeRolePolicy(t, policyFile)

	stdout, stderr, err := runAPS(t, home, policyFile,
		"--profile", "noor",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "obsoleted",
	)
	if err != nil {
		t.Fatalf("owner delete denied: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted 'feature.alpha'") {
		t.Errorf("expected delete confirmation; got: %s", stdout)
	}
}

// TestWorkspaceCtxDelete_ContributorDenied — T-1308 acceptance: a
// contributor hits the role gate even with --note.
func TestWorkspaceCtxDelete_ContributorDenied(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
	})
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
		{ProfileID: "sami", Role: collab.RoleContributor, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeRolePolicy(t, policyFile)

	_, stderr, err := runAPS(t, home, policyFile,
		"--profile", "sami",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "trying to clean up",
	)
	if err == nil {
		t.Fatalf("contributor delete should have been denied; stderr=%q", stderr)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError; got %T (%v)", err, err)
	}
	if got := exitErr.ExitCode(); got != 4 {
		t.Errorf("exit code = %d; want 4. stderr=%q", got, stderr)
	}
	if !strings.Contains(stderr, "workspace-context-write-requires-owner") {
		t.Errorf("stderr missing role policy name; got: %s", stderr)
	}
}

// TestWorkspaceCtxDelete_NoMembershipDenied — T-1308 acceptance: a
// profile with no membership has empty principal.role and is denied.
func TestWorkspaceCtxDelete_NoMembershipDenied(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
	})
	// Workspace has only noor as a member; ghost is not in the list.
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeRolePolicy(t, policyFile)

	_, stderr, err := runAPS(t, home, policyFile,
		"--profile", "ghost",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "outsider",
	)
	if err == nil {
		t.Fatalf("non-member delete should have been denied; stderr=%q", stderr)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError; got %T (%v)", err, err)
	}
	if got := exitErr.ExitCode(); got != 4 {
		t.Errorf("exit code = %d; want 4. stderr=%q", got, stderr)
	}
}

// TestWorkspaceCtxDelete_EnvFallbackForUnknownWorkspace — T-1308
// acceptance: when workspace_id resolves to a non-existent workspace,
// the resolver falls open to KIT_POLICY_ROLE without crashing. With
// KIT_POLICY_ROLE=owner the role-gate rule allows the op. Verifies the
// "no workspace context" failure mode required by the task spec.
func TestWorkspaceCtxDelete_EnvFallbackForUnknownWorkspace(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-missing"
	now := time.Now().UTC().Truncate(time.Second)
	// Seed context but NOT the workspace manifest — the workspace
	// directory has only context.json, so LoadWorkspace returns
	// WorkspaceNotFoundError. Resolver must fall through to the env path
	// rather than panic.
	seedContext(t, home, wsID, []collab.ContextVariable{
		{Key: "feature.alpha", Value: "true", Version: 1, UpdatedBy: "noor", UpdatedAt: now},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeRolePolicy(t, policyFile)

	cmd := exec.Command(apsBinary,
		"--profile", "noor",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "kicking the env fallback",
	)
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
		"KIT_POLICY_ROLE":  true,
		"KIT_POLICY_DISABLE": true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"APS_DATA_PATH=" + filepath.Join(home, ".local", "share", "aps"),
		"APS_POLICY_FILE=" + policyFile,
		"APS_BUS_TOKEN=test-token-for-policy-e2e",
		"APS_BUS_ADDR=ws://127.0.0.1:1/unused",
		"KIT_POLICY_ROLE=owner",
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
	if err := cmd.Run(); err != nil {
		t.Fatalf("env-fallback delete failed: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Deleted 'feature.alpha'") {
		t.Errorf("expected delete confirmation; got: %s", stdout.String())
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

// ----------------------------------------------------------------------
// T-1302 — cross-agent-context-delete-requires-owner
//
// The bundled rule denies deletes of SHARED workspace_context variables
// unless principal.role == "owner". Private variables are exempt: the
// storage-layer visibility filter already keeps them invisible to
// non-writers, so reaching the delete path means the caller wrote them.
// Tests cover the four canonical cases the rule decides.
// ----------------------------------------------------------------------

// TestWorkspaceCtxDelete_T1302_OwnerSharedAllowed — owner deleting a
// shared var passes both the note rule (--note supplied) and the new
// T-1302 owner gate.
func TestWorkspaceCtxDelete_T1302_OwnerSharedAllowed(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{
			Key: "feature.alpha", Value: "true", Version: 1,
			UpdatedBy: "noor", UpdatedAt: now,
			Visibility: collab.VisibilityShared,
		},
	})
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	stdout, stderr, err := runAPS(t, home, policyFile,
		"--profile", "noor",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "obsoleted",
	)
	if err != nil {
		t.Fatalf("owner shared delete denied: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted 'feature.alpha'") {
		t.Errorf("expected delete confirmation; got: %s", stdout)
	}
}

// TestWorkspaceCtxDelete_T1302_ContributorSharedDenied — contributor
// deleting a shared var trips the T-1302 owner gate. --note is supplied
// so the note rule allows; only the role gate denies.
func TestWorkspaceCtxDelete_T1302_ContributorSharedDenied(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{
			Key: "feature.alpha", Value: "true", Version: 1,
			UpdatedBy: "noor", UpdatedAt: now,
			Visibility: collab.VisibilityShared,
		},
	})
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
		{ProfileID: "sami", Role: collab.RoleContributor, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	_, stderr, err := runAPS(t, home, policyFile,
		"--profile", "sami",
		"workspace", "ctx", "delete", "feature.alpha",
		"--workspace", wsID,
		"--force",
		"--note", "trying to clean up",
	)
	if err == nil {
		t.Fatalf("contributor shared delete should have been denied; stderr=%q", stderr)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError; got %T (%v)", err, err)
	}
	if got := exitErr.ExitCode(); got != 4 {
		t.Errorf("exit code = %d; want 4. stderr=%q", got, stderr)
	}
	if !strings.Contains(stderr, "cross-agent-context-delete-requires-owner") {
		t.Errorf("stderr missing T-1302 rule name; got: %s", stderr)
	}
}

// TestWorkspaceCtxDelete_T1302_OwnerPrivateAllowed — owner deleting
// their own private var is allowed: visibility=private exempts the
// rule, and the note rule passes via --note.
func TestWorkspaceCtxDelete_T1302_OwnerPrivateAllowed(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{
			Key: "secret.token", Value: "hunter2", Version: 1,
			UpdatedBy: "noor", UpdatedAt: now,
			Visibility: collab.VisibilityPrivate,
		},
	})
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	stdout, stderr, err := runAPS(t, home, policyFile,
		"--profile", "noor",
		"workspace", "ctx", "delete", "secret.token",
		"--workspace", wsID,
		"--force",
		"--note", "rotated",
	)
	if err != nil {
		t.Fatalf("owner private delete denied: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted 'secret.token'") {
		t.Errorf("expected delete confirmation; got: %s", stdout)
	}
}

// TestWorkspaceCtxDelete_T1302_ContributorPrivateAllowed — contributor
// deleting their own private var is allowed: the storage-layer
// visibility filter already gates cross-profile reads, so reaching the
// delete path means the variable belongs to the caller. T-1302 exempts
// visibility=private from the owner-only rule.
func TestWorkspaceCtxDelete_T1302_ContributorPrivateAllowed(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	const wsID = "ws-team"
	now := time.Now().UTC().Truncate(time.Second)
	seedContext(t, home, wsID, []collab.ContextVariable{
		{
			Key: "scratch.note", Value: "wip", Version: 1,
			UpdatedBy: "sami", UpdatedAt: now,
			Visibility: collab.VisibilityPrivate,
		},
	})
	seedWorkspace(t, home, wsID, []collab.AgentInfo{
		{ProfileID: "noor", Role: collab.RoleOwner, JoinedAt: now, LastSeen: now, Status: "online"},
		{ProfileID: "sami", Role: collab.RoleContributor, JoinedAt: now, LastSeen: now, Status: "online"},
	})
	policyFile := filepath.Join(home, "policies.yaml")
	writeBundledPolicies(t, policyFile)

	stdout, stderr, err := runAPS(t, home, policyFile,
		"--profile", "sami",
		"workspace", "ctx", "delete", "scratch.note",
		"--workspace", wsID,
		"--force",
		"--note", "discarded",
	)
	if err != nil {
		t.Fatalf("contributor private delete denied: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "Deleted 'scratch.note'") {
		t.Errorf("expected delete confirmation; got: %s", stdout)
	}
}
