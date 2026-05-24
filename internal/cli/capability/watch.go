package capability

import (
	"fmt"
	"os"

	"hop.top/aps/internal/cli/clinote"
	"hop.top/aps/internal/core/capability"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newWatchCmd() *cobra.Command {
	var name, tool string

	cmd := &cobra.Command{
		Use:   "watch <path> --name <name> | --tool <tool>",
		Short: "Watch an external file (symlink into APS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var target string
			if len(args) > 0 {
				target = args[0]
			}

			if tool != "" {
				if target != "" {
					return fmt.Errorf("cannot specify both path and --tool")
				}
				pattern, err := capability.GetSmartPattern(tool)
				if err != nil {
					return fmt.Errorf("unknown tool '%s'", tool)
				}
				cwd, _ := os.Getwd()
				target = cwd + "/" + pattern.DefaultPath
				if name == "" {
					name = tool
				}
				fmt.Println(dimStyle.Render(fmt.Sprintf(
					"Smart watch: %s -> %s", tool, pattern.DefaultPath)))
			}

			if target == "" {
				return fmt.Errorf("path argument required if --tool not provided")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			if err := capability.Watch(target, name); err != nil {
				return err
			}
			fmt.Println(successStyle.Render("Watching") + " " +
				target + " as " + boldStyle.Render(name))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of capability")
	cmd.Flags().StringVar(&tool, "tool", "",
		"Smart tool name (e.g. windsurf)")
	clinote.AddFlag(cmd) // T-1291

	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteLocal)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — watch installs a long-running filesystem watcher that
	// rewrites symlinks on change; there is no batch boundary on which
	// to scope a preview.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "watch installs a long-running filesystem watcher that rewrites symlinks on every change event; there is no batch boundary on which a preview could be scoped."); err != nil {
		panic(err)
	}
	return cmd
}
