package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"hop.top/aps/internal/cli/confirmtoken"
)

// Every command gated by kit's typed confirmation token must say how to
// get and pass the token in its own help: --yes / --force do not replace
// it, and the refusal is the same with and without a terminal.
func TestDestructiveTokenCommandsDocumentTheTokenFlow(t *testing.T) {
	var gated []string
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if v := c.Annotations["kit/destructive-token"]; v == "required" || v == "true" {
			gated = append(gated, c.CommandPath())
			if !strings.Contains(c.Long, confirmtoken.Help) {
				t.Errorf("%s: Long help lacks the confirmation token paragraph", c.CommandPath())
			}
		}
		for _, sc := range c.Commands() {
			walk(sc)
		}
	}
	walk(rootCmd)
	if len(gated) < 12 {
		t.Fatalf("found only %d token-gated commands (%v); annotation walk is broken", len(gated), gated)
	}
}

// profile delete's --yes only skips the command's own yes/no question.
func TestProfileDeleteYesHelpDoesNotPromiseToReplaceToken(t *testing.T) {
	f := profileDeleteCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("no --yes flag")
	}
	if !strings.Contains(f.Usage, "--confirm-token") {
		t.Errorf("--yes usage %q must say the token is still required", f.Usage)
	}
	if strings.Contains(profileDeleteCmd.Long, "unless --yes is set or stdin is\nnot a TTY") {
		t.Error("Long help still claims --yes or a non-TTY stdin skips confirmation")
	}
}

func TestDocsHelpNamesDataDirNotAgentsDir(t *testing.T) {
	if strings.Contains(docsCmd.Long, "agents-dir") || strings.Contains(docsCmd.Long, "$APS_DATA_PATH/agents") {
		t.Errorf("docs help still names an agents dir:\n%s", docsCmd.Long)
	}
	if !strings.Contains(docsCmd.Long, "<data-dir>/docs") {
		t.Errorf("docs help must name <data-dir>/docs:\n%s", docsCmd.Long)
	}
}

func TestProfileProfileFlagHelpDoesNotPromiseActiveProfile(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("profile")
	if f == nil {
		t.Skip("profile flag not on rootCmd persistent set")
	}
	if strings.Contains(f.Usage, "active profile") {
		t.Errorf("--profile usage %q promises an active profile; none exists", f.Usage)
	}
}
