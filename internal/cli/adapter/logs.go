package adapter

import (
	"fmt"

	coreadapter "hop.top/aps/internal/core/adapter"

	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var tail int
	var follow bool
	var since string

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "View device logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogs(args[0], tail, follow, since)
		},
	}

	cmd.Flags().IntVar(&tail, "tail", 20, "Number of lines to show")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (stream)")
	cmd.Flags().StringVar(&since, "since", "", "Show logs since duration (e.g., 1h, 30m)")

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
