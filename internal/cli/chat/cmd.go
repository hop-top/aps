package chat

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/core"
	corechat "hop.top/aps/internal/core/chat"
)

func NewCommand() *cobra.Command {
	var opts Options
	cmd := &cobra.Command{
		Use:   "chat <profile-id>[,<profile-id>...]",
		Short: "Chat with a profile-backed assistant",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(cmd, args[0], opts)
		},
	}
	cmd.Flags().StringVar(&opts.Once, "once", "", "Send one prompt, print the assistant response, and exit")
	cmd.Flags().StringVar(&opts.Model, "model", "", "Override the configured chat model")
	cmd.Flags().BoolVar(&opts.NoStream, "no-stream", false, "Disable streaming responses")
	cmd.Flags().StringVar(&opts.Attach, "attach", "", "Attach to an existing chat session")
	cmd.Flags().StringSliceVar(&opts.Invite, "invite", nil, "Additional profile(s) to invite (comma-separated or repeated)")
	cmd.Flags().IntVar(&opts.MaxAutoTurns, "max-auto-turns", corechat.DefaultMaxAutoTurns, "Cap on autonomous turn cycles in :auto mode")
	return cmd
}

func Run(cmd *cobra.Command, primaryRef string, opts Options) error {
	participantIDs, err := ParseParticipants(primaryRef, opts.Invite)
	if err != nil {
		return err
	}

	owner, err := core.LoadProfile(participantIDs[0])
	if err != nil {
		return fmt.Errorf("load profile %s: %w", participantIDs[0], err)
	}

	if len(participantIDs) == 1 {
		engine, err := newEngine(owner)
		if err != nil {
			return err
		}
		sess, err := startOrAttachSession(owner, opts.Attach)
		if err != nil {
			return err
		}
		if opts.Once != "" {
			return runOnce(cmd, engine, sess, opts)
		}
		return runTUI(cmd.Context(), engine, sess, owner, opts)
	}

	profiles := make([]*core.Profile, 0, len(participantIDs))
	for _, id := range participantIDs {
		if id == owner.ID {
			profiles = append(profiles, owner)
			continue
		}
		p, err := core.LoadProfile(id)
		if err != nil {
			return fmt.Errorf("load profile %s: %w", id, err)
		}
		profiles = append(profiles, p)
	}
	participants, err := corechat.NewParticipants(profiles)
	if err != nil {
		return err
	}

	engine, err := newMultiParticipantEngine(context.Background(), owner, participants, core.LoadProfile)
	if err != nil {
		return err
	}
	sess, err := startOrAttachSession(owner, opts.Attach)
	if err != nil {
		return err
	}
	if opts.Once != "" {
		return runOnceMulti(cmd, engine, sess, participants, opts)
	}
	return runTUIMulti(cmd.Context(), engine, sess, owner, participants, opts)
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

// runOnceMulti drives a single round of multi-participant chat: the first
// invited speaker answers the prompt. Round-robin progression across multiple
// turns is exercised in the TUI flow.
func runOnceMulti(cmd *cobra.Command, engine CoreEngine, sess *chatSession, participants []corechat.Participant, opts Options) error {
	prompt := opts.Once
	if err := sess.append(roleUser, prompt); err != nil {
		return err
	}
	policy := corechat.NewRoundRobinPolicy()
	speaker, err := policy.Next(corechat.TurnState{Participants: participants})
	if err != nil {
		return err
	}
	resp, err := engine.Turn(cmd.Context(), TurnRequest{
		SessionID: sess.id,
		ProfileID: speaker.ID,
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
	fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", speaker.DisplayName, resp.Message.Content)
	return nil
}
