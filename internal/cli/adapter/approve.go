package adapter

import (
	"encoding/json"
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/cli/globals"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newApproveCmd() *cobra.Command {
	var (
		approveAll bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "approve <device-id>",
		Short: "Approve a pending mobile device",
		Long: `Flip a pending mobile device entry to "approved" in the
profile's adapter registry. The device must have previously
registered via the pairing flow (aps adapter pair) and currently
sit in the pending state — list candidates with aps adapter
pending.

Pass a single <device-id> to approve one entry, or use --all to
approve every pending device under the active profile. The --json
flag emits a structured result; --quiet (inherited from root)
suppresses the success line. --profile is inherited from the root
global and is required.

Mutates the local registry only. Idempotent: re-approving an
already-approved device is a no-op. Dry-run is opted out because
preview would only echo the device ID the user already passed.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceID := ""
			if len(args) > 0 {
				deviceID = args[0]
			}
			if deviceID == "" && !approveAll {
				return fmt.Errorf("provide a device ID or use --all")
			}
			profileID := globals.Profile()
			if profileID == "" {
				return fmt.Errorf("--profile is required")
			}
			return runApprove(deviceID, profileID, approveAll, jsonOutput, globals.Quiet())
		},
	}

	// T-0648 — read --profile and --quiet from kit-managed globals
	// (root.Viper) rather than redeclaring local flags. The locals
	// shadowed the globals on the leaf, tripping the local-globals
	// signature check.
	cmd.Flags().BoolVar(&approveAll, "all", false, "Approve all pending devices")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "JSON output")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — approve is an atomic registry transition; preview would
	// only echo back the device ID that is the command's argument.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "approve is an atomic registry transition that flips the pending device entry to approved; previewing would only echo the device ID the user already passed."); err != nil {
		panic(err)
	}
	return cmd
}

func runApprove(deviceID, profileID string, approveAll, jsonOut, quiet bool) error {
	registry, err := getRegistry()
	if err != nil {
		return err
	}

	if approveAll {
		pending, err := registry.ListPending(profileID)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			if !quiet {
				fmt.Println(dimStyle.Render("  No pending devices."))
			}
			return nil
		}

		approved := 0
		for _, d := range pending {
			if err := registry.ApproveAdapter(d.AdapterID); err != nil {
				fmt.Fprintf(os.Stderr, "  Failed to approve %s: %v\n", d.AdapterID, err)
				continue
			}
			approved++
		}

		if jsonOut {
			out, _ := json.MarshalIndent(map[string]any{
				"approved": approved,
				"profile":  profileID,
			}, "", "  ")
			fmt.Println(string(out))
			return nil
		}
		if !quiet {
			fmt.Printf("  %s Approved %d devices.\n", successStyle.Render("✓"), approved)
		}
		return nil
	}

	if err := registry.ApproveAdapter(deviceID); err != nil {
		return err
	}

	if jsonOut {
		out, _ := json.MarshalIndent(map[string]string{
			"device_id": deviceID,
			"status":    "approved",
		}, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if !quiet {
		fmt.Printf("  %s Device '%s' approved.\n", successStyle.Render("✓"), deviceID)
	}
	return nil
}
