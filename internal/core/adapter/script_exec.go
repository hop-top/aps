package adapter

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/logging"
)

// ExecAction runs a script-strategy adapter action.
//
// Resolves the action's script path from the manifest, sets
// env vars from profile + action inputs, and executes the
// script. Returns stdout.
func (m *Manager) ExecAction(
	ctx context.Context,
	adapterName string,
	action string,
	inputs map[string]string,
	profileEmail string,
) (string, error) {
	device, err := LoadAdapter(adapterName)
	if err != nil {
		return "", err
	}

	if device.Strategy != StrategyScript {
		return "", fmt.Errorf(
			"adapter %q uses %s strategy; exec requires script",
			adapterName, device.Strategy,
		)
	}

	manifest, err := LoadManifest(device.ManifestPath)
	if err != nil {
		return "", fmt.Errorf("load manifest: %w", err)
	}

	scriptPath, err := resolveActionScript(
		manifest, device, action,
	)
	if err != nil {
		return "", err
	}

	// Typed view of the action's declared inputs. Validation runs
	// before the script is spawned so a rejected call has no side
	// effects, and it checks the caller-supplied map before defaults
	// are merged in — a declared default must not satisfy a
	// required marker.
	//
	// Parse diagnostics come first: they describe the manifest itself,
	// so they must be visible even on the call that a coerced
	// `required` marker goes on to reject.
	//
	// Undeclared keys still reach the script — backend scripts may
	// legitimately read vars the manifest does not enumerate — but
	// they no longer do so silently.
	schemas, diags := parseActionSchemas(manifest)
	warnManifestDiagnostics(os.Stderr, adapterName, diags)

	schema, _ := findActionSchema(schemas, action)
	if err := checkRequiredInputs(schema, action, inputs); err != nil {
		return "", err
	}
	warnUndeclaredInputs(os.Stderr, action, schema, inputs)

	env := buildScriptEnv(device, manifest, profileEmail, applyInputDefaults(schema, inputs))

	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = filepath.Dir(device.ManifestPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Action output is external content. On the failure path the
		// CLI returns the error before reaching its redacting print
		// sink, and the error string is what renders to the terminal,
		// so the output must be redacted here — at the point it enters
		// the error value — rather than at any one caller's boundary.
		safe := logging.Apply(string(out))
		return safe, fmt.Errorf(
			"action %q failed: %w\noutput: %s",
			action, err, safe,
		)
	}
	return string(out), nil
}

// warnManifestDiagnostics reports what parsing the adapter's action
// schemas had to coerce or discard.
//
// Advisory by design, and reported per exec rather than per install:
// there is no manifest-lint step to hang these on, and the exec path is
// the only place that reads the schema. A malformed manifest that the
// current call does not depend on must not break a working adapter, so
// these never change the exit code on their own — but a coerced
// `required` marker will separately reject the next call that omits the
// input, and this line is what explains why.
//
// Goes to w (os.Stderr in the exec path) rather than stdout, which
// carries action output and must stay machine-parseable.
func warnManifestDiagnostics(w io.Writer, adapterName string, diags []string) {
	for _, d := range diags {
		// Advisory: a failed warning write is not worth failing the
		// exec over, and there is no second channel to report it on.
		_, _ = fmt.Fprintf(w, "warn: adapter %q manifest: %s\n", adapterName, d)
	}
}

// warnUndeclaredInputs reports caller-supplied input keys the action's
// manifest does not declare.
//
// Advisory only: the keys still reach the script environment. Scripts
// may read vars their manifest does not enumerate, so an unknown key is
// not an error — but a silent pass-through turns a typo (`bdy=` for
// `body=`) into an input the operator believes was delivered. The
// warning makes that visible without changing the exit code.
//
// Goes to w (os.Stderr in the exec path) rather than stdout, which
// carries action output and must stay machine-parseable.
//
// An action that declares no inputs at all is skipped rather than
// flagging every supplied key: with no declared vocabulary there is
// nothing to be undeclared against, and warning would spam every
// legitimate adapter whose manifest simply omits `input:`.
func warnUndeclaredInputs(
	w io.Writer,
	action string,
	schema *ActionSchema,
	inputs map[string]string,
) {
	undeclared := undeclaredInputNames(schema, inputs)
	if len(undeclared) == 0 {
		return
	}
	// Advisory: see warnManifestDiagnostics.
	_, _ = fmt.Fprintf(
		w,
		"warn: action %q: undeclared input(s) %s; not declared in manifest, forwarded to script anyway\n",
		action, strings.Join(undeclared, ", "),
	)
}

// undeclaredInputNames returns the sorted input keys absent from the
// action's declared input list. Returns nil when the schema declares no
// inputs, so callers cannot mistake "nothing declared" for "everything
// undeclared". Sorted for deterministic output over the input map.
func undeclaredInputNames(
	schema *ActionSchema,
	inputs map[string]string,
) []string {
	if schema == nil || len(schema.Inputs) == 0 {
		return nil
	}
	var undeclared []string
	for name := range inputs {
		if _, ok := schema.FindInput(name); !ok {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	return undeclared
}

func resolveActionScript(
	manifest *AdapterManifest,
	device *Adapter,
	action string,
) (string, error) {
	actions, ok := manifest.Config["actions"]
	if !ok {
		return "", fmt.Errorf("manifest has no actions")
	}

	actionList, ok := actions.([]any)
	if !ok {
		return "", fmt.Errorf("actions must be a list")
	}

	backend, _ := device.Config["backend"].(string)
	if backend == "" {
		backend = "himalaya"
	}

	for _, a := range actionList {
		aMap, ok := a.(map[string]any)
		if !ok {
			continue
		}
		name, _ := aMap["name"].(string)
		if name != action {
			continue
		}
		script, _ := aMap["script"].(string)
		if script == "" {
			return "", fmt.Errorf(
				"action %q has no script path", action,
			)
		}
		// Template {{backend}} substitution
		script = strings.ReplaceAll(
			script, "{{backend}}", backend,
		)
		full := filepath.Join(
			filepath.Dir(device.ManifestPath), script,
		)
		if _, err := os.Stat(full); err != nil {
			return "", fmt.Errorf(
				"script %q not found: %w", full, err,
			)
		}
		return full, nil
	}

	return "", fmt.Errorf(
		"action %q not found in manifest", action,
	)
}

// applyInputDefaults returns the caller-supplied inputs with declared
// defaults filled in for keys the caller omitted entirely.
//
// Precedence: a caller-supplied value always wins, including an
// explicitly empty one — `--input cc=` means "empty", not "use the
// default". Presence of the key, not its emptiness, is the test.
//
// Defaults never satisfy a `required: true` marker: required inputs
// carry no default in practice, and this function only ever adds keys
// that declare a non-empty `default:`, so a missing required input
// stays missing for whatever validates it. Required-input enforcement
// therefore runs on the caller-supplied map, not this result.
//
// The input map is not mutated; a copy is returned only when there is
// something to add.
func applyInputDefaults(
	schema *ActionSchema,
	inputs map[string]string,
) map[string]string {
	defaults := schema.Defaults()
	if len(defaults) == 0 {
		return inputs
	}

	var merged map[string]string
	for name, value := range defaults {
		if _, supplied := inputs[name]; supplied {
			continue
		}
		if merged == nil {
			merged = make(map[string]string, len(inputs)+len(defaults))
			for k, v := range inputs {
				merged[k] = v
			}
		}
		merged[name] = value
	}
	if merged == nil {
		return inputs
	}
	return merged
}

func buildScriptEnv(
	device *Adapter,
	manifest *AdapterManifest,
	profileEmail string,
	inputs map[string]string,
) []string {
	prefix := resolveEnvPrefix(manifest)

	var env []string

	env = append(env,
		"APS_EMAIL_FROM="+profileEmail,
	)

	if account, ok := device.Config["account"].(string); ok {
		env = append(env, "APS_EMAIL_ACCOUNT="+account)
	}

	for k, v := range inputs {
		envKey := prefix + "_" + strings.ToUpper(
			strings.ReplaceAll(k, "-", "_"),
		)
		env = append(env, envKey+"="+v)
	}

	return env
}

// resolveEnvPrefix picks the env-var prefix for action inputs.
// Reads from the manifest only; the Adapter struct also carries an
// EnvPrefix field but it is a serialization passthrough for SaveAdapter
// round-trips, not a runtime override. Falls back to DefaultEnvPrefix
// when the manifest does not declare one.
func resolveEnvPrefix(manifest *AdapterManifest) string {
	if manifest != nil && manifest.EnvPrefix != "" {
		return strings.ToUpper(manifest.EnvPrefix)
	}
	return DefaultEnvPrefix
}

// LoadManifest reads and parses a manifest.yaml file.
func LoadManifest(path string) (*AdapterManifest, error) {
	if path == "" {
		return nil, fmt.Errorf("manifest path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest AdapterManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if err := validateManifestEnvPrefix(manifest.EnvPrefix); err != nil {
		return nil, fmt.Errorf("manifest %s: %w", path, err)
	}
	return &manifest, nil
}

// envPrefixPattern matches POSIX-conformant env-var name prefixes:
// a leading uppercase letter or underscore followed by any number of
// uppercase letters, digits, or underscores. Mirrors the constraint
// the shell imposes on env-var names — a prefix that violates this
// produces malformed env-var names like `MY PREFIX_FOO=...` that most
// shells silently fail to bind.
var envPrefixPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// envPrefixValidationDisableEnv is the operator-set escape hatch.
// Setting it to "1" skips the validateManifestEnvPrefix check at
// LoadManifest time. Use only for migration scenarios where a fixed
// non-conformant prefix must be tolerated briefly; the runtime
// behaviour with such a prefix is undefined.
const envPrefixValidationDisableEnv = "APS_ADAPTER_DISABLE_PREFIX_VALIDATION"

// validateManifestEnvPrefix checks that the prefix is a valid env-var
// name component (after ToUpper) or empty (manifest opts into the
// default). Skipped when envPrefixValidationDisableEnv is set to "1".
func validateManifestEnvPrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	if os.Getenv(envPrefixValidationDisableEnv) == "1" {
		return nil
	}
	upper := strings.ToUpper(prefix)
	if !envPrefixPattern.MatchString(upper) {
		return fmt.Errorf("env_prefix %q is not a valid env-var name component (must match %s); set %s=1 to bypass", prefix, envPrefixPattern.String(), envPrefixValidationDisableEnv)
	}
	return nil
}
