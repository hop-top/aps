package adapter

import "strconv"

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
// untyped config. Tolerant by design: a missing or malformed `actions`
// key yields nil, and entries that are not maps or carry no name are
// skipped rather than failing the whole parse. Callers that need to
// report a missing/malformed actions list keep their own checks.
func parseActionSchemas(manifest *AdapterManifest) []ActionSchema {
	if manifest == nil {
		return nil
	}
	raw, ok := manifest.Config["actions"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	var schemas []ActionSchema
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
		schemas = append(schemas, ActionSchema{
			Name:        name,
			Script:      script,
			Description: description,
			Inputs:      parseActionInputs(aMap["input"]),
		})
	}
	return schemas
}

// parseActionInputs converts an action's raw `input` value into typed
// entries. A missing, non-list, or malformed value means "no declared
// inputs" rather than an error.
func parseActionInputs(raw any) []ActionInput {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	var inputs []ActionInput
	for _, entry := range list {
		iMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := iMap["name"].(string)
		if name == "" {
			continue
		}
		required, _ := iMap["required"].(bool)
		description, _ := iMap["description"].(string)
		inputs = append(inputs, ActionInput{
			Name:        name,
			Required:    required,
			Default:     scalarString(iMap["default"]),
			Description: description,
		})
	}
	return inputs
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
