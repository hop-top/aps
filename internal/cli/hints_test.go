package cli

import (
	"bytes"
	"image/color"
	"io"
	"path/filepath"
	"testing"

	"hop.top/kit/go/console/output"

	"hop.top/aps/internal/cli/service"
)

// TestHints_ProfileList asserts the post-command hint registered in
// registerHints is wired on root.Hints under the namespaced key
// "profile list" (T-0346).
func TestHints_ProfileList(t *testing.T) {
	hints := root.Hints.Lookup("profile list")
	if len(hints) == 0 {
		t.Fatal("expected at least one hint registered for 'profile list'")
	}
	want := "Run `aps profile show <id>` for details."
	if hints[0].Message != want {
		t.Errorf("hint message = %q, want %q", hints[0].Message, want)
	}
}

// TestHints_NativeTransportNotice asserts the uma migration notice is a
// next-step hint on the commands that name a native SMS or WhatsApp service,
// that it is conditional, and that kit's hint renderer keeps it out of
// machine-readable and piped output.
func TestHints_NativeTransportNotice(t *testing.T) {
	service.ResetNativeTransportNotice()
	t.Cleanup(service.ResetNativeTransportNotice)

	for _, key := range []string{"service add", "service show", "service status"} {
		hints := root.Hints.Lookup(key)
		if len(hints) != 1 {
			t.Fatalf("hints for %q = %d, want 1", key, len(hints))
		}
		if hints[0].Message != service.NativeTransportNotice {
			t.Errorf("%q hint message = %q", key, hints[0].Message)
		}
		if len(output.Active(hints)) != 0 {
			t.Errorf("%q hint is active before any native service was seen", key)
		}
	}
	for _, key := range []string{"service test", "service start", "service routes", "profile list"} {
		for _, h := range root.Hints.Lookup(key) {
			if h.Message == service.NativeTransportNotice {
				t.Errorf("%q must not carry the migration notice", key)
			}
		}
	}

	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	cmd := service.NewServiceCmd()
	cmd.PersistentFlags().String("profile", "", "")
	cmd.PersistentFlags().Bool("dry-run", false, "")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{
		"add", "sms-main", "--dry-run", "--type", "sms", "--profile", "assistant",
		"--provider", "twilio", "--from", "+15559870002", "--default-action", "triage",
		"--env", "TWILIO_ACCOUNT_SID=secret:TWILIO_ACCOUNT_SID",
		"--env", "TWILIO_AUTH_TOKEN=secret:TWILIO_AUTH_TOKEN",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("service add: %v", err)
	}

	hints := root.Hints.Lookup("service add")
	if len(output.Active(hints)) != 1 {
		t.Fatalf("hint not active after a native service was seen")
	}
	for _, format := range []output.Format{output.JSON, output.YAML, output.Table} {
		var buf bytes.Buffer
		output.RenderHints(&buf, hints, format, root.Viper, color.Black)
		if buf.Len() != 0 {
			t.Errorf("format %q on a non-terminal wrote %q", format, buf.String())
		}
	}
}
