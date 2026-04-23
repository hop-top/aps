package adapter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	coreadapter "hop.top/aps/internal/core/adapter"
)

func newExecCmd() *cobra.Command {
	var profile string
	var from string
	var inputs []string

	cmd := &cobra.Command{
		Use:   "exec <adapter> <action>",
		Short: "Execute a script-strategy adapter action",
		Long: `Run an action on a script-strategy adapter.

The adapter's manifest defines available actions and their
scripts. Profile email is resolved from the --profile flag.

Examples:
  aps adapter exec email send --profile noor \
    --input to=user@example.com \
    --input subject="Hello" \
    --input body="Message body"

  aps adapter exec email reply --profile noor \
    --input id=7131 \
    --input body="Thanks!"

  aps adapter exec email list --profile noor

  aps adapter exec email read --profile noor \
    --input id=7131

  # Explicit from address (no profile needed)
  aps adapter exec email send --from ops@company.com \
    --input to=user@example.com \
    --input subject="Hello" \
    --input body="Hi"`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			adapterName := args[0]
			action := args[1]

			profileEmail, err := resolveFromAddress(
				from, profile,
			)
			if err != nil {
				return err
			}

			inputMap := parseInputs(inputs)

			mgr := coreadapter.NewManager()
			out, err := mgr.ExecAction(
				context.Background(),
				adapterName,
				action,
				inputMap,
				profileEmail,
			)
			if err != nil {
				return err
			}

			fmt.Print(out)
			return nil
		},
	}

	cmd.Flags().StringVarP(
		&profile, "profile", "p", "",
		"Profile ID (resolves email from profile.yaml)",
	)
	cmd.Flags().StringVar(
		&from, "from", "",
		"Explicit From address (overrides profile lookup)",
	)
	cmd.Flags().StringArrayVarP(
		&inputs, "input", "i", nil,
		"Action input as key=value (repeatable)",
	)

	return cmd
}

func parseInputs(raw []string) map[string]string {
	m := make(map[string]string, len(raw))
	for _, kv := range raw {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

// resolveFromAddress determines the From email address.
// Priority: --from flag > profile.yaml email field.
func resolveFromAddress(
	from string,
	profileID string,
) (string, error) {
	if from != "" {
		return from, nil
	}
	if profileID == "" {
		return "", fmt.Errorf(
			"--from or --profile is required for exec",
		)
	}

	dataDir, err := getProfileDir(profileID)
	if err != nil {
		return "", err
	}

	email, err := readProfileEmail(dataDir)
	if err != nil {
		return "", fmt.Errorf(
			"profile %q has no email; use --from: %w",
			profileID, err,
		)
	}
	return email, nil
}

func getProfileDir(id string) (string, error) {
	dataDir := os.Getenv("APS_DATA_PATH")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataDir = filepath.Join(
			home, ".local", "share", "aps",
		)
	}
	return filepath.Join(dataDir, "profiles", id), nil
}

func readProfileEmail(dir string) (string, error) {
	// 1. Check profile.yaml email field
	profData, err := os.ReadFile(
		filepath.Join(dir, "profile.yaml"),
	)
	if err == nil {
		var profile struct {
			Email string `yaml:"email"`
		}
		if yaml.Unmarshal(profData, &profile) == nil {
			if profile.Email != "" {
				return profile.Email, nil
			}
		}
	}

	// 2. Fall back to gitconfig [user] email
	gitcfg := filepath.Join(dir, "gitconfig")
	data, err := os.ReadFile(gitcfg)
	if err != nil {
		return "", fmt.Errorf(
			"no email in profile.yaml or gitconfig",
		)
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "email") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				email := strings.TrimSpace(parts[1])
				if email != "" && email != "agent@example.com" {
					return email, nil
				}
			}
		}
	}
	return "", fmt.Errorf(
		"no email in profile.yaml or gitconfig",
	)
}
