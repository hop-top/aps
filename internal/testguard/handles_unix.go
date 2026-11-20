//go:build !windows

package testguard

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// openHandles lists files under root this process still holds open, via
// lsof. Windows refuses to delete open files; POSIX does not, so this
// surfaces the same leak there. Returns nil when lsof is unavailable.
func openHandles(root string) []string {
	bin, err := exec.LookPath("lsof")
	if err != nil {
		return nil
	}
	out, err := exec.Command(bin, "-p", strconv.Itoa(os.Getpid()), "-Fn").Output()
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
