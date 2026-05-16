package chat

import (
	"fmt"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/core"
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
	return cmd
}

func Run(cmd *cobra.Command, profileID string, opts Options) error {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("load profile %s: %w", profileID, err)
	}
	engine, err := newEngine(profile)
	if err != nil {
		return err
	}
	sess, err := startOrAttachSession(profile, opts.Attach)
	if err != nil {
		return err
	}
	if opts.Once != "" {
		return runOnce(cmd, engine, sess, opts)
	}
	return runTUI(cmd.Context(), engine, sess, profile, opts)
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
