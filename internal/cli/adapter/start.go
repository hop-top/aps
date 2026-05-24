package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	coreadapter "hop.top/aps/internal/core/adapter"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newStartCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a device",
		Long: `Spawn the runtime for the named adapter device under
the manager. The exact mechanics depend on the device's loading
strategy: subprocess strategies fork the executable named by the
manifest, script strategies invoke the script binary, and builtin
strategies attach the in-process handler. Pair with
aps adapter create <name> to mint a device record first, and
aps adapter stop <name> to terminate.

The command prints a "Starting <name>... running (PID <pid>)"
progress line on a TTY, or emits structured runtime state with
--json. On failure it surfaces the error and — for messenger
devices missing a token — hints the corresponding
aps secrets set <NAME>_TOKEN command. The Already-Running case
returns success.

Mutates local runtime state. Idempotent only in the
already-running sense (no double-spawn). Dry-run is opted out
because previewing would have to bisect the spawn-and-wait path
that is the operation itself.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd.Context(), args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — start spawns the adapter's long-running daemon; preview
	// would have to bisect the spawn-and-wait path that's the whole
	// operation.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "start spawns the adapter's long-running daemon; previewing would have to bisect the spawn-and-wait path that is the operation itself."); err != nil {
		panic(err)
	}
	return cmd
}

func runStart(ctx context.Context, name string, jsonOut bool) error {
	dev, err := coreadapter.LoadAdapter(name)
	if err != nil {
		return err
	}

	if !jsonOut {
		fmt.Printf("Starting %s... ", name)
	}

	err = defaultManager.StartAdapter(ctx, name)
	if err != nil {
		if !jsonOut {
			fmt.Println(errorStyle.Render("failed"))
			fmt.Println()
			return renderStartError(dev, err)
		}
		return err
	}

	runtime, _ := defaultManager.GetRuntime(name)

	if jsonOut {
		return renderStartJSON(name, runtime)
	}

	if runtime.PID > 0 {
		fmt.Printf("%s (PID %d)\n", successStyle.Render("running"), runtime.PID)
	} else {
		fmt.Println(successStyle.Render("running"))
	}

	return nil
}

func renderStartError(dev *coreadapter.Adapter, err error) error {
	if coreadapter.IsAdapterAlreadyRunning(err) {
		fmt.Println(warnStyle.Render("already running"))
		return nil
	}

	fmt.Println(errorStyle.Render("failed"))
	fmt.Println()

	if coreadapter.IsAdapterTypeNotImplemented(err) {
		return err
	}

	fmt.Printf("  Error: %s\n", err)

	if dev.Type == coreadapter.AdapterTypeMessenger {
		tokenEnv := fmt.Sprintf("%s_TOKEN", toEnvName(dev.Name))
		fmt.Printf("  Set it: aps secrets set %s \"your-token\"\n", tokenEnv)
	}

	return err
}

func renderStartJSON(name string, runtime *coreadapter.AdapterRuntime) error {
	data := map[string]interface{}{
		"name":  name,
		"state": runtime.State,
	}
	if runtime.PID > 0 {
		data["pid"] = runtime.PID
	}
	if runtime.StartedAt != nil {
		data["started_at"] = runtime.StartedAt
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
