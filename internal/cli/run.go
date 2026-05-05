package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/console/progress"
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

		// T-0463 — structured progress per cli-conventions-with-kit.md
		// §6.5. The user-supplied subprocess is opaque; aps emits an
		// envelope (exec start + exit ok/fail) so agents reading the
		// JSONL stream see a uniform progress contract without aps
		// touching the child's stdio.
		ctx := cmd.Context()
		r := progress.FromContext(ctx)
		r.Emit(ctx, progress.Event{Phase: "exec", Item: commandName})

		if err := core.RunCommand(profileID, commandName, commandRest); err != nil {
			okFalse := false
			r.Emit(ctx, progress.Event{Phase: "exit", Item: commandName, OK: &okFalse})
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return &output.Error{
					Code:     output.CodeGeneric,
					Message:  fmt.Sprintf("running command: %v", err),
					ExitCode: exitErr.ExitCode(),
				}
			}
			return fmt.Errorf("running command: %w", err)
		}
		okTrue := true
		r.Emit(ctx, progress.Event{Phase: "exit", Item: commandName, OK: &okTrue})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
