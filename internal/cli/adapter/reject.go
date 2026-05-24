package adapter

import (
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newRejectCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "reject <device-id>",
		Short: "Reject a pending mobile device",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// T-0648 — read --profile and --quiet from kit-managed globals.
			profileID := globals.Profile()
			if profileID == "" {
				return fmt.Errorf("--profile is required")
			}
			return runReject(args[0], profileID, jsonOutput, globals.Quiet())
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runReject(deviceID, profileID string, jsonOut, quiet bool) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	if err := registry.RejectAdapter(deviceID); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"device_id": deviceID,
			"status":    "rejected",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !quiet {
		fmt.Printf("  %s Device '%s' rejected.\n", successStyle.Render("✓"), deviceID)
	}
	return nil
}
