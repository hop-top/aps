package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"oss-aps-cli/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		short, _ := cmd.Flags().GetBool("short")

		info := version.Get()

		if short {
			fmt.Println(info.Version)
			return
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return
		}

		fmt.Println(info.String())
	},
}

func init() {
	versionCmd.Flags().Bool("json", false, "Output version info as JSON")
	versionCmd.Flags().Bool("short", false, "Output only the version number")
	rootCmd.AddCommand(versionCmd)
}
