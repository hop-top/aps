package directory

import (
	"bytes"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
	"hop.top/kit/go/console/progress"
)

// TestDiscover_EmitsProgress verifies the discover subcommand emits the
// connect + fetch phase events through the kit/console/progress reporter
// wired into cmd.Context() by kit/cli. The stub discovery client returns
// an empty result set so the call path completes successfully.
func TestDiscover_EmitsProgress(t *testing.T) {
	stubResolver(t, func(name string) (*core.Instance, error) {
		return &core.Instance{Name: name, DirectoryEndpoint: "https://dir.test"}, nil
	})

	var buf bytes.Buffer
	r := progress.JSONL(&buf)

	root, _ := newRootWithDiscover()
	root.SetArgs([]string{"discover", "--capability", "x"})
	root.SetContext(progress.WithReporter(t.Context(), r))
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `"phase":"connect"`) {
		t.Errorf("expected connect phase, got: %q", got)
	}
	if !strings.Contains(got, `"phase":"fetch"`) {
		t.Errorf("expected fetch phase, got: %q", got)
	}
	if !strings.Contains(got, `"ok":true`) {
		t.Errorf("expected ok:true on fetch completion, got: %q", got)
	}
}
