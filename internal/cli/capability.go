package cli

import (
	"fmt"
	"os"

	"oss-aps-cli/internal/core/capability"

	"github.com/spf13/cobra"
)

var capabilityCmd = &cobra.Command{
	Use:     "capability",
	Aliases: []string{"cap"},
	Short:   "Manage capabilities (tools, configs, dotfiles)",
}

var capInstallCmd = &cobra.Command{
	Use:   "install <source> --name <name>",
	Short: "Install a capability from a source directory or URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			fmt.Println("Error: --name is required")
			os.Exit(1)
		}

		if err := capability.Install(name, source); err != nil {
			fmt.Printf("Error installing capability: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Capability '%s' installed successfully.\n", name)
	},
}

var capLinkCmd = &cobra.Command{
	Use:   "link <name> [--target <path>]",
	Short: "Symlink a capability to a target path (Outbound Link)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		target, _ := cmd.Flags().GetString("target")

		// Smart Linking logic
		if target == "" {
			// Check if name matches a Smart Pattern
			if pattern, err := capability.GetSmartPattern(name); err == nil {
				target = pattern.ToolName
			} else {
				fmt.Println("Error: --target is required unless using a Smart Pattern name")
				os.Exit(1)
			}
		}

		if err := capability.Link(name, target); err != nil {
			fmt.Printf("Error linking capability: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Capability '%s' linked to '%s'.\n", name, target)
	},
}

var capDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a capability",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if err := capability.Delete(name); err != nil {
			fmt.Printf("Error deleting capability: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Capability '%s' deleted.\n", name)
	},
}

var capAdoptCmd = &cobra.Command{
	Use:   "adopt <path> --name <name>",
	Short: "Adopt an existing file/dir (Move to APS and symlink back)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		target := args[0]
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			fmt.Println("Error: --name is required")
			os.Exit(1)
		}

		if err := capability.Adopt(target, name); err != nil {
			fmt.Printf("Error adopting capability: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Adopted '%s' as capability '%s'.\n", target, name)
	},
}

var capWatchCmd = &cobra.Command{
	Use:   "watch <path> --name <name> | --tool <tool>",
	Short: "Watch an external file (Symlink into APS)",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		tool, _ := cmd.Flags().GetString("tool")

		var target string
		if len(args) > 0 {
			target = args[0]
		}

		// Handle "Smart Watch" --tool <tool>
		if tool != "" {
			if target != "" {
				fmt.Println("Error: cannot specify both path and --tool")
				os.Exit(1)
			}

			pattern, err := capability.GetSmartPattern(tool)
			if err != nil {
				fmt.Printf("Error: unknown tool '%s'\n", tool)
				os.Exit(1)
			}

			// Resolve target relative to CWD
			cwd, _ := os.Getwd()
			target = fmt.Sprintf("%s/%s", cwd, pattern.DefaultPath)

			if name == "" {
				name = tool
			}
		}

		if target == "" {
			fmt.Println("Error: path argument required if --tool not provided")
			os.Exit(1)
		}
		if name == "" {
			fmt.Println("Error: --name required")
			os.Exit(1)
		}

		if err := capability.Watch(target, name); err != nil {
			fmt.Printf("Error watching capability: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Watching '%s' as capability '%s'.\n", target, name)
	},
}

func init() {
	rootCmd.AddCommand(capabilityCmd)
	capabilityCmd.AddCommand(capListCmd)
	capabilityCmd.AddCommand(capInstallCmd)
	capabilityCmd.AddCommand(capLinkCmd)
	capabilityCmd.AddCommand(capDeleteCmd)
	capabilityCmd.AddCommand(capAdoptCmd)
	capabilityCmd.AddCommand(capWatchCmd)

	capInstallCmd.Flags().String("name", "", "Name of the capability")
	capLinkCmd.Flags().String("target", "", "Target path for symlink")
	capAdoptCmd.Flags().String("name", "", "Name of capability")
	capWatchCmd.Flags().String("name", "", "Name of capability")
	capWatchCmd.Flags().String("tool", "", "Smart tool name (e.g. windsurf)")
}

var capListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed capabilities",
	Run: func(cmd *cobra.Command, args []string) {
		caps, err := capability.List()
		if err != nil {
			fmt.Printf("Error listing capabilities: %v\n", err)
			os.Exit(1)
		}

		if len(caps) == 0 {
			fmt.Println("No capabilities installed.")
			return
		}

		fmt.Printf("%-20s %-10s %-40s\n", "NAME", "TYPE", "PATH")
		fmt.Println("---------------------------------------------------------------------------")
		for _, cap := range caps {
			fmt.Printf("%-20s %-10s %-40s\n", cap.Name, cap.Type, cap.Path)
		}
	},
}
