package device

import "github.com/spf13/cobra"

func NewDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "device",
		Aliases: []string{"devices"},
		Short:   "Manage devices (messengers, protocols, mobile, desktop)",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newLinkCmd())
	cmd.AddCommand(newUnlinkCmd())

	return cmd
}
