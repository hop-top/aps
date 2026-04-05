package bundle

import (
	"fmt"

	"gopkg.in/yaml.v3"
	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var resolved bool

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Print full bundle definition as YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(args[0], resolved)
		},
	}

	cmd.Flags().BoolVar(&resolved, "resolved", false,
		"Apply inheritance and print merged result")

	return cmd
}

func runShow(name string, resolved bool) error {
	reg, err := corebundle.NewRegistry()
	if err != nil {
		return fmt.Errorf("failed to load bundle registry: %w", err)
	}

	b, err := reg.Get(name)
	if err != nil {
		return err
	}

	var target interface{}

	if resolved {
		rb, err := corebundle.Resolve(*b, reg, corebundle.ProfileContext{})
		if err != nil {
			return fmt.Errorf("failed to resolve bundle %q: %w", name, err)
		}
		target = rb
	} else {
		target = b
	}

	data, err := yaml.Marshal(target)
	if err != nil {
		return fmt.Errorf("failed to marshal bundle: %w", err)
	}

	fmt.Print(string(data))
	return nil
}
