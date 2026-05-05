// Tests for policygate request_attrs publication (T-1292, T-1309).
//
// These tests exercise the introspection seam that CEL rules ride
// on: ctx.Value(policy.ContextAttrsKey)["request_attrs"]. Each
// request_attrs key (kind, visibility, …) must be reachable from
// CEL as `context.request_attrs.<key>`.
package policygate

import (
	"context"
	"testing"

	"hop.top/kit/go/runtime/policy"
)

func attrsFrom(t *testing.T, ctx context.Context) map[string]any {
	t.Helper()
	raw := ctx.Value(policy.ContextAttrsKey)
	attrs, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("ContextAttrsKey type = %T, want map[string]any", raw)
	}
	return attrs
}

func requestAttrsFrom(t *testing.T, ctx context.Context) map[string]any {
	t.Helper()
	attrs := attrsFrom(t, ctx)
	ra, ok := attrs["request_attrs"].(map[string]any)
	if !ok {
		t.Fatalf("attrs[request_attrs] type = %T, want map[string]any", attrs["request_attrs"])
	}
	return ra
}

// TestWithContextVariableAttrs_PublishesVisibility (T-1309) — the
// load-bearing contract for CEL rules that need to gate on
// context.request_attrs.visibility.
func TestWithContextVariableAttrs_PublishesVisibility(t *testing.T) {
	ctx := WithContextVariableAttrs(context.Background(), "private")

	ra := requestAttrsFrom(t, ctx)
	if got := ra["visibility"]; got != "private" {
		t.Errorf("request_attrs.visibility = %v, want %q", got, "private")
	}
	if got := ra["kind"]; got != "workspace_context" {
		t.Errorf("request_attrs.kind = %v, want %q (kind must accompany visibility)", got, "workspace_context")
	}
}

// TestWithContextVariableAttrs_SharedDefaultPublishes verifies the
// non-private path also publishes a value. CEL rules can decide
// whether to fire on shared variables themselves.
func TestWithContextVariableAttrs_SharedDefaultPublishes(t *testing.T) {
	ctx := WithContextVariableAttrs(context.Background(), "shared")

	ra := requestAttrsFrom(t, ctx)
	if got := ra["visibility"]; got != "shared" {
		t.Errorf("request_attrs.visibility = %v, want %q", got, "shared")
	}
}

// TestWithContextVariableAttrs_PreservesNote (T-1291 + T-1309) — the
// upstream --note value (set via clinote.WithContext) must survive
// the visibility merge so the existing delete-workspace-context-
// requires-note rule keeps working.
func TestWithContextVariableAttrs_PreservesNote(t *testing.T) {
	base := context.WithValue(context.Background(), policy.ContextAttrsKey, map[string]any{
		"note": "obsolete flag",
	})

	ctx := WithContextVariableAttrs(base, "private")

	attrs := attrsFrom(t, ctx)
	if attrs["note"] != "obsolete flag" {
		t.Errorf("note lost across visibility merge: %v", attrs)
	}
	ra := requestAttrsFrom(t, ctx)
	if ra["visibility"] != "private" {
		t.Errorf("visibility = %v, want %q", ra["visibility"], "private")
	}
}

// TestPublishDeletePrePersisted_PublishesKind (T-1292 regression
// guard) — the delete path must continue to surface
// context.request_attrs.kind with no bus wired.
func TestPublishDeletePrePersisted_PublishesKind(t *testing.T) {
	// Bus not wired; Publish short-circuits but ctx still carries
	// the kind enrichment (intentional — CEL evaluation uses ctx,
	// not the event payload).
	SetBus(nil)

	ctx, err := PublishDeletePrePersisted(context.Background(), "workspace_context", "feature.alpha")
	if err != nil {
		t.Fatalf("PublishDeletePrePersisted: %v", err)
	}

	ra := requestAttrsFrom(t, ctx)
	if got := ra["kind"]; got != "workspace_context" {
		t.Errorf("request_attrs.kind = %v, want workspace_context", got)
	}
}
