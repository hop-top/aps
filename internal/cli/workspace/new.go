package workspace

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	collab "hop.top/aps/internal/core/collaboration"
)

// NewNewCmd creates the "collab new" command.
func NewNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a collaboration workspace",
		Long: `Create a new multi-agent collaboration workspace.

A workspace provides a shared context where agents coordinate,
exchange tasks, and resolve conflicts.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			profile, err := resolveProfile(cmd)
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			policy, _ := cmd.Flags().GetString("policy")

			// Interactive prompts when flags not provided
			if description == "" {
				if err := huh.NewInput().
					Title("Description (optional)").
					Value(&description).
					Run(); err != nil {
					return err
				}
			}

			if !cmd.Flags().Changed("policy") {
				if err := huh.NewSelect[string]().
					Title("Conflict resolution policy").
					Options(
						huh.NewOption("priority", "priority"),
						huh.NewOption("consensus", "consensus"),
						huh.NewOption("keep-last", "keep-last"),
						huh.NewOption("keep-first", "keep-first"),
						huh.NewOption("voting", "voting"),
					).
					Value(&policy).
					Run(); err != nil {
					return err
				}
			}

			mgr, err := getManager()
			if err != nil {
				return err
			}

			config := collab.WorkspaceConfig{
				Name:           name,
				Description:    description,
				OwnerProfileID: profile,
				DefaultPolicy:  collab.ResolutionStrategy(policy),
			}

			ws, err := mgr.Create(cmd.Context(), config)
			if err != nil {
				return err
			}

			if isJSON(cmd) {
				return outputJSON(ws)
			}

			fmt.Printf("Created workspace '%s'\n", name)
			fmt.Println()
			fmt.Println("  Next steps:")
			fmt.Printf("    aps workspace join %s\n", name)
			fmt.Printf("    aps workspace use %s\n", name)

			return nil
		},
	}

	addProfileFlag(cmd)
	_ = cmd.MarkFlagRequired("profile")
	cmd.Flags().String("description", "", "Workspace description")
	cmd.Flags().String("policy", "priority", "Conflict resolution policy")
	addJSONFlag(cmd)

	return cmd
}
