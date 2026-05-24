package migrate

import (
	"bytes"
	"strings"
	"testing"

	"hop.top/kit/go/console/progress"
)

// TestExecuteMigration_EmitsPerItemProgress confirms executeMigration
// emits a per-item migrate event for each messenger in the input batch,
// with Bytes/Total populated so subscribers can render a progress bar.
func TestExecuteMigration_EmitsPerItemProgress(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APS_DATA_PATH", t.TempDir())

	var buf bytes.Buffer
	ctx := progress.WithReporter(t.Context(), progress.JSONL(&buf))

	messengers := []messengerMigrate{
		{Name: "alpha", Type: "messenger", Scope: "global"},
		{Name: "beta", Type: "messenger", Scope: "global"},
	}

	if err := executeMigration(ctx, messengers); err != nil {
		t.Fatalf("executeMigration: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		`"phase":"migrate"`,
		`"item":"alpha"`,
		`"item":"beta"`,
		`"total":2`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in events; got %q", want, got)
		}
	}
}
