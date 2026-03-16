package cli

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if err := mgr.Start(nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Voice backend started.")
	},
}

var voiceServiceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the voice backend service",
	Run: func(cmd *cobra.Command, args []string) {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if err := mgr.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Voice backend stopped.")
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

var voiceSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage active voice sessions",
}

var voiceSessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active voice sessions",
	Run: func(cmd *cobra.Command, args []string) {
		sm := voice.NewSessionManager()
		sessions := sm.List()
		if len(sessions) == 0 {
			fmt.Println("No active voice sessions.")
			return
		}
		for _, s := range sessions {
			fmt.Printf("%s  profile=%-20s  channel=%-10s  state=%s\n",
				s.ID, s.ProfileID, s.ChannelType, s.State)
		}
	},
}

var voiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a voice session",
	Run: func(cmd *cobra.Command, args []string) {
		profileID, _ := cmd.Flags().GetString("profile")
		channel, _ := cmd.Flags().GetString("channel")
		fmt.Printf("Starting voice session: profile=%s channel=%s\n", profileID, channel)
		// TODO: wire up orchestrator
	},
}

func init() {
	rootCmd.AddCommand(voiceCmd)
	voiceCmd.AddCommand(voiceServiceCmd)
	voiceServiceCmd.AddCommand(voiceServiceStartCmd)
	voiceServiceCmd.AddCommand(voiceServiceStopCmd)
	voiceServiceCmd.AddCommand(voiceServiceStatusCmd)
	voiceCmd.AddCommand(voiceSessionCmd)
	voiceSessionCmd.AddCommand(voiceSessionListCmd)
	voiceCmd.AddCommand(voiceStartCmd)
	voiceStartCmd.Flags().String("profile", "", "Profile ID to use for this voice session")
	voiceStartCmd.Flags().String("channel", "web", "Channel: web | tui | telegram | twilio")
}
