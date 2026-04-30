package cli

import "testing"

// TestHints_ProfileList asserts the post-command hint registered in
// registerHints is wired on root.Hints under the namespaced key
// "profile list" (T-0346).
func TestHints_ProfileList(t *testing.T) {
	hints := root.Hints.Lookup("profile list")
	if len(hints) == 0 {
		t.Fatal("expected at least one hint registered for 'profile list'")
	}
	want := "Run `aps profile show <id>` for details."
	if hints[0].Message != want {
		t.Errorf("hint message = %q, want %q", hints[0].Message, want)
	}
}
