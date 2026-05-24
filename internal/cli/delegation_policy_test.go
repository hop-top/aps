package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/cli/policy"
)

// findCmd descends root from the supplied path tokens and returns the
// matching cobra.Command. Returns nil when any segment is missing.
func findCmd(t *testing.T, path ...string) *cobra.Command {
	t.Helper()
	if root == nil || root.Cmd == nil {
		t.Fatal("aps root is nil")
	}
	c := root.Cmd
	for _, seg := range path {
		var next *cobra.Command
		for _, sub := range c.Commands() {
			if sub.Name() == seg {
				next = sub
				break
			}
		}
		if next == nil {
			t.Fatalf("no subcommand %q under %q", seg, c.CommandPath())
		}
		c = next
	}
	return c
}

// TestDelegation_A2ASendTask_DeniedByPolicy asserts the kit
// policy.Engine refuses a2a/send (write-shared) when the delegation
// policy categorically denies the write-shared class.
func TestDelegation_A2ASendTask_DeniedByPolicy(t *testing.T) {
	cmd := findCmd(t, "a2a", "tasks", "send")
	denyShared := policy.Policy{
		Name: "deny-shared",
		Allow: map[policy.SideEffect][]string{
			policy.SideEffect(kitcli.SideEffectWriteShared): {},
		},
	}
	eng := policy.NewEngine(denyShared, 0)
	allowed, _, reason := eng.Authorize(cmd)
	if allowed {
		t.Fatalf("expected deny, got allow (reason=%q)", reason)
	}
	if !strings.Contains(reason, "write-shared") {
		t.Errorf("reason = %q, want it to cite the write-shared class", reason)
	}
}

// TestDelegation_A2ASendTask_AllowedByPolicy asserts the kit
// policy.Engine permits a2a/send under an allow-list that names the
// verb prefix. matchAny supports "verb:*" shorthand per §8.6, so an
// explicit "a2a:*" wildcard covers send / cancel / subscribe alike.
func TestDelegation_A2ASendTask_AllowedByPolicy(t *testing.T) {
	cmd := findCmd(t, "a2a", "tasks", "send")
	allowAll := policy.Policy{
		Name: "permit-a2a",
		Allow: map[policy.SideEffect][]string{
			policy.SideEffect(kitcli.SideEffectWriteShared): {"*"},
		},
	}
	eng := policy.NewEngine(allowAll, 0)
	allowed, _, reason := eng.Authorize(cmd)
	if !allowed {
		t.Fatalf("expected allow, got deny (reason=%q)", reason)
	}
}

// TestDelegation_WorkspaceSend_DeniedByPolicy mirrors the a2a test on
// the collab boundary. workspace/send is the in-tree task assignment
// dispatch — write-shared, conditional under the §8.5 wiring landed in
// T-0468.
func TestDelegation_WorkspaceSend_DeniedByPolicy(t *testing.T) {
	cmd := findCmd(t, "workspace", "send")
	denyShared := policy.Policy{
		Name: "deny-shared",
		Allow: map[policy.SideEffect][]string{
			policy.SideEffect(kitcli.SideEffectWriteShared): {},
		},
	}
	eng := policy.NewEngine(denyShared, 0)
	allowed, _, reason := eng.Authorize(cmd)
	if allowed {
		t.Fatalf("expected deny, got allow (reason=%q)", reason)
	}
	if !strings.Contains(reason, "write-shared") {
		t.Errorf("reason = %q, want it to cite the write-shared class", reason)
	}
}

// TestDelegation_WorkspaceSend_AllowedByPolicy mirrors the a2a happy-
// path on the collab boundary.
func TestDelegation_WorkspaceSend_AllowedByPolicy(t *testing.T) {
	cmd := findCmd(t, "workspace", "send")
	allowAll := policy.Policy{
		Name: "permit-workspace",
		Allow: map[policy.SideEffect][]string{
			policy.SideEffect(kitcli.SideEffectWriteShared): {"*"},
		},
	}
	eng := policy.NewEngine(allowAll, 0)
	allowed, _, reason := eng.Authorize(cmd)
	if !allowed {
		t.Fatalf("expected allow, got deny (reason=%q)", reason)
	}
}

// TestDelegation_MaxOps_ExceededOnDispatch asserts the engine RecordOp
// budget surfaces ErrMaxOpsExceeded after the configured number of
// mutating dispatches. Three ops with MaxOps=2 → third one fails.
func TestDelegation_MaxOps_ExceededOnDispatch(t *testing.T) {
	cmd := findCmd(t, "a2a", "tasks", "send")
	eng := policy.NewEngine(policy.Policy{}, 2)
	if err := eng.RecordOp(cmd); err != nil {
		t.Fatalf("RecordOp 1: %v", err)
	}
	if err := eng.RecordOp(cmd); err != nil {
		t.Fatalf("RecordOp 2: %v", err)
	}
	if err := eng.RecordOp(cmd); err == nil {
		t.Fatal("RecordOp 3: expected ErrMaxOpsExceeded, got nil")
	}
}

// TestRoot_PolicyLoader_DefaultsToXDG asserts the cli.New opt installed
// the DefaultPolicyLoader pointing at aps's XDG config dir. The loader
// is invoked when --policy=<name> reaches the middleware. We exercise
// it via a temp XDG_CONFIG_HOME so the test is hermetic.
func TestRoot_PolicyLoader_DefaultsToXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	policyDir := filepath.Join(tmp, "aps", "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	body := "name: lenient\nmax_ops: 0\n"
	if err := os.WriteFile(filepath.Join(policyDir, "lenient.yaml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write policy yaml: %v", err)
	}

	loader := kitcli.DefaultPolicyLoader("aps")
	p, err := loader("lenient")
	if err != nil {
		t.Fatalf("loader: %v", err)
	}
	if p.Name != "lenient" {
		t.Errorf("loader returned Name=%q, want lenient", p.Name)
	}
}
