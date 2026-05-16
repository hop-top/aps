package chat

import (
	"strings"
	"testing"
)

func TestStorePathRejectsTraversal(t *testing.T) {
	store := NewStore(t.TempDir())
	cases := []string{
		"../escape",
		"../../escape",
		"foo/bar",
		"foo\\bar",
		"a/../b",
		"with space",
		"hash#",
		"",
	}
	for _, sessionID := range cases {
		if _, err := store.path(sessionID); err == nil {
			t.Errorf("path(%q) accepted, want rejection", sessionID)
		}
	}
}

func TestStorePathAcceptsSafeIDs(t *testing.T) {
	store := NewStore(t.TempDir())
	cases := []string{
		"chat-abc123",
		"00000000-0000-0000-0000-000000000000",
		"alpha.beta_gamma",
	}
	for _, sessionID := range cases {
		path, err := store.path(sessionID)
		if err != nil {
			t.Errorf("path(%q) rejected: %v", sessionID, err)
			continue
		}
		if !strings.HasSuffix(path, sessionID+".json") {
			t.Errorf("path(%q) = %q, missing expected suffix", sessionID, path)
		}
	}
}

func TestStoreSaveRejectsTraversalSessionID(t *testing.T) {
	store := NewStore(t.TempDir())
	tr := &Transcript{SessionID: "../../etc/passwd"}
	if err := store.Save(tr); err == nil {
		t.Fatal("Save accepted traversal sessionID, want rejection")
	}
}
