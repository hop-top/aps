package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

// profile show must display a vault reference as written, keep literal
// values redacted, and never invoke the vault to resolve anything.
func TestProfileShowDisplaysReferencesNotResolvedSecrets(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stand-in for op")
	}
	data := t.TempDir()
	t.Setenv("APS_DATA_PATH", data)
	if err := core.CreateProfile("refshow", core.Profile{ID: "refshow", DisplayName: "R"}); err != nil {
		t.Fatal(err)
	}
	dir, err := core.GetProfileDir("refshow")
	if err != nil {
		t.Fatal(err)
	}
	body := "PLAIN=hunter2-literal\nBOS_AGENT_TOKEN=op://V/item/token\n"
	if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	// An `op` that records any invocation and would leak a value if called.
	bin := t.TempDir()
	marker := filepath.Join(bin, "called")
	script := "#!/bin/sh\ntouch " + marker + "\nprintf '%s' resolved-secret-value\n"
	if err := os.WriteFile(filepath.Join(bin, "op"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	p, err := core.LoadProfile("refshow")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := writeProfileShowHuman(&buf, p); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "BOS_AGENT_TOKEN: op://V/item/token") {
		t.Errorf("reference not displayed as written:\n%s", out)
	}
	if !strings.Contains(out, "PLAIN: ***redacted***") {
		t.Errorf("literal secret not redacted:\n%s", out)
	}
	for _, leak := range []string{"hunter2-literal", "resolved-secret-value"} {
		if strings.Contains(out, leak) {
			t.Errorf("output leaks %q:\n%s", leak, out)
		}
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("profile show invoked the vault; inspect commands must not resolve")
	}
}
