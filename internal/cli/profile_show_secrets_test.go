package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hop.top/aps/internal/core"
)

func secretsShowFixture(t *testing.T, id, body string) *core.Profile {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg"))
	t.Setenv("APS_DATA_PATH", filepath.Join(home, "data"))
	if err := core.CreateProfile(id, core.Profile{ID: id, DisplayName: id}); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		dir, err := core.GetProfileDir(id)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "secrets.env"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p, err := core.LoadProfile(id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func showHuman(t *testing.T, p *core.Profile) string {
	t.Helper()
	var buf bytes.Buffer
	if err := writeProfileShowHuman(&buf, p); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// show and list must agree: a profile whose secrets.env is only the
// comment template has no secrets in both views.
func TestProfileShowAgreesWithListOnTemplateOnlySecrets(t *testing.T) {
	p := secretsShowFixture(t, "tmpl", "")
	if profileToSummaryRow(*p).HasSecrets {
		t.Fatal("list row reports secrets for a template-only file")
	}
	out := showHuman(t, p)
	if !strings.Contains(out, "- Secrets: none") {
		t.Errorf("show should report no secrets:\n%s", out)
	}
	if strings.Contains(out, "Secrets: present") {
		t.Errorf("show still reports present:\n%s", out)
	}
}

func TestProfileShowReportsKeyCount(t *testing.T) {
	p := secretsShowFixture(t, "two", "A=sentinel-one\nB=sentinel-two\n")
	if !profileToSummaryRow(*p).HasSecrets {
		t.Fatal("list row reports no secrets for a populated file")
	}
	out := showHuman(t, p)
	if !strings.Contains(out, "- Secrets: 2 keys") {
		t.Errorf("show should report the key count:\n%s", out)
	}
	if strings.Contains(out, "sentinel-") {
		t.Errorf("show leaks a value:\n%s", out)
	}
}

func TestProfileShowOneKeyIsSingular(t *testing.T) {
	p := secretsShowFixture(t, "one", "A=1\n")
	if out := showHuman(t, p); !strings.Contains(out, "- Secrets: 1 key\n") {
		t.Errorf("want singular count:\n%s", out)
	}
}

// A malformed file is reported as unreadable, with location only.
func TestProfileShowMalformedSecretsReportedWithoutContent(t *testing.T) {
	p := secretsShowFixture(t, "bad", "GOOD=1\nBAD!KEY=leak-sentinel\n")
	if profileToSummaryRow(*p).HasSecrets {
		t.Error("list row reports secrets for a malformed file")
	}
	out := showHuman(t, p)
	if !strings.Contains(out, "- Secrets: unreadable") {
		t.Errorf("show should report unreadable secrets:\n%s", out)
	}
	if strings.Contains(out, "leak-sentinel") {
		t.Errorf("show leaks file content:\n%s", out)
	}
}
