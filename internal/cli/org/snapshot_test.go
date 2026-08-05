package org

import (
	"reflect"
	"testing"
)

// TestSanitizeMermaidID: every non-alphanumeric rune becomes `_`; an
// empty id never yields an empty node id.
func TestSanitizeMermaidID(t *testing.T) {
	cases := map[string]string{
		"alice":      "alice",
		"a.b-c":      "a_b_c",
		"team lead":  "team_lead",
		"Grüße":      "Gr__e",
		"":           "_",
		"42-answers": "42_answers",
	}
	for in, want := range cases {
		if got := sanitizeMermaidID(in); got != want {
			t.Errorf("sanitizeMermaidID(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMermaidNodeIDs_CollisionSuffix: ids that sanitize to the same
// token get deterministic numeric suffixes in input (sorted) order.
func TestMermaidNodeIDs_CollisionSuffix(t *testing.T) {
	got := mermaidNodeIDs([]string{"a.b", "a-b", "a_b"})
	want := map[string]string{
		"a.b": "a_b",
		"a-b": "a_b_2",
		"a_b": "a_b_3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mermaidNodeIDs collision mapping = %v, want %v", got, want)
	}
}
