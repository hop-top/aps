package cli

import (
	"fmt"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [profile] -- [command] [args...]",
	Short: "Run a command in a profile context",
	Args:  cobra.MinimumNArgs(1), // At least profile
	RunE: func(cmd *cobra.Command, args []string) error {
		profileID := args[0]

		// Cobra parses flags before "--". Everything after "--" is in args if we configure it right,
		// OR we have to rely on cmd.ArgsLenAtDash()

		dashIdx := cmd.ArgsLenAtDash()
		if dashIdx == -1 {
			return fmt.Errorf("missing '--' separator\nUsage: aps run <profile> -- <command> [args...]")
		}

		// args[0] is profile
		// args[dashIdx] is the first arg after --? No, Cobra usage is tricky here.
		// If command is `aps run profile -- cmd arg`, args will be `[profile, cmd, arg]` and dashIdx will be 1.

		commandArgs := args[dashIdx:]
		if len(commandArgs) == 0 {
			return fmt.Errorf("no command specified")
		}

		commandName := commandArgs[0]
		commandRest := commandArgs[1:]

		if err := core.RunCommand(profileID, commandName, commandRest); err != nil {
			return fmt.Errorf("running command: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
