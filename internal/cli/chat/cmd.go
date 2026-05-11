package chat

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/core"
	corechat "hop.top/aps/internal/core/chat"
)

func NewCommand() *cobra.Command {
	var opts Options
	cmd := &cobra.Command{
		Use:   "chat <profile-id>",
		Short: "Chat with a profile-backed assistant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd, args[0], opts)
		},
	}
	cmd.Flags().StringVar(&opts.Once, "once", "", "Send one prompt, print the assistant response, and exit")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override the configured chat model")
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "Disable streaming responses")
	cmd.Flags().StringVar(&opts.Attach, "attach", "", "Attach to an existing chat session")
	cmd.Flags().StringSliceVar(&opts.Invite, "invite", nil, "Invite additional profile IDs (comma-separated or repeatable)")
	cmd.Flags().IntVar(&opts.MaxAutoTurns, "max-auto-turns", corechat.DefaultMaxAutoTurns, "Maximum autonomous turns before returning control to the human")
	return cmd
}

func Run(cmd *cobra.Command, profileID string, opts Options) error {
	participantIDs, err := ParseParticipants(profileID, opts.Invite)
	if err != nil {
		return err
	}
	profiles, err := loadProfiles(participantIDs)
	if err != nil {
		return err
	}
	activeProfile := profiles[0]
	systemPrompt, err := systemPromptForProfiles(profiles, activeProfile.ID)
	if err != nil {
		return err
	}
	engine, err := newEngine(cmd.Context(), activeProfile, opts, systemPrompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	sess, err := startOrAttachSession(activeProfile, opts.Attach)
	if err != nil {
		return err
	}
	if opts.Once != "" {
		return runOnce(cmd, engine, sess, opts)
	}
	return runTUI(cmd.Context(), engine, sess, activeProfile, opts)
}

func runOnce(cmd *cobra.Command, engine CoreEngine, sess *chatSession, opts Options) error {
	prompt := opts.Once
	if err := sess.append(roleUser, prompt); err != nil {
		return err
	}
	resp, err := engine.Turn(cmd.Context(), TurnRequest{
		SessionID: sess.id,
		ProfileID: sess.profileID,
		Prompt:    prompt,
		Model:     opts.Model,
		NoStream:  opts.NoStream,
		History:   sess.messages,
	})
	if err != nil {
		return err
	}
	if err := sess.append(resp.Message.Role, resp.Message.Content); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), resp.Message.Content)
	return nil
}

func loadProfiles(ids []string) ([]*core.Profile, error) {
	profiles := make([]*core.Profile, 0, len(ids))
	for _, id := range ids {
		profile, err := core.LoadProfile(id)
		if err != nil {
			return nil, fmt.Errorf("load profile %s: %w", id, err)
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func systemPromptForProfiles(profiles []*core.Profile, activeID string) (string, error) {
	if len(profiles) <= 1 {
		return "", nil
	}
	participants, err := corechat.NewParticipants(profiles)
	if err != nil {
		return "", err
	}
	active, err := corechat.NewRoundRobinPolicy().Next(corechat.TurnState{
		Participants:  participants,
		LastSpeakerID: "",
	})
	if err != nil {
		return "", err
	}
	if activeID != "" {
		active.ID = activeID
	}
	return corechat.ComposeSystemPrompt(participants, active.ID)
}
