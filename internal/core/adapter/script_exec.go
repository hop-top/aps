package adapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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

	env := buildScriptEnv(device, profileEmail, inputs)

	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = filepath.Dir(device.ManifestPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf(
			"action %q failed: %w\noutput: %s",
			action, err, string(out),
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
	profileEmail string,
	inputs map[string]string,
) []string {
	var env []string

	env = append(env,
		"APS_EMAIL_FROM="+profileEmail,
	)

	if account, ok := device.Config["account"].(string); ok {
		env = append(env, "APS_EMAIL_ACCOUNT="+account)
	}

	for k, v := range inputs {
		envKey := "EMAIL_" + strings.ToUpper(
			strings.ReplaceAll(k, "-", "_"),
		)
		env = append(env, envKey+"="+v)
	}

	return env
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
	return &manifest, nil
}
