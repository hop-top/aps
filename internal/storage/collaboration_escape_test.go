package storage

import "testing"

// A directory name must not end in a dot or a space: Windows drops them, so
// two ids would share one directory. "." and ".." stay verbatim so the
// containment check refuses them.
func TestEscapeWorkspaceIDTrailingDotAndSpace(t *testing.T) {
	for _, c := range []struct{ id, want string }{
		{"x..", "x.%2E"},
		{"x.", "x%2E"},
		{"x ", "x%20"},
		{"a/..", "a%2F.%2E"},
		{"...", "..%2E"},
		{".", "."},
		{"..", ".."},
		{"..x", "..x"},
		{"a.b", "a.b"},
		{"a b", "a b"},
		{"", ""},
	} {
		got := escapeWorkspaceID(c.id)
		if got != c.want {
			t.Errorf("escapeWorkspaceID(%q) = %q, want %q", c.id, got, c.want)
		}
		if back := unescapeWorkspaceID(got); back != c.id {
			t.Errorf("unescapeWorkspaceID(%q) = %q, want %q", got, back, c.id)
		}
	}
}

// Ids that differ only by trailing dots or spaces map to different names.
func TestEscapeWorkspaceIDKeepsTrailingVariantsApart(t *testing.T) {
	seen := map[string]string{}
	for _, id := range []string{"x", "x.", "x..", "x ", "x. ", "x%2E", "x.%2E"} {
		name := escapeWorkspaceID(id)
		if other, dup := seen[name]; dup {
			t.Errorf("%q and %q both map to %q", id, other, name)
		}
		seen[name] = id
	}
}
