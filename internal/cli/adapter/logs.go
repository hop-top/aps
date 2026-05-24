package adapter

import (
	"fmt"

	coreadapter "hop.top/aps/internal/core/adapter"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newLogsCmd() *cobra.Command {
	var tail int
	var follow bool
	var since string

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "View device logs",
		Long: `Print captured stdout/stderr lines from the named adapter
device's runtime. Defaults to the last 20 lines; --tail <N>
controls the line count, --follow / -f streams new lines as they
arrive (blocks until interrupted), and --since <duration> (e.g.
1h, 30m) restricts to lines emitted within the given window. The
device must exist in the registry; logs are sourced from the
manager's per-adapter log buffer.

Read-only: no state mutation. Idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(args[0], tail, follow, since)
		},
	}

	cmd.Flags().IntVar(&tail, "tail", 20, "Number of lines to show")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (stream)")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since duration (e.g., 1h, 30m)")

	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runLogs(name string, tail int, follow bool, since string) error {
	_, err := coreadapter.LoadAdapter(name)
	if err != nil {
		return err
	}

	logs, err := defaultManager.GetAdapterLogs(name, tail, follow)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		fmt.Println(dimStyle.Render("No logs available"))
		return nil
	}

	for _, line := range logs {
		fmt.Println(line)
	}

	return nil
}
