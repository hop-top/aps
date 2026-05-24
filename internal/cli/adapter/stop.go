package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/prompt"
	coreadapter "hop.top/aps/internal/core/adapter"
)

func newStopCmd() *cobra.Command {
	var force bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a device",
		Long: `Terminate the runtime for the named adapter device.
Default behavior sends SIGTERM and waits for graceful shutdown;
--force escalates to SIGKILL. When the device is linked to one
or more profiles, the command prints the linked-profile list and
prompts for confirmation before stopping (suppressed under --json
or --force).

--json emits the structured outcome instead of the human
"Stopping <name>... stopped" progress line. --dry-run (root
global) prints what would happen — type, current state, PID, and
the profiles that would be unaffected — without sending any
signal.

Mutates local runtime state. Idempotent in the already-stopped
sense; force-stopping a healthy adapter is destructive only for
the in-flight work the device was handling, not for the device
record itself (use aps adapter delete to remove the record).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0648 — read --dry-run from kit-managed global.
			return runStop(cmd.Context(), args[0], force, globals.DryRun(), jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force stop (SIGKILL)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runStop(ctx context.Context, name string, force, dryRun, jsonOut bool) error {
	dev, err := coreadapter.LoadAdapter(name)
	if err != nil {
		return err
	}

	runtime, _ := defaultManager.GetRuntime(name)

	if dryRun {
		return renderStopDryRun(dev, runtime)
	}

	linkedCount := len(dev.LinkedTo)
	if linkedCount > 0 && !force && !jsonOut {
		fmt.Printf("Warning: %s is linked to %d profile(s):\n", name, linkedCount)
		for _, p := range dev.LinkedTo {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println()

		confirmed, err := prompt.Confirm("Stop anyway?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if !jsonOut {
		fmt.Printf("Stopping %s... ", name)
	}

	err = defaultManager.StopAdapter(ctx, name, force)
	if err != nil {
		if !jsonOut {
			fmt.Println(errorStyle.Render("failed"))
		}
		return err
	}

	if jsonOut {
		return renderStopJSON(name)
	}

	fmt.Println(successStyle.Render("stopped"))
	return nil
}

func renderStopDryRun(dev *coreadapter.Adapter, runtime *coreadapter.AdapterRuntime) error {
	fmt.Printf("Dry run: stopping %s\n\n", dev.Name)
	fmt.Printf("  Type:       %s\n", dev.Type)
	fmt.Printf("  State:      %s\n", runtime.State)
	if runtime.PID > 0 {
		fmt.Printf("  PID:        %d (would send SIGTERM)\n", runtime.PID)
	}
	if len(dev.LinkedTo) > 0 {
		fmt.Printf("  Profiles:   %s (would be unaffected)\n", join(dev.LinkedTo))
	}
	fmt.Println()
	fmt.Println("No changes made. Remove --dry-run to stop.")
	return nil
}

func renderStopJSON(name string) error {
	data := map[string]interface{}{
		"name":  name,
		"state": "stopped",
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
