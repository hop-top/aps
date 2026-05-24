package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/voice"
	kitcli "hop.top/kit/go/console/cli"
)

var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "Manage voice sessions and the voice backend service",
}

var voiceServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Control the voice backend service",
}

var voiceServiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the voice backend service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if err := mgr.Start(nil); err != nil {
			return fmt.Errorf("starting voice backend: %w", err)
		}
		fmt.Println("Voice backend started.")
		return nil
	},
}

var voiceServiceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the voice backend service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("stopping voice backend: %w", err)
		}
		fmt.Println("Voice backend stopped.")
		return nil
	},
}

var voiceServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show voice backend service status",
	Run: func(cmd *cobra.Command, args []string) {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if mgr.IsRunning() {
			fmt.Println("running")
		} else {
			fmt.Println("stopped")
		}
	},
}

var voiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a voice session",
	RunE: func(cmd *cobra.Command, args []string) error {
		// --profile is a kit-shipped root-persistent global; read the
		// inherited flag rather than redeclaring locally (T-0648 batch 8
		// local-globals dedup).
		profileID := globals.Profile()
		channel, _ := cmd.Flags().GetString("channel")
		if profileID == "" {
			return fmt.Errorf("--profile is required")
		}
		info, err := voice.RegisterSession(profileID, channel)
		if err != nil {
			return fmt.Errorf("starting voice session: %w", err)
		}
		fmt.Printf("Started voice session %s (profile=%s channel=%s)\n", info.ID, profileID, channel)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(voiceCmd)
	voiceCmd.AddCommand(voiceServiceCmd)
	voiceServiceCmd.AddCommand(voiceServiceStartCmd)
	voiceServiceCmd.AddCommand(voiceServiceStopCmd)
	voiceServiceCmd.AddCommand(voiceServiceStatusCmd)
	voiceCmd.AddCommand(voiceStartCmd)
	// T-0648 batch 8 — local --profile dropped to stop shadowing the
	// root-persistent global. The session reads root.Viper.GetString
	// ("profile") so `aps --profile X voice start` and the future
	// `aps voice start --profile X` (via inherited persistent flag)
	// both resolve to the same source.
	voiceStartCmd.Flags().String("channel", "web", "Channel: web | tui | telegram | twilio")

	// T-0648 — kit signature annotations. service start/stop write the
	// local backend state (start is idempotent: NOP when already up).
	// `voice start` mints a fresh voice session each call (write-local,
	// not idempotent without a key).
	kitcli.SetSideEffect(voiceServiceStartCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(voiceServiceStartCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(voiceServiceStopCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(voiceServiceStopCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(voiceServiceStatusCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(voiceServiceStatusCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(voiceStartCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(voiceStartCmd, kitcli.IdempotencyNo)
}
