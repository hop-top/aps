package device

import (
	"context"
	"encoding/json"
	"fmt"

	"oss-aps-cli/internal/core/device"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	var force bool
	var dryRun bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd.Context(), args[0], force, dryRun, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force stop (SIGKILL)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be stopped without stopping")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runStop(ctx context.Context, name string, force, dryRun, jsonOut bool) error {
	dev, err := device.LoadDevice(name)
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
		fmt.Print("Stop anyway? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if !jsonOut {
		fmt.Printf("Stopping %s... ", name)
	}

	err = defaultManager.StopDevice(ctx, name, force)
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

func renderStopDryRun(dev *device.Device, runtime *device.DeviceRuntime) error {
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
