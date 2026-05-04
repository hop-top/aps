package squad

import (
	"io"
	"os"
	"testing"
)

// captureStdout swaps os.Stdout for a pipe so list-command output
// rendered via os.Stdout-bound writers (kit/output, listing.RenderList)
// can be asserted in tests. The returned restore func reinstates
// os.Stdout; readPipe drains the buffered side.
//
// Returns (old stdout, write end, restore). A package-level pipeReader
// is stashed for readPipe.
var pipeReader *os.File

func captureStdout(t *testing.T) (*os.File, *os.File, func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	pipeReader = r
	restore := func() {
		os.Stdout = old
	}
	return old, w, restore
}

// readPipe drains the package-level pipeReader stashed by captureStdout.
// Caller must have already closed the write end.
func readPipe(t *testing.T) string {
	t.Helper()
	if pipeReader == nil {
		return ""
	}
	defer func() { pipeReader = nil }()
	data, err := io.ReadAll(pipeReader)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	return string(data)
}
