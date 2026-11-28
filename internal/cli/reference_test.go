package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var helpFlagLine = regexp.MustCompile(`^\s+(?:-[A-Za-z] )?(--[a-z][a-z0-9-]*)\b`)

// captureStdout runs fn with os.Stdout redirected and returns what it
// wrote; the styled help renderer writes there.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()
	func() {
		defer func() { os.Stdout = orig }()
		fn()
	}()
	_ = w.Close()
	return <-done
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
	// Documentation tree first: dispatch below makes fang add its own
	// commands, which the validation Prepare runs would reject.
	prepared, err := RootCommand()
	if err != nil {
		t.Fatalf("RootCommand: %v", err)
	}
	// Then the real dispatch path, as the binary runs it; read the help
	// it renders.
	root.SetArgs([]string{"profile", "delete", "--help"})
	helpOut := captureStdout(t, func() { _ = root.Execute(context.Background()) })

	cmd, _, err := prepared.Find([]string{"profile", "delete"})
	if err != nil || cmd.Name() != "delete" {
		t.Fatalf("profile delete not found: %v", err)
	}
	have := allFlagNames(cmd)

	seen := 0
	for _, line := range strings.Split(helpOut, "\n") {
		m := helpFlagLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		seen++
		if !have[m[1]] {
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
