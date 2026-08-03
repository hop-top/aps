package adapter

import (
	"os"
	"path/filepath"
	"reflect"
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

	schemas := parseActionSchemas(manifest)
	if len(schemas) != 1 {
		t.Fatalf("want 1 action, got %d: %+v", len(schemas), schemas)
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

	list, ok := findActionSchema(parseActionSchemas(manifest), "list")
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

	list, _ := findActionSchema(parseActionSchemas(manifest), "list")
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

	reply, ok := findActionSchema(parseActionSchemas(manifest), "reply")
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

	sync, ok := findActionSchema(parseActionSchemas(manifest), "sync")
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
			name: "malformed input entries are skipped",
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
				Inputs: []ActionInput{{Name: "to"}},
			}},
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
			got := parseActionSchemas(manifestFromYAML(t, tt.yaml))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseActionSchemas() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseActionSchemas_NilManifest(t *testing.T) {
	if got := parseActionSchemas(nil); got != nil {
		t.Fatalf("parseActionSchemas(nil) = %v, want nil", got)
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

	schemas := parseActionSchemas(manifest)
	if len(schemas) != 4 {
		t.Fatalf("want 4 actions, got %d", len(schemas))
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
