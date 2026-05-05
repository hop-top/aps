// T-1308 — unit tests for the workspace-aware aps principal resolver.
//
// The resolver layers per-workspace AgentRole lookup over kit's
// DefaultPrincipalResolver. These tests stub workspaceRoleLookup +
// callingProfileResolver via package vars so the assertions stay focused
// on resolver semantics — storage / FS coverage lives in the e2e tests.
package cli

import (
	"context"
	"errors"
	"os"
	"testing"

	collab "hop.top/aps/internal/core/collaboration"

	"hop.top/kit/go/runtime/policy"
)

// withResolverStubs swaps the package-level lookup hooks for the
// duration of a test. Restores them via t.Cleanup.
func withResolverStubs(t *testing.T, profile string, role collab.AgentRole, lookupErr error) {
	t.Helper()
	prevLookup := workspaceRoleLookup
	prevProfile := callingProfileResolver
	t.Cleanup(func() {
		workspaceRoleLookup = prevLookup
		callingProfileResolver = prevProfile
	})
	callingProfileResolver = func() string { return profile }
	workspaceRoleLookup = func(string, string) (collab.AgentRole, error) {
		return role, lookupErr
	}
}

// ctxWithWorkspace mirrors what policygate.PublishDeletePrePersistedWithAttrs
// stamps into ctx — request_attrs.{kind,workspace_id}.
func ctxWithWorkspace(wsID string) context.Context {
	attrs := map[string]any{
		"request_attrs": map[string]any{
			"kind":         "workspace_context",
			"workspace_id": wsID,
		},
	}
	return context.WithValue(context.Background(), policy.ContextAttrsKey, attrs)
}

func TestApsPrincipalResolver_OwnerFromMembership(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "")
	withResolverStubs(t, "noor", collab.RoleOwner, nil)

	got := apsPrincipalResolver(ctxWithWorkspace("ws-team"))

	if got.Role != string(collab.RoleOwner) {
		t.Errorf("Role = %q, want %q", got.Role, collab.RoleOwner)
	}
	if got.ID != "noor" {
		t.Errorf("ID = %q, want %q", got.ID, "noor")
	}
	if got.Source != "aps.workspace" {
		t.Errorf("Source = %q, want %q", got.Source, "aps.workspace")
	}
}

func TestApsPrincipalResolver_ContributorFromMembership(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "")
	withResolverStubs(t, "sami", collab.RoleContributor, nil)

	got := apsPrincipalResolver(ctxWithWorkspace("ws-team"))

	if got.Role != string(collab.RoleContributor) {
		t.Errorf("Role = %q, want %q", got.Role, collab.RoleContributor)
	}
}

func TestApsPrincipalResolver_NoMembershipFallsBackToEnv(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "ops")
	t.Setenv("USER", "")
	// Lookup returns empty role (membership absent). Should fall through
	// to kit's default which surfaces KIT_POLICY_ROLE=ops.
	withResolverStubs(t, "ghost", "", nil)

	got := apsPrincipalResolver(ctxWithWorkspace("ws-team"))

	if got.Role != "ops" {
		t.Errorf("Role = %q, want %q (env fallback)", got.Role, "ops")
	}
	if got.Source != "env" {
		t.Errorf("Source = %q, want %q", got.Source, "env")
	}
}

func TestApsPrincipalResolver_NoWorkspaceContextUsesEnv(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "ops")
	t.Setenv("USER", "")
	withResolverStubs(t, "noor", collab.RoleOwner, nil)

	// No request_attrs.workspace_id on ctx — resolver MUST short-circuit
	// to kit default before doing any storage work.
	got := apsPrincipalResolver(context.Background())

	if got.Role != "ops" {
		t.Errorf("Role = %q, want %q", got.Role, "ops")
	}
}

func TestApsPrincipalResolver_NoCallingProfileUsesEnv(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "ops")
	t.Setenv("USER", "")
	withResolverStubs(t, "", collab.RoleOwner, nil) // would-be owner, but no profile

	got := apsPrincipalResolver(ctxWithWorkspace("ws-team"))

	if got.Role != "ops" {
		t.Errorf("Role = %q, want %q (env fallback)", got.Role, "ops")
	}
}

func TestApsPrincipalResolver_LookupErrorFallsBackToEnv(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "ops")
	t.Setenv("USER", "")
	withResolverStubs(t, "noor", "", errors.New("disk gone"))

	got := apsPrincipalResolver(ctxWithWorkspace("ws-team"))

	if got.Role != "ops" {
		t.Errorf("Role = %q, want %q (env fallback on lookup err)", got.Role, "ops")
	}
}

func TestApsPrincipalResolver_PanicSafety(t *testing.T) {
	t.Setenv("KIT_POLICY_ROLE", "fallback-role")
	t.Setenv("USER", "")

	prevLookup := workspaceRoleLookup
	prevProfile := callingProfileResolver
	t.Cleanup(func() {
		workspaceRoleLookup = prevLookup
		callingProfileResolver = prevProfile
	})
	callingProfileResolver = func() string { return "noor" }
	workspaceRoleLookup = func(string, string) (collab.AgentRole, error) {
		panic("simulated storage panic")
	}

	// Resolver MUST recover and return the kit default — never propagate
	// the panic to the policy engine, which runs synchronously on the
	// CLI's hot path.
	got := apsPrincipalResolver(ctxWithWorkspace("ws-team"))

	if got.Role != "fallback-role" {
		t.Errorf("Role = %q, want %q (env fallback on panic)", got.Role, "fallback-role")
	}
}

func TestCallingProfileResolver_ApsProfileEnv(t *testing.T) {
	// The default callingProfileResolver consults root.Viper first, then
	// APS_PROFILE. Tests that don't boot the CLI see Viper as nil-ish
	// (no flag set), so APS_PROFILE wins. Confirm we don't accidentally
	// short-circuit on an empty viper string.
	t.Setenv("APS_PROFILE", "operator-x")

	if got := callingProfileResolver(); got != "operator-x" {
		t.Errorf("callingProfileResolver() = %q, want %q", got, "operator-x")
	}
}

func TestCallingProfileResolver_ViperOverridesEnv(t *testing.T) {
	if root.Viper == nil {
		t.Skip("root.Viper not initialised; covered by integration tests")
	}
	prev := root.Viper.GetString("profile")
	root.Viper.Set("profile", "viper-pid")
	t.Cleanup(func() { root.Viper.Set("profile", prev) })
	t.Setenv("APS_PROFILE", "env-pid")

	if got := callingProfileResolver(); got != "viper-pid" {
		t.Errorf("callingProfileResolver() = %q, want %q (viper wins)", got, "viper-pid")
	}
}

// TestWorkspaceIDFromContext exercises the request_attrs reader directly;
// the rest of the suite relies on it indirectly via ctxWithWorkspace.
func TestWorkspaceIDFromContext(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{"nil ctx", nil, ""},
		{"no attrs", context.Background(), ""},
		{
			"workspace_id present",
			context.WithValue(context.Background(), policy.ContextAttrsKey, map[string]any{
				"request_attrs": map[string]any{"workspace_id": "ws-x"},
			}),
			"ws-x",
		},
		{
			"request_attrs missing",
			context.WithValue(context.Background(), policy.ContextAttrsKey, map[string]any{
				"note": "hi",
			}),
			"",
		},
		{
			"workspace_id wrong type",
			context.WithValue(context.Background(), policy.ContextAttrsKey, map[string]any{
				"request_attrs": map[string]any{"workspace_id": 42},
			}),
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workspaceIDFromContext(tc.ctx); got != tc.want {
				t.Errorf("workspaceIDFromContext = %q, want %q", got, tc.want)
			}
		})
	}
}

// Smoke test: the package var defaults compile and don't depend on a
// running CLI. callingProfileResolver hits Viper but tolerates a nil
// instance. workspaceRoleLookup hits real storage — bound to a temp dir
// so we don't pollute the developer's home.
func TestPackageVarDefaultsLookupable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APS_DATA_PATH", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	t.Setenv("HOME", tmp)

	// Empty inputs short-circuit; no storage call.
	role, err := workspaceRoleLookup("", "noor")
	if err != nil {
		t.Errorf("empty workspaceID: err = %v", err)
	}
	if role != "" {
		t.Errorf("empty workspaceID: role = %q, want empty", role)
	}

	// Non-existent workspace returns ("", nil) — see WorkspaceNotFoundError
	// branch in the resolver implementation.
	role, err = workspaceRoleLookup("ws-does-not-exist", "noor")
	if err != nil {
		t.Errorf("missing workspace: unexpected err = %v", err)
	}
	if role != "" {
		t.Errorf("missing workspace: role = %q, want empty", role)
	}

	// Sanity: callingProfileResolver tolerates whatever Viper state the
	// test runner has (default zero); never panics.
	_ = callingProfileResolver()

	// Sanity: APS_PROFILE wins when Viper has no value.
	t.Setenv("APS_PROFILE", "fallback-pid")
	if root.Viper != nil {
		root.Viper.Set("profile", "")
	}
	if got := callingProfileResolver(); got != "fallback-pid" {
		t.Errorf("APS_PROFILE fallback = %q, want %q", got, "fallback-pid")
	}
	_ = os.Unsetenv("APS_PROFILE")
}
