package device

import (
	"context"
	"encoding/json"
	"fmt"

	"oss-aps-cli/internal/core/device"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd.Context(), args[0], jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")

	return cmd
}

func runStart(ctx context.Context, name string, jsonOut bool) error {
	dev, err := device.LoadDevice(name)
	if err != nil {
		return err
	}

	if !jsonOut {
		fmt.Printf("Starting %s... ", name)
	}

	err = defaultManager.StartDevice(ctx, name)
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

func renderStartError(dev *device.Device, err error) error {
	if device.IsDeviceAlreadyRunning(err) {
		fmt.Println(warnStyle.Render("already running"))
		return nil
	}

	fmt.Println(errorStyle.Render("failed"))
	fmt.Println()

	if device.IsDeviceTypeNotImplemented(err) {
		return err
	}

	fmt.Printf("  Error: %s\n", err)

	if dev.Type == device.DeviceTypeMessenger {
		tokenEnv := fmt.Sprintf("%s_TOKEN", toEnvName(dev.Name))
		fmt.Printf("  Set it: aps secrets set %s \"your-token\"\n", tokenEnv)
	}

	return err
}

func renderStartJSON(name string, runtime *device.DeviceRuntime) error {
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
