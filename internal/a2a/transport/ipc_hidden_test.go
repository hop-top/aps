package transport

import (
	"os"
	"path/filepath"
	"testing"
)

// A staged ".<name>.tmp-<rand>" file from an in-flight atomic write must not be
// consumed (or deleted) by the queue poller.
func TestProcessQueuedMessagesSkipsHiddenStaging(t *testing.T) {
	dir := t.TempDir()
	tr, err := NewIPCTransport(&IPCConfig{ProfileID: "p", QueuePath: dir}, nil)
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, ".1_x.json.tmp-abc")
	if err := os.WriteFile(staged, []byte(`{"messageId":"m"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tr.processQueuedMessages()
	if _, err := os.Stat(staged); err != nil {
		t.Fatalf("staged file consumed: %v", err)
	}
}
