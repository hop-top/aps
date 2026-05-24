package capability

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/progress"
)

// TestInstall_EmitsProgress wires the install subcommand against an
// isolated APS_DATA_PATH so the on-disk copy succeeds, then asserts the
// install phase event sequence emerges from the kit progress reporter.
func TestInstall_EmitsProgress(t *testing.T) {
	t.Setenv("APS_DATA_PATH", t.TempDir())

	source := t.TempDir()

	var buf bytes.Buffer
	ctx := progress.WithReporter(t.Context(), progress.JSONL(&buf))

	root := &cobra.Command{Use: "aps"}
	root.AddCommand(newInstallCmd())
	root.SetContext(ctx)
	root.SetArgs([]string{"install", source, "--name", "demo"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `"phase":"install"`) {
		t.Errorf("expected install phase, got: %q", got)
	}
	if !strings.Contains(got, `"item":"demo"`) {
		t.Errorf("expected item=demo, got: %q", got)
	}
	if !strings.Contains(got, `"ok":true`) {
		t.Errorf("expected ok:true on completion, got: %q", got)
	}
}
