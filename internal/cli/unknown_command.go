package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"hop.top/kit/go/console/output"

	"hop.top/aps/internal/cli/exit"
	"hop.top/aps/internal/styles"
)

// flagTakesValue reports whether f consumes the following token as its
// value. Unknown flags (nil) are assumed to take one: guessing wrong in
// that direction only skips a token, whereas assuming otherwise would
// misreport a flag value as an unknown subcommand.
func flagTakesValue(f *pflag.Flag) bool {
	if f == nil {
		return true
	}
	// NoOptDefVal is set for flags usable as bare switches (--verbose),
	// which do not consume the next token.
	return f.NoOptDefVal == ""
}

// unknownSubcommandError builds the structured usage envelope returned
// for a mistyped subcommand on a group node.
func unknownSubcommandError(group *cobra.Command, token string) *output.Error {
	fix := "run `" + group.CommandPath() + " --help` to see available commands"
	return &output.Error{
		Code:         output.CodeUsage,
		Message:      fmt.Sprintf("unknown command %q for %q — %s", token, group.CommandPath(), fix),
		SuggestedFix: fix,
		ExitCode:     2,
	}
}

// findUnknownSubcommand reports the group node and offending token when
// args resolve to a group that still carries an unconsumed positional.
//
// It exists because cobra reports success for this case. Command.execute
// returns flag.ErrHelp for a non-runnable node BEFORE reaching
// ValidateArgs (the `if !c.Runnable()` bail precedes the
// `c.ValidateArgs` call in cobra's command.go), and Execute maps
// ErrHelp to a nil error — so `aps profile frobnicate` prints help and
// exits 0, telling an agent branching on $? that a mistyped command
// succeeded. Setting cmd.Args on the group cannot fix it: that
// validator is never consulted, and assigning it additionally
// suppresses cobra's own legacyArgs unknown-command check, which Find
// only applies while Args is nil.
//
// Resolution therefore happens before dispatch, against the same
// Command.Find cobra itself uses, so a valid path resolves identically
// to normal execution.
//
// Returns (nil, "") when args are well-formed: a runnable target, a
// bare group invocation (cobra renders help, exit 0), or leftovers that
// are purely flags such as `aps profile --help`.
func findUnknownSubcommand(root *cobra.Command, args []string) (*cobra.Command, string) {
	target, rest, err := root.Find(args)
	if err != nil || target == nil {
		// Find already failed (cobra's own unknown-command path on the
		// root); let Execute surface its error unchanged.
		return nil, ""
	}
	// A runnable target owns its positionals; a leaf validates its own
	// args via cobra.Args.
	if target.Runnable() || !target.HasSubCommands() {
		return nil, ""
	}
	if token, ok := firstPositional(target, rest); ok {
		return target, token
	}
	return nil, ""
}

// firstPositional returns the first true positional in args, resolved
// against target's flag set.
//
// Flag values must not be mistaken for subcommand names: in
// `aps profile --format json frobnicate`, "json" is the argument to
// --format and "frobnicate" is the offending token. Arity therefore
// has to be resolved against the actual flag definitions rather than
// by scanning for a leading "-", which is why the flag set is merged
// first (Flags() folds in inherited persistent flags such as --format).
//
// Everything after a `--` terminator belongs to the invoked command
// and is never inspected.
func firstPositional(target *cobra.Command, args []string) (string, bool) {
	flags := target.Flags()
	for i := 0; i < len(args); i++ {
		s := args[i]
		switch {
		case s == "--":
			return "", false
		case strings.HasPrefix(s, "--"):
			// `--flag=value` carries its value inline; a bare `--flag`
			// consumes the next token unless it is boolean-like.
			if !strings.Contains(s, "=") && flagTakesValue(flags.Lookup(strings.TrimPrefix(s, "--"))) {
				i++
			}
		case strings.HasPrefix(s, "-") && s != "-":
			name := strings.TrimPrefix(s, "-")
			if !strings.Contains(s, "=") && len(name) == 1 &&
				flagTakesValue(flags.ShorthandLookup(name)) {
				i++
			}
		default:
			return s, true
		}
	}
	return "", false
}

// scanArgsForFormat resolves the --format value straight from argv.
//
// This check runs ahead of cobra's flag parsing (the whole point is to
// answer before dispatch), so the bound flag value is not yet
// populated. Mirrors kit's own pre-parse scan for --api-version.
// Returns "" when unset, which RenderError treats as the human default.
func scanArgsForFormat(args []string) string {
	for i, a := range args {
		switch {
		case a == "--format", a == "-f":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case strings.HasPrefix(a, "--format="):
			return strings.TrimPrefix(a, "--format=")
		}
	}
	return ""
}

// renderPreDispatchError writes err to stderr in the format the user
// asked for, for failures detected before kit's RunE middleware is in
// play. A structured envelope renders as JSON/YAML under --format
// json|yaml so agents can parse it; anything else falls back to the
// styled human line.
func renderPreDispatchError(err error) {
	var envelope *output.Error
	if errors.As(err, &envelope) {
		format := scanArgsForFormat(os.Args[1:])
		if renderErr := output.RenderError(os.Stderr, format, envelope); renderErr == nil {
			return
		}
	}
	fmt.Fprintln(os.Stderr, styles.Error.Render("Error: "+err.Error()))
}

// classifyLeafErrors walks the tree and wraps every RunE so a returned
// domain error carries the envelope its class implies.
//
// Kit's own middleware flattens any error that does not already
// implement AsCLIError into GENERIC/ExitCode=1, which made
// `aps profile create <dup>` exit 1 despite raising domain.ErrConflict
// (spec: 4) and a missing profile exit 1 instead of the not-found
// class (spec: 3). Running this pass before kit's WrapRunE means kit
// sees an already-classified error and preserves it.
//
// Idempotent: the annotation keeps the wrapper single-shot per leaf.
func classifyLeafErrors(root *cobra.Command) {
	const classifiedAnnotation = "aps/errors-classified"

	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, c := range cmd.Commands() {
			walk(c)
		}
		if cmd.RunE == nil || cmd.Annotations[classifiedAnnotation] == "true" {
			return
		}
		inner := cmd.RunE
		cmd.RunE = func(c *cobra.Command, args []string) error {
			return exit.Envelope(inner(c, args))
		}
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations[classifiedAnnotation] = "true"
	}
	walk(root)
}

// rejectUnknownSubcommand returns the usage envelope for args that name
// a nonexistent subcommand under a group node, or nil when args are
// well-formed. Callers surface the envelope instead of executing, so
// the process exits 2 with a corrective, machine-readable diagnostic.
//
// The tree is left untouched: no node is made runnable and no Args
// validator is replaced. That matters because kit gates MissingLong,
// kit/top-level-verb, and MaxTopLevelVerbs on cmd.Runnable() — flipping
// aps's 22 group nodes runnable to reach their validators would re-file
// them as depth-1 leaves and fail the strict gates aps ships with.
func rejectUnknownSubcommand(root *cobra.Command, args []string) error {
	group, token := findUnknownSubcommand(root, args)
	if group == nil {
		return nil
	}
	return unknownSubcommandError(group, token)
}
