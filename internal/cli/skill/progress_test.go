package skill

import (
	"bytes"
	"strings"
	"testing"

	"hop.top/kit/go/console/progress"
)

// TestRunSkillScript_EmitsProgress confirms `aps skill run` emits the
// exec + exit phase events through the kit progress reporter wired into
// ctx, mirroring the envelope contract used by `aps run`.
func TestRunSkillScript_EmitsProgress(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	profileID := "noor"
	writeSkillFixture(t, dataDir, profileID, "runner", map[string]string{
		"hello.sh": "#!/bin/sh\necho ok\n",
	})

	var pbuf bytes.Buffer
	ctx := progress.WithReporter(t.Context(), progress.JSONL(&pbuf))

	var stdout, stderr bytes.Buffer
	if err := runSkillScript(ctx, profileID, []string{"runner", "hello.sh"}, 1, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("runSkillScript: %v", err)
	}

	got := pbuf.String()
	if !strings.Contains(got, `"phase":"exec"`) {
		t.Errorf("expected exec phase, got: %q", got)
	}
	if !strings.Contains(got, `"phase":"exit"`) {
		t.Errorf("expected exit phase, got: %q", got)
	}
	if !strings.Contains(got, `"ok":true`) {
		t.Errorf("expected ok:true on exit, got: %q", got)
	}
}

// TestRunSkillScript_EmitsFailureProgress confirms a non-zero exit
// surfaces as ok:false on the exit phase.
func TestRunSkillScript_EmitsFailureProgress(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	profileID := "noor"
	writeSkillFixture(t, dataDir, profileID, "runner", map[string]string{
		"boom.sh": "#!/bin/sh\nexit 1\n",
	})

	var pbuf bytes.Buffer
	ctx := progress.WithReporter(t.Context(), progress.JSONL(&pbuf))

	if err := runSkillScript(ctx, profileID, []string{"runner", "boom.sh"}, 1, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected non-zero exit error")
	}

	got := pbuf.String()
	if !strings.Contains(got, `"phase":"exit"`) || !strings.Contains(got, `"ok":false`) {
		t.Errorf("expected exit phase with ok:false, got: %q", got)
	}
}
