package cli

import (
	"fmt"
	"os"

	"hop.top/aps/internal/core"

	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Generate shell aliases for profiles",
	Long: `Generate shell aliases for all available profiles.
Add the following to your shell configuration file:

  eval "$(aps alias)"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := core.ListProfiles()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		// Detect shell to decide alias format
		shellPath := core.DetectShell()
		shellName := core.GetShellName(shellPath)

		for _, p := range profiles {
			// Check for conflicts
			if core.IsCommandAvailable(p) {
				fmt.Fprintf(os.Stderr, "WARNING: Skipping alias for '%s' because a command with that name already exists in PATH.\n", p)
				continue
			}

			// Format alias based on shell
			// For now, standard POSIX alias works for zsh/bash/fish
			// alias <p>='aps <p>'
			switch shellName {
			case "powershell", "pwsh":
				fmt.Printf("function %s { aps %s @args }\n", p, p)
			default:
				fmt.Printf("alias %s='aps %s'\n", p, p)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aliasCmd)
}
