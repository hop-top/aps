package bundle

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	corebundle "hop.top/aps/internal/core/bundle"

	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a bundle YAML file and report issues",
		Long: `Validate a bundle YAML file at the given path. Parses the file
into the bundle struct (reporting YAML syntax errors with file
context), then runs the registry validator against it — required
fields, valid capability references, well-formed inheritance, and
any other invariants the registry enforces. Errors are surfaced
with the file path prefixed for grepability.

Read-only: no state mutation. Idempotent. The <file> argument is a
literal filesystem path and is not resolved through the bundle
name registry — pass aps/bundles/<name>.yaml under the user config
directory when validating a user bundle.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args[0])
		},
	}
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runValidate(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var b corebundle.Bundle
	if err := yaml.Unmarshal(data, &b); err != nil {
		return fmt.Errorf("%s: invalid YAML: %w", filePath, err)
	}

	reg, err := corebundle.NewRegistry()
	if err != nil {
		return fmt.Errorf("failed to load bundle registry: %w", err)
	}

	if err := reg.Validate(&b); err != nil {
		return fmt.Errorf("%s: %w", filePath, err)
	}

	fmt.Printf("%s %s is valid\n", successStyle.Render("✓"), filePath)
	return nil
}
