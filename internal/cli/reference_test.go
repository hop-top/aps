package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var helpFlagLine = regexp.MustCompile(`^\s+(?:-[A-Za-z] )?(--[a-z][a-z0-9-]*)\b`)

// binaryHelp returns what `aps <args>` prints on stdout, from a child
// process: dispatch changes the shared command tree in place (commands
// added at dispatch fail the validation other tests run), so it must not
// happen in this process.
func binaryHelp(t *testing.T, args ...string) string {
	t.Helper()
	dataHome := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestExecuteHelperProcess$")
	cmd.Env = append(os.Environ(),
		execHelperEnv+"="+strings.Join(args, " "),
		"XDG_DATA_HOME="+dataHome,
		"XDG_STATE_HOME="+filepath.Join(dataHome, "state"),
		"APS_DATA_PATH="+filepath.Join(dataHome, "aps"),
		"APS_NO_BUS_WARN=1",
		"APS_JOBS_DISABLE=",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("aps %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return string(out)
}

func allFlagNames(cmd *cobra.Command) map[string]bool {
	names := map[string]bool{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) { names["--"+f.Name] = true })
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) { names["--"+f.Name] = true })
	return names
}

// RootCommand is what documentation is generated from. It must expose
// every flag the real binary's help shows for a command, including the
// ones kit attaches at dispatch (--confirm-token, --idempotency-key)
// and the persistent globals.
func TestRootCommandFlagsCoverBinaryHelp(t *testing.T) {
	prepared, err := RootCommand()
	if err != nil {
		t.Fatalf("RootCommand: %v", err)
	}
	cmd, _, err := prepared.Find([]string{"profile", "delete"})
	if err != nil || cmd.Name() != "delete" {
		t.Fatalf("profile delete not found: %v", err)
	}
	have := allFlagNames(cmd)
	// The real dispatch path, as the binary runs it; read the help it
	// renders.
	helpOut := binaryHelp(t, "profile", "delete", "--help")

	seen := 0
	for _, line := range strings.Split(helpOut, "\n") {
		m := helpFlagLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		seen++
		// cobra adds --help lazily; climd lists it for every command.
		if m[1] != "--help" && !have[m[1]] {
			t.Errorf("binary help for `profile delete` shows %s, RootCommand tree lacks it", m[1])
		}
	}
	if seen < 8 {
		t.Fatalf("parsed only %d flags from help; output:\n%s", seen, helpOut)
	}
	for _, want := range []string{"--confirm-token", "--idempotency-key", "--note", "--yes", "--force", "--profile", "--format"} {
		if !have[want] {
			t.Errorf("RootCommand tree lacks %s on `profile delete`", want)
		}
	}
}
