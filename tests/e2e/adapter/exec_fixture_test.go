package adapter_e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Shared fixture for `aps adapter exec` end-to-end coverage.
//
// Registers a script-strategy adapter into an isolated HOME whose
// action scripts echo the environment they received instead of
// invoking a real backend. Assertions can then read exactly which
// <PREFIX>_* vars reached the script. No external binary (himalaya
// or otherwise) is required, so the fixture runs unchanged in CI.

// execFixtureName is the adapter name registered by
// writeExecFixtureAdapter.
const execFixtureName = "fixture"

// execFixtureEnvPrefix is the manifest env_prefix. Action inputs
// reach scripts as <execFixtureEnvPrefix>_<UPPER_INPUT_NAME>.
const execFixtureEnvPrefix = "FIXTURE"

// execFixtureEnvMarker prefixes every env line the stub scripts emit,
// so assertions can separate fixture output from anything else the
// command prints.
const execFixtureEnvMarker = "ENV "

// execFixtureBackend is the backend directory name the manifest's
// {{backend}} template resolves to. Left unset in adapter config so
// resolveActionScript falls back to its "himalaya" default; scripts
// live under backends/himalaya/ accordingly.
const execFixtureBackend = "himalaya"

// execFixtureOptions tunes the adapter written by
// writeExecFixtureAdapter. The zero value yields the standard
// all-actions-succeed fixture.
type execFixtureOptions struct {
	// Name overrides the adapter name (default execFixtureName).
	Name string

	// FailingActions maps an action name to the exit code its stub
	// script should return. A failing stub still echoes its env on
	// stdout, then writes a diagnostic to stderr and exits non-zero.
	FailingActions map[string]int

	// Backend, when non-empty, is written to the adapter's
	// config.backend and used as the backends/<backend>/ directory
	// the {{backend}} template resolves to.
	Backend string

	// Account, when non-empty, is written to config.account. The
	// runtime forwards it as APS_EMAIL_ACCOUNT.
	Account string

	// EnvPrefix overrides the manifest env_prefix (default
	// execFixtureEnvPrefix). Empty string keeps the default; use
	// OmitEnvPrefix to test the manifest-omits-prefix path.
	EnvPrefix string

	// OmitEnvPrefix drops env_prefix from the manifest entirely so
	// the runtime falls back to its built-in default prefix.
	OmitEnvPrefix bool
}

// execFixtureActions lists the actions the fixture manifest declares.
// Input shapes mirror the real email adapter: required inputs,
// inputs carrying a default, and plain optional inputs.
var execFixtureActions = []execFixtureAction{
	{
		Name:        "send",
		Description: "Send a message",
		Inputs: []execFixtureInput{
			{Name: "to", Required: true},
			{Name: "subject", Required: true},
			{Name: "body", Required: true},
			{Name: "cc"},
		},
	},
	{
		Name:        "reply",
		Description: "Reply to a message",
		Inputs: []execFixtureInput{
			{Name: "id", Required: true, Description: "Envelope ID to reply to"},
			{Name: "body", Required: true},
		},
	},
	{
		Name:        "list",
		Description: "List envelopes",
		Inputs: []execFixtureInput{
			{Name: "limit", Default: "10"},
			{Name: "folder", Default: "INBOX"},
			{Name: "query"},
		},
	},
	{
		Name:        "read",
		Description: "Read a message",
		Inputs: []execFixtureInput{
			{Name: "id", Required: true, Description: "Envelope ID to read"},
			{Name: "preview-only"},
		},
	},
}

type execFixtureAction struct {
	Name        string
	Description string
	Inputs      []execFixtureInput
}

type execFixtureInput struct {
	Name        string
	Description string
	Required    bool
	Default     string
}

// writeExecFixtureAdapter registers the script-strategy fixture
// adapter under home and returns its name. Scripts are written
// executable (0o755) under <adapterDir>/backends/<backend>/, matching
// the relative script paths in the manifest — the runtime resolves
// them against the manifest's directory.
func writeExecFixtureAdapter(
	t *testing.T,
	home string,
	opts execFixtureOptions,
) string {
	t.Helper()

	name := opts.Name
	if name == "" {
		name = execFixtureName
	}
	backend := opts.Backend
	if backend == "" {
		backend = execFixtureBackend
	}

	writeGlobalAdapter(t, home, name, execFixtureManifest(name, opts))

	scriptDir := filepath.Join(
		home, ".local", "share", "aps", "devices", name,
		"backends", backend,
	)
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("mkdir fixture script dir: %v", err)
	}

	for _, action := range execFixtureActions {
		exitCode := opts.FailingActions[action.Name]
		path := filepath.Join(scriptDir, action.Name+".sh")
		body := execFixtureScript(action.Name, exitCode)
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("write fixture script %s: %v", path, err)
		}
	}

	return name
}

// execFixtureManifest renders the fixture manifest.yaml.
func execFixtureManifest(name string, opts execFixtureOptions) string {
	var b strings.Builder

	b.WriteString("api_version: v1\n")
	b.WriteString("kind: adapter\n")
	fmt.Fprintf(&b, "name: %s\n", name)
	b.WriteString("type: messenger\n")
	b.WriteString("strategy: script\n")

	if !opts.OmitEnvPrefix {
		prefix := opts.EnvPrefix
		if prefix == "" {
			prefix = execFixtureEnvPrefix
		}
		fmt.Fprintf(&b, "env_prefix: %s\n", prefix)
	}

	b.WriteString("description: Script-strategy fixture adapter for exec tests\n")
	b.WriteString("config:\n")
	if opts.Backend != "" {
		fmt.Fprintf(&b, "  backend: %s\n", opts.Backend)
	}
	if opts.Account != "" {
		fmt.Fprintf(&b, "  account: %s\n", opts.Account)
	}
	b.WriteString("  actions:\n")

	for _, action := range execFixtureActions {
		fmt.Fprintf(&b, "  - name: %s\n", action.Name)
		fmt.Fprintf(&b, "    description: %s\n", action.Description)
		fmt.Fprintf(
			&b, "    script: backends/{{backend}}/%s.sh\n", action.Name,
		)
		b.WriteString("    input:\n")
		for _, in := range action.Inputs {
			fmt.Fprintf(&b, "      - name: %s\n", in.Name)
			fmt.Fprintf(&b, "        required: %t\n", in.Required)
			if in.Default != "" {
				fmt.Fprintf(&b, "        default: %q\n", in.Default)
			}
			if in.Description != "" {
				fmt.Fprintf(&b, "        description: %s\n", in.Description)
			}
		}
	}

	return b.String()
}

// execFixtureScript renders a stub action script. The stub prints the
// action name, then one `ENV <KEY>=<VALUE>` line per APS_* and
// prefix-scoped env var it received, sorted for deterministic output.
// A non-zero exitCode additionally writes to stderr and exits with it.
func execFixtureScript(action string, exitCode int) string {
	var b strings.Builder

	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -u\n")
	fmt.Fprintf(&b, "echo \"ACTION %s\"\n", action)
	b.WriteString("echo \"PWD $PWD\"\n")
	// env | grep is intentionally tolerant: no match must not fail
	// the script, so the pipeline ends with `|| true`.
	fmt.Fprintf(
		&b,
		"env | grep -E '^(APS_|%s_|%s_)' | sort | sed 's/^/%s/' || true\n",
		execFixtureEnvPrefix, "ADAPTER", execFixtureEnvMarker,
	)

	if exitCode != 0 {
		fmt.Fprintf(
			&b,
			"echo \"fixture action %s failed deliberately\" >&2\n",
			action,
		)
		fmt.Fprintf(&b, "exit %d\n", exitCode)
	}
	b.WriteString("exit 0\n")

	return b.String()
}

// parseFixtureEnv extracts the `ENV KEY=VALUE` lines a stub script
// emitted into a map, dropping the marker prefix.
func parseFixtureEnv(out string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, execFixtureEnvMarker) {
			continue
		}
		kv := strings.TrimPrefix(line, execFixtureEnvMarker)
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		env[parts[0]] = parts[1]
	}
	return env
}

// fixtureEnvKeys returns the sorted keys of a parseFixtureEnv map,
// for readable failure messages.
func fixtureEnvKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestExecFixture_SmokeEchoesEnv proves the fixture works end-to-end:
// the stub script runs under `aps adapter exec` and its stdout carries
// the action inputs as prefixed env vars.
func TestExecFixture_SmokeEchoesEnv(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=Hello",
		"--input", "body=Message body",
	)
	if err != nil {
		t.Fatalf("exec fixture send: %v\nstdout: %s\nstderr: %s",
			err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("expected non-empty stdout, got empty (stderr: %s)", stderr)
	}
	if !strings.Contains(stdout, "ACTION send") {
		t.Errorf("stdout missing action marker:\n%s", stdout)
	}

	env := parseFixtureEnv(stdout)
	want := map[string]string{
		"FIXTURE_TO":      "user@example.com",
		"FIXTURE_SUBJECT": "Hello",
		"FIXTURE_BODY":    "Message body",
		"APS_EMAIL_FROM":  "ops@example.com",
	}
	for k, v := range want {
		got, ok := env[k]
		if !ok {
			t.Errorf("env %s not received by script; got keys %v",
				k, fixtureEnvKeys(env))
			continue
		}
		if got != v {
			t.Errorf("env %s = %q, want %q", k, got, v)
		}
	}
}

// TestExecFixture_FailingVariant proves the opt-in failing variant
// surfaces a non-zero exit and its stderr diagnostic.
func TestExecFixture_FailingVariant(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	name := writeExecFixtureAdapter(t, home, execFixtureOptions{
		Name:           "failing",
		FailingActions: map[string]int{"send": 3},
	})

	stdout, stderr, err := runAPS(t, home,
		"adapter", "exec", name, "send",
		"--from", "ops@example.com",
		"--input", "to=user@example.com",
		"--input", "subject=Hello",
		"--input", "body=Message body",
	)
	if err == nil {
		t.Fatalf("expected failure, got success\nstdout: %s", stdout)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "failed deliberately") {
		t.Errorf("expected stub stderr in output, got:\nstdout: %s\nstderr: %s",
			stdout, stderr)
	}
}
