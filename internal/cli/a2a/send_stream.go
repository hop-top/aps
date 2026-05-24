package a2a

import (
	"errors"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func NewSendStreamCmd() *cobra.Command {
	var (
		targetProfile string
		message       string
		taskID        string
	)

	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Send a message with streaming updates (not yet supported)",
		Long:  `Send a message with streaming updates. This feature requires SDK support for streaming.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("send-stream not yet supported by a2a-go SDK v0.3.4")
		},
	}

	cmd.Flags().StringVarP(&targetProfile, "target", "t", "", "Target profile ID")
	cmd.Flags().StringVarP(&message, "message", "m", "", "Message text")
	cmd.Flags().StringVar(&taskID, "task-id", "", "Existing task ID (optional)")

	// T-0648 — kit 0.4 signature annotations. Mirrors `a2a tasks send`:
	// outbound network call; each invocation mints fresh state.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyConditional)
	// T-0656 — same shape as send: minting fresh IDs over JSON-RPC.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "send-stream opens a streaming JSON-RPC call that mints fresh message and task IDs on the peer; previewing would have to fake the stream that the wire call establishes."); err != nil {
		panic(err)
	}

	return cmd
}
