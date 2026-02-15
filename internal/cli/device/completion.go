package device

import (
	"oss-aps-cli/internal/core/device"

	"github.com/spf13/cobra"
)

func completeDeviceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	devices, err := device.ListDevices(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeStoppedDeviceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	devices, err := device.ListDevices(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var names []string
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeRunningDeviceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	devices, err := device.ListDevices(nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	mgr := device.NewManager()
	running := mgr.ListRunningDevices()

	var names []string
	for _, d := range devices {
		for _, r := range running {
			if d.Name == r {
				names = append(names, d.Name)
				break
			}
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeDeviceTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var types []string
	for _, meta := range device.DeviceTypes {
		types = append(types, string(meta.Type))
	}
	return types, cobra.ShellCompDirectiveNoFileComp
}

func completeImplementedDeviceTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var types []string
	for _, meta := range device.DeviceTypes {
		if meta.Implemented {
			types = append(types, string(meta.Type))
		}
	}
	return types, cobra.ShellCompDirectiveNoFileComp
}

func completeStrategies(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		string(device.StrategySubprocess),
		string(device.StrategyScript),
		string(device.StrategyBuiltin),
	}, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	listCmd := newListCmd()
	listCmd.RegisterFlagCompletionFunc("type", completeDeviceTypes)

	createCmd := newCreateCmd()
	createCmd.RegisterFlagCompletionFunc("type", completeImplementedDeviceTypes)
	createCmd.RegisterFlagCompletionFunc("strategy", completeStrategies)
}
