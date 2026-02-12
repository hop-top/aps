package capability

import (
	"fmt"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/capability"
	"oss-aps-cli/internal/styles"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show capability details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args[0])
		},
	}
}

func runShow(name string) error {
	// Check builtin first
	if b, err := capability.GetBuiltin(name); err == nil {
		fmt.Println(headerStyle.Render(b.Name))
		fmt.Println()
		fmt.Println(boldStyle.Render("Kind") + ":        " +
			styles.KindBadge("builtin"))
		fmt.Println(boldStyle.Render("Description") + ": " + b.Description)

		profs, _ := core.ProfilesUsingCapability(name)
		if len(profs) > 0 {
			fmt.Println(boldStyle.Render("Profiles") + ":    " + join(profs))
		} else {
			fmt.Println(boldStyle.Render("Profiles") + ":    " +
				dimStyle.Render("none"))
		}
		return nil
	}

	// External
	cap, err := capability.LoadCapability(name)
	if err != nil {
		return fmt.Errorf("capability '%s' not found", name)
	}

	fmt.Println(headerStyle.Render(cap.Name))
	fmt.Println()
	fmt.Println(boldStyle.Render("Kind") + ":        " +
		styles.KindBadge("external"))
	fmt.Println(boldStyle.Render("Type") + ":        " +
		styles.TypeBadge(string(cap.Type)))
	fmt.Println(boldStyle.Render("Path") + ":        " + cap.Path)

	if cap.Description != "" {
		fmt.Println(boldStyle.Render("Description") + ": " + cap.Description)
	}
	if cap.Source != "" {
		fmt.Println(boldStyle.Render("Source") + ":      " + cap.Source)
	}
	if !cap.InstalledAt.IsZero() {
		fmt.Println(boldStyle.Render("Installed") + ":   " +
			cap.InstalledAt.Format("2006-01-02 15:04"))
	}

	if len(cap.Links) > 0 {
		fmt.Println()
		fmt.Println(boldStyle.Render(
			fmt.Sprintf("Links (%d)", len(cap.Links))))
		for target, source := range cap.Links {
			fmt.Printf("  %s -> %s\n", target, dimStyle.Render(source))
		}
	}

	profs, _ := core.ProfilesUsingCapability(name)
	if len(profs) > 0 {
		fmt.Println()
		fmt.Println(boldStyle.Render("Profiles") + ": " + join(profs))
	}

	return nil
}
