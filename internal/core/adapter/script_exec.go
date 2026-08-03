package adapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

	// Typed view of the action's declared inputs. Parsed here so input
	// validation has a model to work against; nothing consumes it yet,
	// so caller-supplied inputs still pass through unfiltered.
	schema, _ := findActionSchema(parseActionSchemas(manifest), action)
	_ = schema

	env := buildScriptEnv(device, manifest, profileEmail, inputs)

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
