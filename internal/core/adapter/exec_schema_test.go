package adapter

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// scriptAdapter installs a script-strategy adapter named "probe" under a
// temp APS_DATA_PATH and returns its directory. manifestBody is the
// manifest YAML; script is the shell body of backends/probe.sh.
//
// Exists so the enforcement sequence can be exercised through the real
// ExecAction entry point — LoadAdapter, LoadManifest, script resolution
// and spawn included — rather than through its parts.
func scriptAdapter(t *testing.T, manifestBody, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("script-strategy adapters need a POSIX shell")
	}

	data := t.TempDir()
	t.Setenv("APS_DATA_PATH", data)

	dir := filepath.Join(data, "devices", "probe")
	backends := filepath.Join(dir, "backends")
	if err := os.MkdirAll(backends, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, ManifestFileName), []byte(manifestBody), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(backends, "probe.sh"), []byte(script), 0o755,
	); err != nil {
		t.Fatal(err)
	}
}

// echoEnvScript dumps every PROBE_-prefixed env var, one per line, so a
// test can assert exactly what reached the script.
const echoEnvScript = "#!/bin/sh\nenv | grep '^PROBE_' | sort\n"

const probeManifest = `name: probe
type: messenger
strategy: script
env_prefix: PROBE
config:
  actions:
  - name: send
    script: backends/probe.sh
    input:
      - name: to
        required: true
      - name: folder
        default: INBOX
`

// TestExecAction_RequiredCheckPrecedesDefaults is the ordering contract:
// required-input enforcement reads the CALLER-supplied map, and it runs
// before defaults are merged. An input that is both `required: true` and
// carries a `default:` must still be rejected when omitted — otherwise
// the default silently satisfies the marker and the requirement is dead
// text in every manifest that pairs the two.
//
// Driven through ExecAction rather than checkRequiredInputs so the two
// axes are wired in the order production wires them; a unit test of the
// checker alone cannot observe a reordering of the exec path.
func TestExecAction_RequiredCheckPrecedesDefaults(t *testing.T) {
	scriptAdapter(t, `name: probe
type: messenger
strategy: script
env_prefix: PROBE
config:
  actions:
  - name: send
    script: backends/probe.sh
    input:
      - name: to
        required: true
        default: ops@example.com
`, "#!/bin/sh\necho spawned\n")

	out, err := NewManager().ExecAction(
		context.Background(), "probe", "send", nil, "me@example.com",
	)
	if err == nil {
		t.Fatalf("required input with a declared default must still be "+
			"required; got output %q", out)
	}
	if !strings.Contains(err.Error(), "missing required input 'to'") {
		t.Fatalf("error should name the missing required input; got %v", err)
	}
	// Rejection happens before spawn: a script that ran would have
	// echoed, and the default would have masked the omission.
	if strings.Contains(out, "spawned") {
		t.Fatalf("script was spawned despite a rejected call; out=%q", out)
	}
	if strings.Contains(out, "ops@example.com") ||
		strings.Contains(err.Error(), "ops@example.com") {
		t.Fatalf("default leaked into the rejected call: out=%q err=%v", out, err)
	}
}

// TestExecAction_SequenceOnAcceptedCall walks the accepted path end to
// end: required satisfied, undeclared key warned about but forwarded,
// declared default filled in for the omitted key.
func TestExecAction_SequenceOnAcceptedCall(t *testing.T) {
	scriptAdapter(t, probeManifest, echoEnvScript)

	out, err := NewManager().ExecAction(
		context.Background(), "probe", "send",
		map[string]string{"to": "u@example.com", "bdy": "typo"},
		"me@example.com",
	)
	if err != nil {
		t.Fatalf("ExecAction: %v\noutput: %s", err, out)
	}

	for _, want := range []string{
		"PROBE_TO=u@example.com", // caller-supplied
		"PROBE_FOLDER=INBOX",     // declared default filled in
		"PROBE_BDY=typo",         // undeclared, still forwarded
	} {
		if !strings.Contains(out, want) {
			t.Errorf("env missing %q; got:\n%s", want, out)
		}
	}
}

// TestExecAction_UndeclaredKeyCollidingWithDefaultName pins the
// precedence when the two optional axes meet on the same name: a
// supplied key is a supplied key even when the manifest also declares a
// default for it, so the default must not overwrite it.
func TestExecAction_UndeclaredKeyCollidingWithDefaultName(t *testing.T) {
	scriptAdapter(t, probeManifest, echoEnvScript)

	out, err := NewManager().ExecAction(
		context.Background(), "probe", "send",
		map[string]string{"to": "u@example.com", "folder": "Sent"},
		"me@example.com",
	)
	if err != nil {
		t.Fatalf("ExecAction: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "PROBE_FOLDER=Sent") {
		t.Errorf("supplied value should win over the default; got:\n%s", out)
	}
	if strings.Contains(out, "PROBE_FOLDER=INBOX") {
		t.Errorf("default overwrote the supplied value; got:\n%s", out)
	}
}

// TestExecAction_UnknownActionRejectedBeforeSchema guards the ordering
// against the other side: an action the manifest does not declare fails
// at script resolution, so a nil schema never reaches enforcement and
// never panics there.
func TestExecAction_UnknownActionRejectedBeforeSchema(t *testing.T) {
	scriptAdapter(t, probeManifest, echoEnvScript)

	_, err := NewManager().ExecAction(
		context.Background(), "probe", "archive", nil, "me@example.com",
	)
	if err == nil {
		t.Fatal("unknown action should be rejected")
	}
	if !strings.Contains(err.Error(), "archive") {
		t.Fatalf("error should name the unknown action; got %v", err)
	}
}

// TestExecAction_NonScriptStrategyRejected keeps the strategy guard
// ahead of manifest parsing: a non-script adapter must not reach the
// schema path at all.
func TestExecAction_NonScriptStrategyRejected(t *testing.T) {
	scriptAdapter(t, `name: probe
type: messenger
strategy: builtin
config:
  actions:
  - name: send
    script: backends/probe.sh
`, echoEnvScript)

	_, err := NewManager().ExecAction(
		context.Background(), "probe", "send", nil, "me@example.com",
	)
	if err == nil {
		t.Fatal("non-script strategy should be rejected by exec")
	}
	if !strings.Contains(err.Error(), "script") {
		t.Fatalf("error should explain the strategy requirement; got %v", err)
	}
}

// TestParseActionInputs_DuplicateNames is the duplicate-name contract:
// a name declared twice collapses to the FIRST declaration at parse
// time, and the collapse is reported.
//
// Dropping the later entry is what makes the accessors agree. While
// both entries survived, FindInput read the first and Defaults() the
// last, so two call sites could take two different answers off one
// manifest — the exec path could warn that an input was undeclared
// while filling in a default for it. Collapsing at the parse boundary
// means every accessor sees one entry per name by construction rather
// than by each accessor's own scan order.
func TestParseActionInputs_DuplicateNames(t *testing.T) {
	manifest := manifestFromYAML(t, `name: probe
type: messenger
config:
  actions:
  - name: list
    script: backends/list.sh
    input:
      - name: folder
        default: INBOX
        description: first
      - name: folder
        default: Sent
        description: second
`)

	schemas, diags := parseActionSchemas(manifest)
	list, ok := findActionSchema(schemas, "list")
	if !ok {
		t.Fatal(`action "list" not parsed`)
	}
	if got := len(list.Inputs); got != 1 {
		t.Fatalf("duplicate should collapse to one entry; got %d: %+v",
			got, list.Inputs)
	}
	if len(diags) != 1 ||
		!strings.Contains(diags[0], `input "folder" declared more than once`) {
		t.Fatalf("collapse should be reported; got %v", diags)
	}

	first, _ := list.FindInput("folder")
	if first.Description != "first" {
		t.Errorf("FindInput should return the first declaration; got %q",
			first.Description)
	}
	// The point of the collapse: both accessors now name the same entry.
	if got, want := list.Defaults(), map[string]string{"folder": "INBOX"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Defaults() = %v, want %v (first declaration wins, "+
			"matching FindInput)", got, want)
	}
}

// TestParseActionInputs_DuplicateRequiredMarker: a name declared twice
// yields one required entry, not one per declaration, because the
// duplicate never reaches the schema.
func TestParseActionInputs_DuplicateRequiredMarker(t *testing.T) {
	manifest := manifestFromYAML(t, `name: probe
type: messenger
config:
  actions:
  - name: send
    script: backends/send.sh
    input:
      - name: to
        required: true
      - name: to
        required: true
`)

	send, _ := findActionSchema(schemasOf(manifest), "send")
	if got, want := send.RequiredInputs(), []string{"to"}; !reflect.DeepEqual(got, want) {
		t.Errorf("RequiredInputs() = %v, want %v; a collapsed duplicate must "+
			"not be reported twice", got, want)
	}
	if err := checkRequiredInputs(send, "send", map[string]string{"to": "u@example.com"}); err != nil {
		t.Fatalf("supplying the name should satisfy it; got %v", err)
	}
	// And the rejection names it once, not once per declaration.
	err := checkRequiredInputs(send, "send", nil)
	if err == nil {
		t.Fatal("omitting the required input should be rejected")
	}
	if got := strings.Count(err.Error(), "'to'"); got != 1 {
		t.Errorf("missing input named %d times, want 1: %v", got, err)
	}
}

// TestParseActionInputs_RequiredAsString is the defect this change
// closes. `required: "true"` is a string to yaml.v3, not a bool. It
// used to fail a discarded type assertion and leave the input OPTIONAL
// — a quoted marker silently retired the requirement, with nothing
// visible at any layer.
//
// Now an unreadable marker resolves to REQUIRED and says so. Failing
// closed is the safe direction: the worst case is a rejected call that
// names the input and the manifest line to fix, rather than an accepted
// call that was supposed to be blocked.
func TestParseActionInputs_RequiredAsString(t *testing.T) {
	manifest := manifestFromYAML(t, `name: probe
type: messenger
config:
  actions:
  - name: send
    script: backends/send.sh
    input:
      - name: to
        required: "true"
      - name: subject
        required: true
`)

	schemas, diags := parseActionSchemas(manifest)
	send, _ := findActionSchema(schemas, "send")
	if got, want := send.RequiredInputs(), []string{"to", "subject"}; !reflect.DeepEqual(got, want) {
		t.Errorf("RequiredInputs() = %v, want %v; a quoted marker must not "+
			"silently leave the input optional", got, want)
	}
	if len(diags) != 1 {
		t.Fatalf("the coercion should be reported once; got %v", diags)
	}
	for _, want := range []string{
		`action "send"`, `input "to"`, "non-boolean required",
		"REQUIRED", "`required: true`",
	} {
		if !strings.Contains(diags[0], want) {
			t.Errorf("diagnostic %q missing %q", diags[0], want)
		}
	}

	// The consequence at the enforcement boundary: the quoted marker is
	// now enforced, where before it was inert.
	err := checkRequiredInputs(send, "send", map[string]string{"subject": "hi"})
	if err == nil {
		t.Fatal("a quoted required marker must be enforced, not ignored")
	}
	if !strings.Contains(err.Error(), "'to'") {
		t.Errorf("rejection should name the input; got %v", err)
	}
}

// TestWarnManifestDiagnostics_RendersToStderr pins where the parse
// diagnostics surface: one advisory line per diagnostic on the writer
// the exec path points at stderr, naming the adapter.
func TestWarnManifestDiagnostics_RendersToStderr(t *testing.T) {
	var buf strings.Builder
	warnManifestDiagnostics(&buf, "probe", []string{"first thing", "second thing"})

	got := buf.String()
	for _, want := range []string{
		"warn: adapter \"probe\" manifest: first thing\n",
		"warn: adapter \"probe\" manifest: second thing\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q missing %q", got, want)
		}
	}

	// Nothing to report writes nothing at all.
	var quiet strings.Builder
	warnManifestDiagnostics(&quiet, "probe", nil)
	if quiet.String() != "" {
		t.Errorf("clean manifest should print nothing; got %q", quiet.String())
	}
}

// TestExecAction_QuotedRequiredMarkerIsEnforced walks the fix through
// the real entry point: a manifest whose `required` marker is quoted
// both warns on stderr and rejects the call that omits the input.
// Before this change the same manifest ran the script silently.
func TestExecAction_QuotedRequiredMarkerIsEnforced(t *testing.T) {
	scriptAdapter(t, `name: probe
type: messenger
strategy: script
env_prefix: PROBE
config:
  actions:
  - name: send
    script: backends/probe.sh
    input:
      - name: to
        required: "true"
`, "#!/bin/sh\necho spawned\n")

	out, err := NewManager().ExecAction(
		context.Background(), "probe", "send", nil, "me@example.com",
	)
	if err == nil {
		t.Fatalf("quoted required marker must reject the omitting call; "+
			"got output %q", out)
	}
	if !strings.Contains(err.Error(), "missing required input 'to'") {
		t.Fatalf("error should name the missing required input; got %v", err)
	}
	if strings.Contains(out, "spawned") {
		t.Fatalf("script ran despite a rejected call; out=%q", out)
	}
}

// TestScalarString covers the YAML-scalar normalisation that feeds
// Defaults(). Manifest authors write defaults unquoted, so the value
// arrives typed and must render back to the string the script env needs.
func TestScalarString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil is empty", nil, ""},
		{"string passes through", "INBOX", "INBOX"},
		{"empty string stays empty", "", ""},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 10, "10"},
		{"negative int", -3, "-3"},
		{"int64", int64(9007199254740993), "9007199254740993"},
		{"float renders without exponent", 1.5, "1.5"},
		{"whole float drops the point", 2.0, "2"},
		// yaml.v3 hands back types this switch does not enumerate
		// (sequences, mappings, uints); they render empty rather than
		// leaking a Go-syntax string into the script env.
		{"unsupported slice yields empty", []any{"a"}, ""},
		{"unsupported map yields empty", map[string]any{"a": 1}, ""},
		{"unsupported uint yields empty", uint(5), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scalarString(tt.in); got != tt.want {
				t.Errorf("scalarString(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestParseActionInputs_UnsupportedDefaultTypeDropped ties the
// scalarString default branch to Defaults(): a list-valued default
// normalises to "" and so is filtered out entirely rather than reaching
// the script as an empty env var.
//
// The drop itself is kept — unlike `required`, a lost default degrades
// to "input not supplied", which the script's own fallback or a
// required marker already covers, so failing the exec over it would
// break working adapters for a milder problem. What changes is that the
// drop is now reported instead of silent.
func TestParseActionInputs_UnsupportedDefaultTypeDropped(t *testing.T) {
	manifest := manifestFromYAML(t, `name: probe
type: messenger
config:
  actions:
  - name: list
    script: backends/list.sh
    input:
      - name: folders
        default: [INBOX, Sent]
      - name: limit
        default: 10
`)

	schemas, diags := parseActionSchemas(manifest)
	list, _ := findActionSchema(schemas, "list")
	if got, want := list.Defaults(), map[string]string{"limit": "10"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Defaults() = %v, want %v", got, want)
	}
	folders, ok := list.FindInput("folders")
	if !ok {
		t.Fatal(`input "folders" should still be declared`)
	}
	if folders.Default != "" {
		t.Errorf("unsupported default should normalise to empty; got %q",
			folders.Default)
	}
	// Consequence: no PROBE_FOLDERS is synthesised.
	if _, ok := applyInputDefaults(list, map[string]string{})["folders"]; ok {
		t.Error("an unrenderable default must not be applied")
	}

	// The drop is reported, and only for the value that could not be
	// rendered — `limit: 10` normalises fine and says nothing.
	if len(diags) != 1 {
		t.Fatalf("want exactly one diagnostic; got %v", diags)
	}
	for _, want := range []string{
		`action "list"`, `input "folders"`, "unsupported type", "dropping it",
	} {
		if !strings.Contains(diags[0], want) {
			t.Errorf("diagnostic %q missing %q", diags[0], want)
		}
	}
}

// TestParseActionInputs_EmptyDefaultIsNotADiagnostic: `default: ""` is
// a legitimate declaration of nothing, not an unrenderable value, so it
// must not be reported alongside the genuine drops.
func TestParseActionInputs_EmptyDefaultIsNotADiagnostic(t *testing.T) {
	manifest := manifestFromYAML(t, `name: probe
type: messenger
config:
  actions:
  - name: list
    script: backends/list.sh
    input:
      - name: query
        default: ""
      - name: folder
`)

	schemas, diags := parseActionSchemas(manifest)
	if len(diags) != 0 {
		t.Errorf("empty and absent defaults should report nothing; got %v", diags)
	}
	list, _ := findActionSchema(schemas, "list")
	if got := list.Defaults(); got != nil {
		t.Errorf("Defaults() = %v, want nil", got)
	}
}
