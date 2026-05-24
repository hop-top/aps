package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"

	"hop.top/aps/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build metadata.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		short, _ := cmd.Flags().GetBool("short")
		info := version.Get()

		if short {
			fmt.Println(info.Version)
			return nil
		}

		format := root.Viper.GetString("format")
		if format == output.Table {
			fmt.Println(info.String())
			return nil
		}
		return output.Render(os.Stdout, format, info)
	},
}

func init() {
	versionCmd.Flags().Bool("short", false, "Output only the version number")
	// T-0648 — kit signature annotations. `version` prints build metadata
	// only; pure read, deterministically identical across runs.
	kitcli.SetSideEffect(versionCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(versionCmd, kitcli.IdempotencyYes)
	// T-0679 — depth-1 runnable leaf under `aps`; mark intentional so the
	// shape validator (kit/top-level-verb) accepts it under EnforceValidate.
	kitcli.SetTopLevelVerb(versionCmd)
	rootCmd.AddCommand(versionCmd)
}
