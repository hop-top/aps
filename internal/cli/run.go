package cli

import (
	"fmt"
	"os"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [profile] -- [command] [args...]",
	Short: "Run a command in a profile context",
	Args:  cobra.MinimumNArgs(1), // At least profile
	Run: func(cmd *cobra.Command, args []string) {
		profileID := args[0]

		// Cobra parses flags before "--". Everything after "--" is in args if we configure it right,
		// OR we have to rely on cmd.ArgsLenAtDash()

		dashIdx := cmd.ArgsLenAtDash()
		if dashIdx == -1 {
			fmt.Fprintln(os.Stderr, "Error: missing '--' separator")
			fmt.Fprintln(os.Stderr, "Usage: aps run <profile> -- <command> [args...]")
			os.Exit(1)
		}

		// args[0] is profile
		// args[dashIdx] is the first arg after --? No, Cobra usage is tricky here.
		// If command is `aps run profile -- cmd arg`, args will be `[profile, cmd, arg]` and dashIdx will be 1.

		commandArgs := args[dashIdx:]
		if len(commandArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Error: no command specified")
			os.Exit(1)
		}

		commandName := commandArgs[0]
		commandRest := commandArgs[1:]

		if err := core.RunCommand(profileID, commandName, commandRest); err != nil {
			// Pass through exit code if possible?
			// For now, just exit 1 on error
			fmt.Fprintf(os.Stderr, "Error running command: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
