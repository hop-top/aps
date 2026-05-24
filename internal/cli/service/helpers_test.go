package service

import "github.com/spf13/cobra"

// newTestServiceCmd returns a fake aps root with --profile, --workspace,
// --dry-run registered as persistent flags and the service group mounted
// underneath. Tests should call cmd.SetArgs with the "service" prefix
// already included (or pass via the args directly since the fake root
// has the service group as its only child and TraverseChildren=true).
//
// T-0648 — service leaves no longer redeclare these globals locally;
// production gets them via the kit/cli Globals config. Tests previously
// constructed NewServiceCmd() standalone and passed --profile etc as
// local flags. This helper restores that ergonomic in tests without
// re-introducing the local-globals signature violation.
//
// The returned command is the fake root. Call SetArgs/SetOut on it and
// pass the subcommand path explicitly, e.g.
//
//	cmd := newTestServiceCmd()
//	cmd.SetArgs([]string{"service", "add", "foo", "--profile", "p"})
//	cmd.Execute()
//
// or — to preserve the existing test style which doesn't prefix
// "service" — use newTestServiceSubcmd which returns a fresh root
// whose Use is "service" and which directly carries the service
// group's children. See below.
func newTestServiceCmd() *cobra.Command {
	root := &cobra.Command{Use: "service"}
	root.PersistentFlags().String("profile", "", "profile id")
	root.PersistentFlags().String("workspace", "", "workspace id")
	root.PersistentFlags().Bool("dry-run", false, "preview side effects")
	// Mount the real service group's children directly on the test
	// root so SetArgs(["add", ...]) finds them without a "service"
	// prefix — matches the pre-T-0648 ergonomics where NewServiceCmd
	// was the de-facto root.
	svc := NewServiceCmd()
	for _, c := range svc.Commands() {
		root.AddCommand(c)
	}
	return root
}
