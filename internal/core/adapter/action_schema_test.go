package adapter

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// manifestFromYAML writes src to a temp manifest and loads it, so the
// parsing tests operate on the same untyped shapes yaml.v3 produces at
// runtime rather than on hand-built map literals.
func manifestFromYAML(t *testing.T, src string) *AdapterManifest {
	t.Helper()
	path := filepath.Join(t.TempDir(), ManifestFileName)
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	return manifest
}

// schemasOf parses and discards diagnostics, for the tests whose
// subject is the parsed shape rather than what parsing reported.
func schemasOf(manifest *AdapterManifest) []ActionSchema {
	schemas, _ := parseActionSchemas(manifest)
	return schemas
}

// diagsOf parses and keeps only the diagnostics.
func diagsOf(manifest *AdapterManifest) []string {
	_, diags := parseActionSchemas(manifest)
	return diags
}

func TestParseActionSchemas_RequiredInputs(t *testing.T) {
	manifest := manifestFromYAML(t, `name: email
type: messenger
config:
  actions:
  - name: send
    description: Send a new email
    script: backends/{{backend}}/send.sh
    input:
      - name: to
        required: true
      - name: subject
        required: true
      - name: body
        required: true
      - name: cc
        required: false
`)

	schemas, diags := parseActionSchemas(manifest)
	if len(schemas) != 1 {
		t.Fatalf("want 1 action, got %d: %+v", len(schemas), schemas)
	}
	// A well-formed input block reports nothing.
	if len(diags) != 0 {
		t.Errorf("well-formed manifest produced diagnostics: %v", diags)
	}

	send, ok := findActionSchema(schemas, "send")
	if !ok {
		t.Fatal(`action "send" not parsed`)
	}
	if send.Script != "backends/{{backend}}/send.sh" {
		t.Errorf("Script = %q; templating must stay raw", send.Script)
	}
	if send.Description != "Send a new email" {
		t.Errorf("Description = %q", send.Description)
	}
	if got, want := len(send.Inputs), 4; got != want {
		t.Fatalf("Inputs len = %d, want %d", got, want)
	}

	wantRequired := []string{"to", "subject", "body"}
	if got := send.RequiredInputs(); !reflect.DeepEqual(got, wantRequired) {
		t.Errorf("RequiredInputs() = %v, want %v", got, wantRequired)
	}
	if got := send.Defaults(); len(got) != 0 {
		t.Errorf("Defaults() = %v, want empty", got)
	}

	cc, ok := send.FindInput("cc")
	if !ok {
		t.Fatal(`input "cc" not found`)
	}
	if cc.Required {
		t.Error(`input "cc" should not be required`)
	}
}

func TestParseActionSchemas_Defaults(t *testing.T) {
	manifest := manifestFromYAML(t, `name: email
type: messenger
config:
  actions:
  - name: list
    script: backends/list.sh
    input:
      - name: limit
        required: false
        default: "10"
      - name: folder
        required: false
        default: INBOX
`)

	list, ok := findActionSchema(schemasOf(manifest), "list")
	if !ok {
		t.Fatal(`action "list" not parsed`)
	}
	if got := list.RequiredInputs(); got != nil {
		t.Errorf("RequiredInputs() = %v, want nil", got)
	}
	want := map[string]string{"limit": "10", "folder": "INBOX"}
	if got := list.Defaults(); !reflect.DeepEqual(got, want) {
		t.Errorf("Defaults() = %v, want %v", got, want)
	}
}

func TestParseActionSchemas_UnquotedScalarDefaults(t *testing.T) {
	manifest := manifestFromYAML(t, `name: email
type: messenger
config:
  actions:
  - name: list
    script: backends/list.sh
    input:
      - name: limit
        default: 10
      - name: ratio
        default: 1.5
      - name: verbose
        default: true
      - name: quiet
        default: false
`)

	list, _ := findActionSchema(schemasOf(manifest), "list")
	want := map[string]string{
		"limit":   "10",
		"ratio":   "1.5",
		"verbose": "true",
		"quiet":   "false",
	}
	got := list.Defaults()
	// `quiet` renders as "false" but is a non-empty default, so it must
	// survive the Defaults() filter.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Defaults() = %v, want %v", got, want)
	}
}

func TestParseActionSchemas_MixedRequiredAndDefaults(t *testing.T) {
	manifest := manifestFromYAML(t, `name: email
type: messenger
config:
  actions:
  - name: reply
    script: backends/reply.sh
    input:
      - name: id
        required: true
        description: Envelope ID to reply to
      - name: body
        required: true
      - name: folder
        required: false
        default: INBOX
`)

	reply, ok := findActionSchema(schemasOf(manifest), "reply")
	if !ok {
		t.Fatal(`action "reply" not parsed`)
	}
	if got, want := reply.RequiredInputs(), []string{"id", "body"}; !reflect.DeepEqual(got, want) {
		t.Errorf("RequiredInputs() = %v, want %v", got, want)
	}
	if got, want := reply.Defaults(), map[string]string{"folder": "INBOX"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Defaults() = %v, want %v", got, want)
	}
	id, _ := reply.FindInput("id")
	if id.Description != "Envelope ID to reply to" {
		t.Errorf("Description = %q", id.Description)
	}
}

func TestParseActionSchemas_NoInputKey(t *testing.T) {
	manifest := manifestFromYAML(t, `name: email
type: messenger
config:
  actions:
  - name: sync
    script: backends/sync.sh
`)

	sync, ok := findActionSchema(schemasOf(manifest), "sync")
	if !ok {
		t.Fatal(`action "sync" not parsed`)
	}
	if sync.Inputs != nil {
		t.Errorf("Inputs = %v, want nil for an action with no input list", sync.Inputs)
	}
	if got := sync.RequiredInputs(); got != nil {
		t.Errorf("RequiredInputs() = %v, want nil", got)
	}
	if got := sync.Defaults(); got != nil {
		t.Errorf("Defaults() = %v, want nil", got)
	}
	if _, ok := sync.FindInput("anything"); ok {
		t.Error("FindInput should miss on an action with no inputs")
	}
}

func TestParseActionSchemas_Malformed(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want []ActionSchema
		// wantDiags are substrings every returned diagnostic set must
		// contain, one per expected diagnostic. Empty means the parse
		// must report nothing.
		wantDiags []string
	}{
		{
			name: "no config at all",
			yaml: "name: email\ntype: messenger\n",
			want: nil,
		},
		{
			name: "config without actions key",
			yaml: "name: email\ntype: messenger\nconfig:\n  backend: himalaya\n",
			want: nil,
		},
		{
			name: "actions is a map not a list",
			yaml: "name: email\ntype: messenger\nconfig:\n  actions:\n    send: backends/send.sh\n",
			want: nil,
		},
		{
			name: "actions is a scalar",
			yaml: "name: email\ntype: messenger\nconfig:\n  actions: nope\n",
			want: nil,
		},
		{
			name: "entries that are scalars or unnamed are skipped",
			yaml: `name: email
type: messenger
config:
  actions:
  - just-a-string
  - script: backends/orphan.sh
  - name: ok
    script: backends/ok.sh
`,
			want: []ActionSchema{{Name: "ok", Script: "backends/ok.sh"}},
		},
		{
			name: "input is a scalar means no declared inputs",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    script: backends/ok.sh
    input: to
`,
			want: []ActionSchema{{Name: "ok", Script: "backends/ok.sh"}},
		},
		{
			name: "input is a map means no declared inputs",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    script: backends/ok.sh
    input:
      to: true
`,
			want: []ActionSchema{{Name: "ok", Script: "backends/ok.sh"}},
		},
		{
			// Nameless and non-map entries are still skipped silently:
			// there is no input there to say anything about. But a named
			// input whose `required:` cannot be read fails CLOSED, and
			// says so — the one thing it must not do is quietly become
			// optional.
			name: "malformed input entries are skipped; unreadable required fails closed",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    script: backends/ok.sh
    input:
      - bare-string
      - required: true
      - name: ""
        required: true
      - name: to
        required: yes-not-a-bool
`,
			want: []ActionSchema{{
				Name:   "ok",
				Script: "backends/ok.sh",
				Inputs: []ActionInput{{Name: "to", Required: true}},
			}},
			wantDiags: []string{
				`input "to" has non-boolean required: yes-not-a-bool (string); treating the input as REQUIRED`,
			},
		},
		{
			// The quoted-scalar case that motivated the strictness: an
			// author mirroring a nearby quoted `default: "10"` writes
			// `required: "true"` and must not silently lose the marker.
			name: "quoted required marker is honoured, not dropped",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    script: backends/ok.sh
    input:
      - name: to
        required: "true"
      - name: cc
        required: "false"
      - name: bcc
        required: 1
`,
			want: []ActionSchema{{
				Name:   "ok",
				Script: "backends/ok.sh",
				Inputs: []ActionInput{
					{Name: "to", Required: true},
					// "false" is not a bool either. It resolves the same
					// way every unreadable marker does — closed — rather
					// than being special-cased into a string-to-bool
					// parse that would re-open the exact hole this
					// closes for `required: "true"`.
					{Name: "cc", Required: true},
					{Name: "bcc", Required: true},
				},
			}},
			wantDiags: []string{
				`input "to" has non-boolean required: true (string)`,
				`input "cc" has non-boolean required: false (string)`,
				`input "bcc" has non-boolean required: 1 (int)`,
			},
		},
		{
			// First-wins duplicate collapse, reported.
			name: "duplicate input names collapse to the first declaration",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    script: backends/ok.sh
    input:
      - name: folder
        default: INBOX
        description: first
      - name: folder
        default: Sent
        description: second
`,
			want: []ActionSchema{{
				Name:   "ok",
				Script: "backends/ok.sh",
				Inputs: []ActionInput{
					{Name: "folder", Default: "INBOX", Description: "first"},
				},
			}},
			wantDiags: []string{
				`input "folder" declared more than once; keeping the first`,
			},
		},
		{
			// Unrenderable defaults are still dropped — but no longer
			// without a word.
			name: "unsupported default type is dropped with a diagnostic",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    script: backends/ok.sh
    input:
      - name: folders
        default: [INBOX, Sent]
      - name: empty
        default: ""
`,
			want: []ActionSchema{{
				Name:   "ok",
				Script: "backends/ok.sh",
				Inputs: []ActionInput{{Name: "folders"}, {Name: "empty"}},
			}},
			wantDiags: []string{
				`input "folders" has a default of unsupported type []interface {}; dropping it`,
			},
		},
		{
			name: "action without script still parses",
			yaml: `name: email
type: messenger
config:
  actions:
  - name: ok
    input:
      - name: to
        required: true
`,
			want: []ActionSchema{{
				Name:   "ok",
				Inputs: []ActionInput{{Name: "to", Required: true}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := parseActionSchemas(manifestFromYAML(t, tt.yaml))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseActionSchemas() = %+v, want %+v", got, tt.want)
			}
			if len(diags) != len(tt.wantDiags) {
				t.Fatalf("got %d diagnostics, want %d:\ngot:  %v\nwant: %v",
					len(diags), len(tt.wantDiags), diags, tt.wantDiags)
			}
			for i, want := range tt.wantDiags {
				if !strings.Contains(diags[i], want) {
					t.Errorf("diagnostic %d = %q, want it to contain %q",
						i, diags[i], want)
				}
			}
		})
	}
}

func TestParseActionSchemas_NilManifest(t *testing.T) {
	got, diags := parseActionSchemas(nil)
	if got != nil {
		t.Fatalf("parseActionSchemas(nil) = %v, want nil", got)
	}
	if diags != nil {
		t.Fatalf("parseActionSchemas(nil) diagnostics = %v, want nil", diags)
	}
}

func TestActionSchemaAccessors_NilReceiver(t *testing.T) {
	var s *ActionSchema
	if _, ok := s.FindInput("to"); ok {
		t.Error("FindInput on nil receiver should report miss")
	}
	if got := s.RequiredInputs(); got != nil {
		t.Errorf("RequiredInputs() = %v, want nil", got)
	}
	if got := s.Defaults(); got != nil {
		t.Errorf("Defaults() = %v, want nil", got)
	}
}

func TestFindActionSchema(t *testing.T) {
	schemas := []ActionSchema{
		{Name: "send"},
		{Name: "list", Inputs: []ActionInput{{Name: "limit", Default: "10"}}},
	}

	got, ok := findActionSchema(schemas, "list")
	if !ok {
		t.Fatal(`findActionSchema("list") reported a miss`)
	}
	if got.Name != "list" || len(got.Inputs) != 1 {
		t.Fatalf("wrong schema returned: %+v", got)
	}
	if _, ok := findActionSchema(schemas, "archive"); ok {
		t.Error(`findActionSchema("archive") should report a miss`)
	}
	if _, ok := findActionSchema(nil, "send"); ok {
		t.Error("findActionSchema(nil, ...) should report a miss")
	}
}

// TestParseActionSchemas_RealEmailManifest guards the parser against the
// shipped adapter manifest, which is the shape the exec path actually
// meets at runtime.
func TestParseActionSchemas_RealEmailManifest(t *testing.T) {
	path := filepath.Join("..", "..", "..", "adapters", "email", ManifestFileName)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("email adapter manifest not present: %v", err)
	}
	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	schemas, diags := parseActionSchemas(manifest)
	if len(schemas) != 4 {
		t.Fatalf("want 4 actions, got %d", len(schemas))
	}
	if len(diags) != 0 {
		t.Errorf("shipped manifest should parse cleanly; got %v", diags)
	}

	send, ok := findActionSchema(schemas, "send")
	if !ok {
		t.Fatal(`action "send" not parsed`)
	}
	if got, want := send.RequiredInputs(), []string{"to", "subject", "body"}; !reflect.DeepEqual(got, want) {
		t.Errorf("send RequiredInputs() = %v, want %v", got, want)
	}

	list, ok := findActionSchema(schemas, "list")
	if !ok {
		t.Fatal(`action "list" not parsed`)
	}
	want := map[string]string{"limit": "10", "folder": "INBOX"}
	if got := list.Defaults(); !reflect.DeepEqual(got, want) {
		t.Errorf("list Defaults() = %v, want %v", got, want)
	}
}

// TestParseActionSchemas_ShippedManifestsAreClean is the regression
// guard for the strictness added here: every manifest that ships in the
// repo must parse without a single diagnostic. A shipped manifest that
// trips the new checks would print warnings on every exec, so the
// checks and the manifests have to stay in agreement.
func TestParseActionSchemas_ShippedManifestsAreClean(t *testing.T) {
	root := filepath.Join("..", "..", "..", "adapters")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("adapters dir not present: %v", err)
	}

	var checked int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(root, e.Name(), ManifestFileName)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			manifest, err := LoadManifest(path)
			if err != nil {
				t.Fatalf("LoadManifest: %v", err)
			}
			schemas, diags := parseActionSchemas(manifest)
			if len(schemas) == 0 {
				t.Fatal("shipped manifest declares no actions")
			}
			for _, d := range diags {
				t.Errorf("shipped manifest is not clean: %s", d)
			}
		})
		checked++
	}
	if checked == 0 {
		t.Fatal("no shipped manifests found to check")
	}
}

func TestCheckRequiredInputs_AllSupplied(t *testing.T) {
	schema := &ActionSchema{
		Name: "send",
		Inputs: []ActionInput{
			{Name: "to", Required: true},
			{Name: "subject", Required: true},
			{Name: "cc"},
		},
	}
	inputs := map[string]string{"to": "u@example.com", "subject": "hi"}
	if err := checkRequiredInputs(schema, "send", inputs); err != nil {
		t.Fatalf("all required inputs supplied should pass; got %v", err)
	}
}

func TestCheckRequiredInputs_EmptyValueSatisfies(t *testing.T) {
	// parseInputs treats "cc=" as a present key with an empty value, so
	// presence — not emptiness — is what `required` asks about.
	schema := &ActionSchema{
		Name:   "send",
		Inputs: []ActionInput{{Name: "to", Required: true}},
	}
	if err := checkRequiredInputs(schema, "send", map[string]string{"to": ""}); err != nil {
		t.Fatalf("explicitly empty value should satisfy required; got %v", err)
	}
}

func TestCheckRequiredInputs_ReportsEveryMissingInManifestOrder(t *testing.T) {
	schema := &ActionSchema{
		Name: "send",
		Inputs: []ActionInput{
			{Name: "to", Required: true},
			{Name: "subject", Required: true},
			{Name: "body", Required: true},
			{Name: "cc"},
		},
	}

	err := checkRequiredInputs(schema, "send", nil)
	if err == nil {
		t.Fatal("missing required inputs should be rejected")
	}

	msg := err.Error()
	for _, want := range []string{"to", "subject", "body"} {
		if !strings.Contains(msg, "'"+want+"'") {
			t.Errorf("error does not name missing input %q: %s", want, msg)
		}
	}
	if !strings.Contains(msg, "'send'") {
		t.Errorf("error does not name the action: %s", msg)
	}

	// Manifest order, not map iteration order.
	iTo := strings.Index(msg, "to")
	iSubject := strings.Index(msg, "subject")
	iBody := strings.Index(msg, "body")
	if !(iTo < iSubject && iSubject < iBody) {
		t.Errorf("missing inputs not reported in manifest order: %s", msg)
	}

	// The optional input is not dragged into the diagnostic.
	if strings.Contains(msg, "cc") {
		t.Errorf("error names a non-required input: %s", msg)
	}
}

func TestCheckRequiredInputs_PartiallySupplied(t *testing.T) {
	schema := &ActionSchema{
		Name: "send",
		Inputs: []ActionInput{
			{Name: "to", Required: true},
			{Name: "subject", Required: true},
			{Name: "body", Required: true},
		},
	}

	err := checkRequiredInputs(schema, "send", map[string]string{"subject": "hi"})
	if err == nil {
		t.Fatal("partially supplied required inputs should be rejected")
	}
	msg := err.Error()
	if !strings.Contains(msg, "'to'") || !strings.Contains(msg, "'body'") {
		t.Errorf("error should name both still-missing inputs: %s", msg)
	}
	if strings.Contains(msg, "'subject'") {
		t.Errorf("error names a supplied input: %s", msg)
	}
}

func TestCheckRequiredInputs_DefaultDoesNotSatisfyRequired(t *testing.T) {
	// A `default:` is a convenience for optional inputs. Enforcement
	// runs against what the caller supplied, so a default on a
	// `required: true` input must not excuse its absence.
	schema := &ActionSchema{
		Name:   "send",
		Inputs: []ActionInput{{Name: "to", Required: true, Default: "ops@example.com"}},
	}
	if err := checkRequiredInputs(schema, "send", nil); err == nil {
		t.Fatal("a declared default must not satisfy required: true")
	}
}

func TestCheckRequiredInputs_NilSchema(t *testing.T) {
	// An unparseable manifest or an absent action declares nothing, so
	// it must neither panic nor spuriously reject.
	if err := checkRequiredInputs(nil, "send", nil); err != nil {
		t.Fatalf("nil schema should not reject; got %v", err)
	}
	if err := checkRequiredInputs(nil, "send", map[string]string{"to": "x"}); err != nil {
		t.Fatalf("nil schema should not reject; got %v", err)
	}
}

func TestCheckRequiredInputs_NoRequiredInputs(t *testing.T) {
	schema := &ActionSchema{
		Name: "list",
		Inputs: []ActionInput{
			{Name: "limit", Default: "10"},
			{Name: "query"},
		},
	}
	if err := checkRequiredInputs(schema, "list", nil); err != nil {
		t.Fatalf("action with no required inputs should pass; got %v", err)
	}
}
