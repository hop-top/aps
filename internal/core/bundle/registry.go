package bundle

import (
	"fmt"
)

// Registry holds the merged set of built-in and user-defined bundles.
type Registry struct {
	bundles []Bundle
}

// NewRegistry creates a Registry by merging built-in bundles with user overrides.
// User-defined bundles win over built-ins when both have the same name.
func NewRegistry() (*Registry, error) {
	builtins, err := LoadBuiltins()
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to load built-in bundles: %w", err)
	}

	overrides, err := LoadUserOverrides()
	if err != nil {
		return nil, fmt.Errorf("bundle: failed to load user bundle overrides: %w", err)
	}

	// Index built-ins by name.
	merged := make(map[string]Bundle, len(builtins)+len(overrides))
	for _, b := range builtins {
		merged[b.Name] = b
	}

	// User overrides replace built-ins with the same name.
	for _, b := range overrides {
		merged[b.Name] = b
	}

	// Collect into a slice; order is not guaranteed by map iteration,
	// but callers should not rely on ordering.
	result := make([]Bundle, 0, len(merged))
	for _, b := range merged {
		result = append(result, b)
	}

	return &Registry{bundles: result}, nil
}

// List returns all registered bundles.
func (r *Registry) List() []Bundle {
	out := make([]Bundle, len(r.bundles))
	copy(out, r.bundles)
	return out
}

// Get returns the bundle with the given name, or an error if not found.
func (r *Registry) Get(name string) (*Bundle, error) {
	for i := range r.bundles {
		if r.bundles[i].Name == name {
			b := r.bundles[i]
			return &b, nil
		}
	}
	return nil, fmt.Errorf("bundle %q not found", name)
}

// Validate checks that a bundle definition is well-formed.
//
// Rules enforced:
//   - name must not be empty
//   - each BinaryRequirement.Missing must be one of: "skip", "warn", "error", or ""
//   - each BinaryRequirement.DenyPolicy must be one of: "strip", "error", or ""
//   - if Extends is set, the parent must exist in the registry and must not itself extend another bundle (one-level depth only)
func (r *Registry) Validate(b *Bundle) error {
	if b.Name == "" {
		return fmt.Errorf("bundle: name is required")
	}

	validMissing := map[string]bool{
		"":      true,
		"skip":  true,
		"warn":  true,
		"error": true,
	}
	validDenyPolicy := map[string]bool{
		"":      true,
		"strip": true,
		"error": true,
	}

	for i, req := range b.Requires {
		if !validMissing[req.Missing] {
			return fmt.Errorf("bundle %q: requires[%d] (%s): invalid missing policy %q; must be one of: skip, warn, error", b.Name, i, req.Binary, req.Missing)
		}
		if !validDenyPolicy[req.DenyPolicy] {
			return fmt.Errorf("bundle %q: requires[%d] (%s): invalid deny_policy %q; must be one of: strip, error", b.Name, i, req.Binary, req.DenyPolicy)
		}
	}

	if b.Extends != "" {
		parent, err := r.Get(b.Extends)
		if err != nil {
			return fmt.Errorf("bundle %q: extends %q which does not exist in the registry", b.Name, b.Extends)
		}
		if parent.Extends != "" {
			return fmt.Errorf("bundle %q: extends %q which itself extends %q; inheritance is limited to one level", b.Name, b.Extends, parent.Extends)
		}
	}

	return nil
}
