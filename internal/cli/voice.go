package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/voice"
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
		profileID, _ := cmd.Flags().GetString("profile")
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
	voiceStartCmd.Flags().String("profile", "", "Profile ID to use for this voice session")
	voiceStartCmd.Flags().String("channel", "web", "Channel: web | tui | telegram | twilio")
}
