// Package adapter loads adapter definitions and runs their actions.
package adapter

import (
	"fmt"
	"strconv"
	"strings"
)

// ActionInput is a single declared input of a manifest action.
//
// Mirrors one entry of an action's `input` list:
//
//	input:
//	  - name: limit
//	    required: false
//	    default: "10"
//	    description: Max envelopes to return
type ActionInput struct {
	Name        string `json:"name" yaml:"name"`
	Required    bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ActionSchema is the typed view of one manifest action entry: its name,
// its raw (untemplated) script path, and its declared inputs.
//
// Parsing is descriptive only — nothing in the exec path currently
// validates caller-supplied inputs against Inputs.
type ActionSchema struct {
	Name        string        `json:"name" yaml:"name"`
	Script      string        `json:"script,omitempty" yaml:"script,omitempty"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	Inputs      []ActionInput `json:"input,omitempty" yaml:"input,omitempty"`
}

// FindInput returns the declared input with the given name.
//
// Parsing drops duplicate declarations, so at most one entry can match
// and every accessor on this type resolves the same name identically.
func (s *ActionSchema) FindInput(name string) (ActionInput, bool) {
	if s == nil {
		return ActionInput{}, false
	}
	for _, in := range s.Inputs {
		if in.Name == name {
			return in, true
		}
	}
	return ActionInput{}, false
}

// RequiredInputs returns the names of inputs declared `required: true`,
// in manifest order.
func (s *ActionSchema) RequiredInputs() []string {
	if s == nil {
		return nil
	}
	var names []string
	for _, in := range s.Inputs {
		if in.Required {
			names = append(names, in.Name)
		}
	}
	return names
}

// Defaults returns declared name -> default value pairs for inputs that
// carry a non-empty default.
func (s *ActionSchema) Defaults() map[string]string {
	if s == nil {
		return nil
	}
	var defaults map[string]string
	for _, in := range s.Inputs {
		if in.Default == "" {
			continue
		}
		if defaults == nil {
			defaults = make(map[string]string, len(s.Inputs))
		}
		defaults[in.Name] = in.Default
	}
	return defaults
}

// parseActionSchemas extracts the typed action list from a manifest's
// untyped config, along with diagnostics describing anything the parse
// had to coerce or discard.
//
// Tolerant by design: a missing or malformed `actions` key yields nil,
// and entries that are not maps or carry no name are skipped rather
// than failing the whole parse. Callers that need to report a
// missing/malformed actions list keep their own checks.
//
// Tolerance is not silence. Structural problems inside a declared
// input — a `required:` that is not a YAML bool, a `default:` that is
// not a scalar, a name declared twice — are reported through the
// returned diagnostics so the exec path can surface them. Parsing
// itself never fails: a manifest whose actions block is unreadable
// disables enforcement rather than breaking an adapter that works.
func parseActionSchemas(manifest *AdapterManifest) ([]ActionSchema, []string) {
	if manifest == nil {
		return nil, nil
	}
	raw, ok := manifest.Config["actions"]
	if !ok {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, nil
	}

	var (
		schemas []ActionSchema
		diags   []string
	)
	for _, entry := range list {
		aMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := aMap["name"].(string)
		if name == "" {
			continue
		}
		script, _ := aMap["script"].(string)
		description, _ := aMap["description"].(string)
		inputs, inputDiags := parseActionInputs(name, aMap["input"])
		diags = append(diags, inputDiags...)
		schemas = append(schemas, ActionSchema{
			Name:        name,
			Script:      script,
			Description: description,
			Inputs:      inputs,
		})
	}
	return schemas, diags
}

// parseActionInputs converts an action's raw `input` value into typed
// entries, plus diagnostics for values it had to coerce or discard.
// A missing or non-list value means "no declared inputs" rather than an
// error; entries that are not maps or carry no name are skipped.
//
// Duplicate names: the FIRST declaration wins and later ones are
// dropped. Chosen over last-wins because it is the rule the reading
// accessors already implied (FindInput scans in order), and because
// dropping at parse time is what makes FindInput, RequiredInputs and
// Defaults agree — a duplicate that survived parsing would let two call
// sites read two different answers from one manifest.
//
// action names the enclosing action so a diagnostic points at the
// manifest entry to edit.
func parseActionInputs(action string, raw any) ([]ActionInput, []string) {
	list, ok := raw.([]any)
	if !ok {
		return nil, nil
	}

	var (
		inputs []ActionInput
		diags  []string
		seen   map[string]struct{}
	)
	for _, entry := range list {
		iMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := iMap["name"].(string)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			diags = append(diags, fmt.Sprintf(
				"action %q: input %q declared more than once; "+
					"keeping the first declaration and ignoring the rest",
				action, name))
			continue
		}
		if seen == nil {
			seen = make(map[string]struct{}, len(list))
		}
		seen[name] = struct{}{}

		required, reqDiag := parseRequiredMarker(action, name, iMap["required"])
		if reqDiag != "" {
			diags = append(diags, reqDiag)
		}
		def, defDiag := parseDefaultValue(action, name, iMap["default"])
		if defDiag != "" {
			diags = append(diags, defDiag)
		}

		description, _ := iMap["description"].(string)
		inputs = append(inputs, ActionInput{
			Name:        name,
			Required:    required,
			Default:     def,
			Description: description,
		})
	}
	return inputs, diags
}

// parseRequiredMarker reads an input's `required:` value.
//
// An absent marker means optional — the manifest said nothing, so
// nothing is asserted. Anything present that is not a YAML bool is a
// manifest bug, and it resolves to REQUIRED rather than optional.
//
// The asymmetry is deliberate. `required` is the only safety marker in
// the schema, and the two failure directions are not equal: resolving
// an unreadable marker to optional silently retires the requirement and
// lets a call through that the author meant to block, with nothing
// visible at any layer. Resolving it to required is loud — the next
// call that omits the input is rejected by name, which is exactly the
// nudge that gets the manifest fixed. Quoting is the common case
// (`required: "true"`, next to a quoted `default: "10"`) and the author
// there meant required; a genuinely malformed value (`required: yes`,
// `required: 1`) has no defensible reading, so it fails closed too. In
// both cases the diagnostic names the value and the accepted spelling.
func parseRequiredMarker(action, name string, raw any) (bool, string) {
	switch v := raw.(type) {
	case nil:
		return false, ""
	case bool:
		return v, ""
	default:
		return true, fmt.Sprintf(
			"action %q: input %q has non-boolean required: %v (%T); "+
				"treating the input as REQUIRED — write an unquoted "+
				"`required: true` or `required: false`",
			action, name, raw, raw)
	}
}

// parseDefaultValue renders an input's `default:` as the string the
// script env needs, reporting values scalarString cannot render.
//
// Failing open is tolerable here in a way it is not for `required`: a
// dropped default degrades to "input not supplied", which the script's
// own fallback or a required marker already covers. So the value is
// still dropped — but no longer silently, since the alternative is an
// operator reading a default in the manifest that never reaches the
// script.
//
// An explicitly empty default (`default: ""`) is a legitimate
// declaration of nothing and draws no diagnostic.
func parseDefaultValue(action, name string, raw any) (string, string) {
	rendered := scalarString(raw)
	if rendered != "" || raw == nil {
		return rendered, ""
	}
	if s, ok := raw.(string); ok && s == "" {
		return "", ""
	}
	return "", fmt.Sprintf(
		"action %q: input %q has a default of unsupported type %T; "+
			"dropping it — declare a string, number or boolean scalar",
		action, name, raw)
}

// findActionSchema returns the schema for the named action.
func findActionSchema(schemas []ActionSchema, action string) (*ActionSchema, bool) {
	for i := range schemas {
		if schemas[i].Name == action {
			return &schemas[i], true
		}
	}
	return nil, false
}

// checkRequiredInputs rejects a call that omits inputs the action
// declares `required: true`. Reports every missing name at once, in
// manifest order, so the caller fixes the whole invocation in one pass.
//
// Enforced against what the CALLER supplied: a declared `default:` is a
// convenience for optional inputs and never satisfies `required: true`.
// A nil schema (action absent or manifest unparseable) declares nothing
// and so rejects nothing.
func checkRequiredInputs(
	schema *ActionSchema,
	action string,
	inputs map[string]string,
) error {
	var missing []string
	for _, name := range schema.RequiredInputs() {
		if _, ok := inputs[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"missing required input '%s' for action '%s': expected '--input %s=<value>'",
		strings.Join(missing, "', '"), action,
		missing[0])
}

// scalarString renders a YAML scalar as a string. Manifest authors write
// defaults unquoted (`default: 10`, `default: true`), so yaml.v3 hands
// back int/bool/float as often as string.
func scalarString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return ""
	}
}
