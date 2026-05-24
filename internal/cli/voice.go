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
	Long: `Start the long-running voice backend daemon that hosts aps
voice sessions. The daemon owns the audio-capture pipeline and
the realtime STT/TTS provider connections; aps voice start
(per-session command) talks to this daemon.

Idempotent: starting an already-running backend is a no-op
(NewBackendManager().Start returns nil when the process is already
up). --dry-run is opted out because previewing the spawn would
have to bisect the spawn-and-wait path that defines the
operation. Pair with aps voice service stop to terminate and aps
voice service status to inspect.`,
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
	Long: `Stop the voice backend daemon by sending SIGTERM and waiting
for graceful cleanup. Any active aps voice sessions are torn down
when the daemon exits; new aps voice start calls will fail until
aps voice service start is run again.

Idempotent: stopping an already-stopped backend is a no-op.
--dry-run is opted out because previewing would have to fake the
OS signal path that defines the operation.`,
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
	Use:   statusCmdName,
	Short: "Show voice backend service status",
	Long: `Print the voice backend daemon's running state: "running"
if the manager reports the process is up, "stopped" otherwise.
Used as a quick liveness probe before invoking aps voice start.

Read-only: no daemon state mutation. Idempotent.`,
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
	Long: `Register a fresh voice session against the running voice
backend daemon and print the assigned session id. The session
binds to the named profile (via the inherited --profile global,
required) and the channel selected by --channel (web | tui |
telegram | twilio; default web).

Mints a new session record per call (not idempotent). The backend
daemon must already be running (see aps voice service start);
otherwise this call fails with a backend-unreachable error.
--dry-run is opted out because previewing would have to fake the
audio-stream handshake that defines the session.`,
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
	// T-0656 — service start spawns the long-running voice backend
	// daemon; there is no batch boundary on which a preview could be
	// scoped.
	kitcli.OptOutDryRun(voiceServiceStartCmd)
	if err := kitcli.SetDryRunRationale(voiceServiceStartCmd, "service start spawns the long-running voice backend daemon; previewing would have to bisect the spawn-and-wait path that is the operation itself."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(voiceServiceStopCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(voiceServiceStopCmd, kitcli.IdempotencyYes)
	// T-0656 — service stop sends SIGTERM and waits for cleanup;
	// previewing would have to fake the OS signal path.
	kitcli.OptOutDryRun(voiceServiceStopCmd)
	if err := kitcli.SetDryRunRationale(voiceServiceStopCmd, "service stop sends SIGTERM to the voice backend daemon and waits for cleanup; previewing would have to fake the OS signal path that defines the operation."); err != nil {
		panic(err)
	}
	kitcli.SetSideEffect(voiceServiceStatusCmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(voiceServiceStatusCmd, kitcli.IdempotencyYes)
	kitcli.SetSideEffect(voiceStartCmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(voiceStartCmd, kitcli.IdempotencyNo)
	// T-0656 — start opens a fresh interactive voice session against
	// the backend; there is no batch boundary on which a preview could
	// be scoped.
	kitcli.OptOutDryRun(voiceStartCmd)
	if err := kitcli.SetDryRunRationale(voiceStartCmd, "start opens a fresh interactive voice session against the backend daemon; previewing would have to fake the audio-stream handshake that defines the session."); err != nil {
		panic(err)
	}
}
