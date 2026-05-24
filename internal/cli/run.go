package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
	"hop.top/kit/go/console/output"
	"hop.top/kit/go/console/progress"
)

var (
	runEnvOverrides []string
	runEnvFiles     []string
)

var runCmd = &cobra.Command{
	Use:   "run [profile] -- [command] [args...]",
	Short: "Run a command in a profile context",
	Long: `Spawn an external command under the named profile's resolved
environment (secrets, bundle env vars, profile-scoped settings).
The "--" separator is required; everything before it is parsed by
aps, everything after is the user command and its arguments.

A structured progress envelope (exec start + exit) is emitted on
stdout per kit's JSONL progress contract so agents that read the
stream observe a uniform lifecycle without aps touching the child
process's stdio. The child's exit code is propagated as the aps
exit code on failure.

Mutating: spawns an opaque subprocess whose side effects aps cannot
enumerate; idempotency is conditional on the spawned command.
--dry-run is opted out because previewing a third-party binary
without invoking it is impossible.`,
	Args: cobra.MinimumNArgs(1), // At least profile
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

		overrides, err := core.BuildOverrideEnv(runEnvFiles, runEnvOverrides)
		if err != nil {
			return &output.Error{
				Code:     output.CodeGeneric,
				Message:  err.Error(),
				ExitCode: 2,
			}
		}

		// T-0463 — structured progress per cli-conventions-with-kit.md
		// §6.5. The user-supplied subprocess is opaque; aps emits an
		// envelope (exec start + exit ok/fail) so agents reading the
		// JSONL stream see a uniform progress contract without aps
		// touching the child's stdio.
		ctx := cmd.Context()
		r := progress.FromContext(ctx)
		r.Emit(ctx, progress.Event{Phase: "exec", Item: commandName})

		if err := core.RunCommand(profileID, commandName, commandRest, overrides); err != nil {
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
	// T-0648 — kit signature annotations. `run` executes an opaque
	// subprocess in a profile context; the agent budget treats it as
	// a local state-mutating verb. Idempotency depends on the spawned
	// command, so the kit-level tag is conditional.
	kitcli.SetSideEffect(runCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(runCmd, kitcli.IdempotencyConditional)
	// T-0679 — depth-1 runnable leaf under `aps`; mark intentional so the
	// shape validator (kit/top-level-verb) accepts it under EnforceValidate.
	kitcli.SetTopLevelVerb(runCmd)
	// T-0656 — run spawns an opaque subprocess under the named profile;
	// the spawned process owns its own side effects and aps cannot
	// preview a third-party binary's behavior.
	kitcli.OptOutDryRun(runCmd)
	if err := kitcli.SetDryRunRationale(runCmd, "run spawns an opaque subprocess in the named profile context; the spawned process owns its side effects and aps cannot preview a third-party binary without invoking it."); err != nil {
		panic(err)
	}
	runCmd.Flags().StringArrayVar(&runEnvOverrides, "env", nil, "Set KEY=VALUE in the child env; repeatable; later --env wins for duplicate keys")
	runCmd.Flags().StringSliceVar(&runEnvFiles, "env-file", nil, "Read KEY=VALUE entries from a dotenv-style file; repeatable; later files win for duplicate keys")
	rootCmd.AddCommand(runCmd)
}
