package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"hop.top/kit/go/ai/toolspec"
)

// toolspecSchemaVersion identifies the ToolSpec schema this aps build emits.
const toolspecSchemaVersion = "1.0.0"

// toolspecCommand aliases toolspec.Command so tests can name it locally
// without re-importing the kit type.
type toolspecCommand = toolspec.Command

// buildToolSpec walks the cobra command tree under root and produces a
// kit-compatible ToolSpec describing every non-hidden command, persistent
// flag, and curated error/workflow knowledge.
func buildToolSpec(root *cobra.Command) *toolspec.ToolSpec {
	spec := &toolspec.ToolSpec{
		Name:          root.Name(),
		SchemaVersion: toolspecSchemaVersion,
		Commands:      collectCommands(root),
		Flags:         collectPersistentFlags(root),
		ErrorPatterns: apsErrorPatterns(),
		Workflows:     apsWorkflows(),
		StateIntrospection: &toolspec.StateIntrospection{
			ConfigCommands: []string{"aps env", "aps version"},
			EnvVars:        []string{"APS_DATA_PATH"},
		},
	}
	return spec
}

// collectCommands returns the children of c as toolspec.Command nodes,
// recursing into each subcommand. The root node itself is not emitted.
// Hidden, "help", and "completion" subcommands are skipped — they are
// noise from an agent-consumption perspective.
func collectCommands(c *cobra.Command) []toolspec.Command {
	var out []toolspec.Command
	for _, sub := range c.Commands() {
		if sub.Hidden {
			continue
		}
		switch sub.Name() {
		case "help", "completion":
			continue
		}
		out = append(out, commandFromCobra(sub))
	}
	return out
}

// commandFromCobra projects a single cobra.Command (and its descendants)
// into the toolspec representation. Safety/contract are inferred from
// destructive-name heuristics so agents get cautious-by-default metadata.
func commandFromCobra(c *cobra.Command) toolspec.Command {
	cmd := toolspec.Command{
		Name:     c.Name(),
		Aliases:  append([]string(nil), c.Aliases...),
		Flags:    collectLocalFlags(c),
		Children: collectCommands(c),
		Safety:   inferSafety(c.Name()),
	}
	if c.Deprecated != "" {
		cmd.Deprecated = true
		cmd.DeprecatedSince = c.Deprecated
	}
	return cmd
}

// collectPersistentFlags returns the persistent flags defined directly on c.
// Used for the root spec's top-level Flags slice (global flags applied to
// every subcommand).
func collectPersistentFlags(c *cobra.Command) []toolspec.Flag {
	var flags []toolspec.Flag
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, flagFromPflag(f))
	})
	return flags
}

// collectLocalFlags returns flags defined on the command itself (excluding
// inherited persistent flags from ancestors — those live on the root spec).
func collectLocalFlags(c *cobra.Command) []toolspec.Flag {
	var flags []toolspec.Flag
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, flagFromPflag(f))
	})
	return flags
}

// inferSafety classifies a command by name. Destructive verbs map to
// dangerous + RequiresConfirmation; everything else defaults to safe.
// Lightweight heuristic — agents should still cross-check with the
// command's --help before invocation.
func inferSafety(name string) *toolspec.Safety {
	destructive := map[string]bool{
		"delete": true, "remove": true, "rm": true, "destroy": true,
		"purge": true, "drop": true,
	}
	if destructive[name] {
		return &toolspec.Safety{
			Level:                toolspec.SafetyLevelDangerous,
			RequiresConfirmation: true,
		}
	}
	return &toolspec.Safety{Level: toolspec.SafetyLevelSafe}
}

// flagFromPflag converts a pflag.Flag into a toolspec.Flag, prepending --
// to the long name to keep the namespace consistent with kit's --help-derived
// flag specs.
func flagFromPflag(f *pflag.Flag) toolspec.Flag {
	flag := toolspec.Flag{
		Name:        "--" + f.Name,
		Type:        f.Value.Type(),
		Description: f.Usage,
	}
	if f.Shorthand != "" {
		flag.Short = "-" + f.Shorthand
	}
	if f.Deprecated != "" {
		flag.Deprecated = true
	}
	return flag
}

// apsErrorPatterns is the curated set of common aps error strings agents
// hit, paired with the canonical fix. Update this when adding a new
// user-facing error class.
func apsErrorPatterns() []toolspec.ErrorPattern {
	return []toolspec.ErrorPattern{
		{
			Pattern: "unknown command or profile",
			Cause:   "first positional arg is neither a registered subcommand nor an existing profile id",
			Fix:     "run `aps profile list` to see valid ids; verify spelling",
		},
		{
			Pattern: "profile.*not found",
			Cause:   "profile id does not exist in $APS_DATA_PATH/profiles",
			Fix:     "create with `aps profile add <id>` or list existing with `aps profile list`",
		},
		{
			Pattern: "capability '.*' not found on profile",
			Cause:   "profile has no record of the requested capability",
			Fix:     "register with `aps capability add <profile> <capability>` first",
		},
		{
			Pattern: "invalid isolation level",
			Cause:   "profile.yaml specifies an unknown isolation.level value",
			Fix:     "use one of: none, env, worktree (see `aps profile show <id>`)",
		},
		{
			Pattern: "bundle .* not found",
			Cause:   "named bundle missing from registry",
			Fix:     "list with `aps bundle list`; create with `aps bundle create`",
		},
	}
}

// apsWorkflows lists the canonical multi-step sequences agents are expected
// to execute when operating on aps. Steps are ordered; each entry is a
// shell-invocable command line.
func apsWorkflows() []toolspec.Workflow {
	return []toolspec.Workflow{
		{
			Name: "create-and-launch-profile",
			Steps: []string{
				"aps profile add <id>",
				"aps capability add <id> <capability>",
				"aps <id>",
			},
		},
		{
			Name: "run-command-as-profile",
			Steps: []string{
				"aps profile list",
				"aps <profile-id> <command> [args...]",
			},
		},
		{
			Name: "register-adapter",
			Steps: []string{
				"aps adapter list",
				"aps adapter register <name>",
				"aps adapter exec <name> <action>",
			},
		},
		{
			Name: "upgrade-self",
			Steps: []string{
				"aps upgrade --quiet",
				"aps upgrade --auto",
			},
		},
	}
}

// newToolspecCmd builds the `aps toolspec` cobra subcommand. The command is
// pure metadata; it does not touch profiles, sessions, or the filesystem.
func newToolspecCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "toolspec",
		Short: "Print the aps tool specification (commands, flags, errors, workflows)",
		Long: `Output a structured ToolSpec describing every aps command, persistent
flag, known error pattern, and canonical workflow. Intended for agent
consumption — point an LLM tool runner at this output to teach it how to
drive aps without parsing --help.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			spec := buildToolSpec(rootCmd)
			return writeToolSpec(cmd.OutOrStdout(), strings.ToLower(format), spec)
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "json", "Output format (json, yaml)")
	return cmd
}

// writeToolSpec serialises spec to w in the requested format. Defaults to
// JSON for empty/unknown values to keep agents that omit --format working.
func writeToolSpec(w io.Writer, format string, spec *toolspec.ToolSpec) error {
	switch format {
	case "", "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(spec)
	case "yaml", "yml":
		return yaml.NewEncoder(w).Encode(spec)
	default:
		return fmt.Errorf("unknown format %q (valid: json, yaml)", format)
	}
}

func init() {
	rootCmd.AddCommand(newToolspecCmd())
}
