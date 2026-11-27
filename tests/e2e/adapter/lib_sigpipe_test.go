package adapter_e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLib_ManifestWithManyDuplicateKeys guards aps__yaml_value in
// adapters/_lib.sh. It keeps the first of several matching manifest
// lines; taking that first line from a pipe made the writer die of
// SIGPIPE under pipefail once the matches exceeded a pipe buffer, and
// the caller aborted.
//
// aps_init_backend only calls the helper inside $(...), where bash
// drops errexit, so the failure is latent there: the captured value is
// still right. A direct call under set -e (a caller that does not wrap
// it in a substitution) is where it surfaces, so that is what the
// probe script does.
//
// The lib resolves backend.yaml next to the calling script and is
// found three directories up, so the test builds that layout in a temp
// dir around a copy of the real _lib.sh.
func TestLib_ManifestWithManyDuplicateKeys(t *testing.T) {
	requireBash(t)
	t.Parallel()

	root := t.TempDir()
	backendDir := filepath.Join(root, "adapters", "probe", "backends", "stub")
	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	lib, err := os.ReadFile(filepath.Join(adaptersRoot(t), "_lib.sh"))
	if err != nil {
		t.Fatalf("read _lib.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "adapters", "_lib.sh"), lib, 0o644); err != nil {
		t.Fatalf("write _lib.sh: %v", err)
	}

	manifest := "binary: sh\n" + strings.Repeat("binary: other\n", 10000)
	if err := os.WriteFile(filepath.Join(backendDir, "backend.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write backend.yaml: %v", err)
	}

	script := "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		". \"$(cd \"$(dirname \"${BASH_SOURCE[0]}\")\" && pwd)/../../../_lib.sh\"\n" +
		"aps__yaml_value probe \"$(dirname \"${BASH_SOURCE[0]}\")/backend.yaml\" binary > \"$1\"\n" +
		"echo \"BIN=$(cat \"$1\")\"\n"
	scriptPath := filepath.Join(backendDir, "probe.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command("bash", scriptPath, filepath.Join(root, "value"))
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + root}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aps__yaml_value failed on a large manifest: %v\noutput: %.500s", err, out)
	}
	if !strings.Contains(string(out), "BIN=sh\n") {
		t.Errorf("first duplicate did not win; output: %.500s", out)
	}
}
