package profile_e2e

// E2E coverage for the kit-styled-table rollout (T-0451). The same
// `aps profile list` invocation must:
//   - emit ANSI + lipgloss box-drawing characters when stdout is a TTY
//   - emit plain tabwriter output when stdout is a pipe / non-TTY
//   - return content-identical rows in both modes once ANSI is stripped
//
// We allocate a real pseudo-terminal pair via creack/pty (the same
// pattern used by kit's go/console/cli/tablestyle_e2e_test.go) and
// drain the master end into a buffer.

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/creack/pty"
)

var (
	ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	// boxRunes is the smallest set of lipgloss NormalBorder runes the
	// styled renderer always emits. We accept any one as proof of
	// box-drawing presence.
	boxRunes = []rune{'┌', '┐', '└', '┘', '│', '─', '├', '┤', '┬', '┴'}
)

// stripANSI removes ANSI color/style escape sequences so styled output
// can be diffed against plain tabwriter output. Mirrors the helper in
// kit/console/output/tablestyle_test.go.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// hasBoxRune reports whether s contains at least one of the lipgloss
// NormalBorder box-drawing runes.
func hasBoxRune(s string) bool {
	for _, r := range boxRunes {
		if strings.ContainsRune(s, r) {
			return true
		}
	}
	return false
}

// runAPSPTY runs the compiled aps binary attached to a real pseudo-
// terminal pair so writerIsTTY in kit/output sees an *os.File terminal.
// Returns the bytes drained from the master end.
func runAPSPTY(t *testing.T, home string, args ...string) []byte {
	t.Helper()

	cmd := exec.Command(apsBinary, args...)
	cmd.Env = e2eEnv(home)

	f, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty.Start: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	// Drain master end concurrently — the writer side blocks until the
	// kernel buffer drains.
	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, f)
		done <- buf.Bytes()
	}()

	if err := cmd.Wait(); err != nil {
		t.Fatalf("aps wait: %v\nptyOutput: %s", err, string(<-done))
	}
	_ = f.Close()
	return <-done
}

// e2eEnv mirrors helpers_test.go:runAPS env construction so PTY runs
// see the same isolated HOME the non-PTY tests use.
func e2eEnv(home string) []string {
	override := map[string]bool{
		"HOME":          true,
		"USERPROFILE":   true,
		"XDG_DATA_HOME": true,
		"APS_DATA_PATH": true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if override[key] {
			continue
		}
		env = append(env, e)
	}
	return env
}

// TestProfileList_Styled_TTYAndNonTTYContentIdentity is the headline
// e2e for the rollout. The same `aps profile list` run on a TTY emits
// ANSI + box-drawing while on a pipe it emits plain tabwriter; row
// data is content-identical once ANSI is stripped.
func TestProfileList_Styled_TTYAndNonTTYContentIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creack/pty is unix-only")
	}

	home := t.TempDir()
	writeProfile(t, home, "alpha", `id: alpha
display_name: Alpha Agent
email: alpha@example.com
roles:
  - owner
`)
	writeProfile(t, home, "beta", `id: beta
display_name: Beta Agent
email: beta@example.com
`)

	// Non-TTY path: pipe stdout via the existing runAPS helper.
	plain, stderr, err := runAPS(t, home, "profile", "list")
	if err != nil {
		t.Fatalf("non-TTY profile list: %v\nstderr: %s", err, stderr)
	}
	if ansiRe.MatchString(plain) {
		t.Errorf("non-TTY output leaked ANSI escapes: %q", plain)
	}
	if hasBoxRune(plain) {
		t.Errorf("non-TTY output leaked box-drawing chars: %q", plain)
	}

	// TTY path: PTY pair so writerIsTTY in kit/output returns true.
	ttyBytes := runAPSPTY(t, home, "profile", "list")
	tty := string(ttyBytes)

	if !ansiRe.MatchString(tty) {
		t.Errorf("TTY output missing ANSI escapes: %q", tty)
	}
	if !hasBoxRune(tty) {
		t.Errorf("TTY output missing box-drawing characters: %q", tty)
	}

	// Content identity: both modes must surface the same row data.
	stripped := stripANSI(tty)
	for _, want := range []string{"alpha", "Alpha Agent", "beta", "Beta Agent"} {
		if !strings.Contains(plain, want) {
			t.Errorf("plain output missing %q\nstdout: %s", want, plain)
		}
		if !strings.Contains(stripped, want) {
			t.Errorf("TTY output (stripped) missing %q\nstdout: %s", want, stripped)
		}
	}
	for _, want := range []string{"ID", "DISPLAY NAME"} {
		if !strings.Contains(plain, want) {
			t.Errorf("plain output missing header %q\nstdout: %s", want, plain)
		}
		if !strings.Contains(stripped, want) {
			t.Errorf("TTY output (stripped) missing header %q\nstdout: %s", want, stripped)
		}
	}
}
