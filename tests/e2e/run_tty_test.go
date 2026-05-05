package e2e

// Regression coverage for `aps run -- <interactive-cmd>` losing TTY
// pass-through after Wave 2a (T-0460) wrapped child stdout/stderr in
// logging.NewWriter. The wrapper turned *os.File into io.Writer, which
// forced Go's exec to allocate a pipe instead of dup2-ing the parent
// fd. Children that probe TTY-ness (claude code, vim, top, fzf, etc.)
// fell into non-interactive fallbacks.
//
// Fix: stdioWriter only wraps when the destination is NOT a TTY, so
// redaction stays active on pipes/files but interactive subcommands
// inherit the real terminal.

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

// stdoutIsTTYProbe is a tiny shell program that probes stdout's
// TTY-ness (the surface the Wave 2a regression broke) and prints a
// stable token we can grep. We use the file-descriptor form of `tty`:
// `tty <&1` redirects fd1 onto fd0 so the `tty` utility's
// stdin-based ttyname(3) call actually inspects what was stdout.
const stdoutIsTTYProbe = `if [ -t 1 ]; then echo STDOUT_IS_TTY; else echo STDOUT_IS_PIPE; fi`

// TestRunTTYPassthrough_NonTTY documents the redaction-active path.
// When the parent's stdout is a pipe (the default Go-test capture
// mode), the child sees a pipe too, the wrapper engages, redaction
// applies. The probe must report STDOUT_IS_PIPE.
func TestRunTTYPassthrough_NonTTY(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "tty-test-nontty")
	require.NoError(t, err)

	stdout, _, err := runAPS(t, home, "run", "tty-test-nontty", "--",
		"sh", "-c", stdoutIsTTYProbe)
	require.NoError(t, err)
	require.Contains(t, stdout, "STDOUT_IS_PIPE",
		"piped parent stdout should propagate as pipe to child")
}

// TestRunTTYPassthrough_TTY is the regression test. With a real PTY
// allocated for the parent, the child must see a TTY too. Without
// the fix, stdioWriter wraps os.Stdout (a *os.File pointing at the
// pty slave) into an io.Writer, which forces Go's exec to allocate
// a pipe between aps and the child. The child's `test -t 1` then
// returns false → STDOUT_IS_PIPE → fails this assertion.
func TestRunTTYPassthrough_TTY(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creack/pty is unix-only")
	}
	t.Parallel()
	home := t.TempDir()

	_, _, err := runAPS(t, home, "profile", "create", "tty-test-pty")
	require.NoError(t, err)

	// pty.Start allocates a pty pair, sets cmd's stdin/stdout/stderr
	// to the slave end. From aps's perspective, os.Stdout.Fd() is
	// now a real terminal — so stdioWriter must skip the wrap.
	cmd := prepareAPS(t, home, nil, "run", "tty-test-pty", "--",
		"sh", "-c", stdoutIsTTYProbe)
	f, err := pty.Start(cmd)
	require.NoError(t, err, "pty.Start")
	t.Cleanup(func() { _ = f.Close() })

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, f)
		done <- buf.String()
	}()

	waitErr := cmd.Wait()
	out := <-done
	require.NoError(t, waitErr, "aps run failed under PTY: %s", out)

	require.Contains(t, out, "STDOUT_IS_TTY",
		"PTY parent must propagate TTY-ness to child stdout; got: %q", out)
	require.NotContains(t, out, "STDOUT_IS_PIPE",
		"child saw pipe stdout despite PTY parent — stdioWriter wrap "+
			"engaged when it should have passed through")
}
