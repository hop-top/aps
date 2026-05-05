package cli

import (
	"fmt"

	"hop.top/aps/internal/core/capability"
	"hop.top/aps/internal/logging"

	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Output environment variables for configured capabilities",
	Long: `Outputs shell commands to export environment variables for all installed capabilities.
Usage: eval $(aps env)`,
	Run: func(cmd *cobra.Command, args []string) {
		exports, err := capability.GenerateEnvExports()
		if err != nil {
			// Print error as a comment so eval doesn't choke
			fmt.Printf("# Error generating envs: %v\n", err)
			return
		}

		// T-0460 — exports may include capability env that resolves to
		// secrets (O1 in docs/cli/redact-inventory.md). Run through the
		// redacting writer so the values are tagged unless --no-redact
		// is set. With redaction on, `eval $(aps env)` exports
		// placeholder values; use --no-redact when shell-eval-friendly
		// output is required and the operator has confirmed the sink
		// is private.
		for _, export := range exports {
			_, _ = logging.Println(export)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
