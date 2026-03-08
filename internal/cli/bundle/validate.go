package bundle

import (
	"fmt"
	"os"

	corebundle "hop.top/aps/internal/core/bundle"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a bundle YAML file and report issues",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args[0])
		},
	}
}

func runValidate(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var b corebundle.Bundle
	if err := yaml.Unmarshal(data, &b); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: invalid YAML: %v\n",
			errorStyle.Render("✗"), filePath, err)
		os.Exit(1)
	}

	reg, err := corebundle.NewRegistry()
	if err != nil {
		return fmt.Errorf("failed to load bundle registry: %w", err)
	}

	if err := reg.Validate(&b); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %v\n",
			errorStyle.Render("✗"), filePath, err)
		os.Exit(1)
	}

	fmt.Printf("%s %s is valid\n", successStyle.Render("✓"), filePath)
	return nil
}
