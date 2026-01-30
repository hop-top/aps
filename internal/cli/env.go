package cli

import (
	"fmt"

	"oss-aps-cli/internal/core/capability"

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

		for _, export := range exports {
			fmt.Println(export)
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
