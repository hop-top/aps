package cli

import (
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

// TestEnforceGuidanceCoverage is the regression net for T-0655.
//
// Walks the rootCmd tree the same way kit/cli's checkConfigurableGates
// does (see kit cli.go:1053) and asserts every runnable, non-builtin
// leaf carries kit/examples; every non-read leaf additionally carries
// kit/next-steps. The aps Config sets EnforceGuidance=true so any leaf
// added without these annotations would fail Execute() once
// DisableValidate flips back to false in T-0657. This test fails ahead
// of CI so the contributor adding the leaf is named directly.
//
// Annotations come from internal/cli/zz_guidance_annotations.go's
// init-time pass keyed on CommandPath().
func TestEnforceGuidanceCoverage(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}

	var missingEx, missingNS []string
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if isBuiltinCmd(sub) {
				continue
			}
			if sub.HasSubCommands() {
				walk(sub)
				continue
			}
			if !sub.Runnable() {
				continue
			}
			se, _ := kitcli.GetSideEffect(sub)
			if _, ok := kitcli.GetExamples(sub); !ok {
				missingEx = append(missingEx, sub.CommandPath())
			}
			if se != kitcli.SideEffectRead {
				if _, ok := kitcli.GetNextSteps(sub); !ok {
					missingNS = append(missingNS, sub.CommandPath())
				}
			}
		}
	}
	walk(rootCmd)

	sort.Strings(missingEx)
	sort.Strings(missingNS)
	if len(missingEx) > 0 {
		t.Errorf("missing kit/examples on %d leaves:\n  %s",
			len(missingEx), strings.Join(missingEx, "\n  "))
	}
	if len(missingNS) > 0 {
		t.Errorf("missing kit/next-steps on %d non-read leaves:\n  %s",
			len(missingNS), strings.Join(missingNS, "\n  "))
	}
}
