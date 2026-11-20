//go:build !windows

package testguard

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// lsofTimeout bounds the open-handle probe so a wedged lsof cannot hang
// the test binary's exit.
const lsofTimeout = 10 * time.Second

// openHandles lists files under root this process still holds open, via
// lsof. Windows refuses to delete open files; POSIX does not, so this
// surfaces the same leak there. Returns nil when lsof is unavailable.
func openHandles(root string) []string {
	if _, err := exec.LookPath("lsof"); err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), lsofTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "lsof", "-p", strconv.Itoa(os.Getpid()), "-Fn").Output() // #nosec G204 -- fixed binary and flags; the only variable is this process's own pid

	if err != nil && len(out) == 0 {
		return nil
	}
	var open []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "n") {
			continue
		}
		if name := line[1:]; strings.HasPrefix(name, root+string(os.PathSeparator)) {
			open = append(open, name)
		}
	}
	return open
}
