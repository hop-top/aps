package squad

import (
	"fmt"
	"path/filepath"
)

// ScopeRule defines what resources a squad can access.
type ScopeRule struct {
	FilePatterns []string `json:"file_patterns,omitempty" yaml:"file_patterns,omitempty"`
	Operations   []string `json:"operations,omitempty" yaml:"operations,omitempty"`
	Tools        []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	Secrets      []string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Networks     []string `json:"networks,omitempty" yaml:"networks,omitempty"`
}

// Scope binds a ScopeRule to a specific squad.
type Scope struct {
	SquadID string    `json:"squad_id" yaml:"squad_id"`
	Rules   ScopeRule `json:"rules" yaml:"rules"`
}

// Validate checks required fields.
func (s *Scope) Validate() error {
	if s.SquadID == "" {
		return fmt.Errorf("scope squad ID is required")
	}
	return nil
}

// IntersectScopes returns the most restrictive ScopeRule by intersecting all
// provided scopes. For each slice field, empty means "all allowed"; when
// multiple scopes specify values, only items present in ALL scopes are kept.
func IntersectScopes(scopes ...Scope) ScopeRule {
	if len(scopes) == 0 {
		return ScopeRule{}
	}
	if len(scopes) == 1 {
		return scopes[0].Rules
	}

	return ScopeRule{
		FilePatterns: intersectField(scopes, func(r ScopeRule) []string { return r.FilePatterns }),
		Operations:   intersectField(scopes, func(r ScopeRule) []string { return r.Operations }),
		Tools:        intersectField(scopes, func(r ScopeRule) []string { return r.Tools }),
		Secrets:      intersectField(scopes, func(r ScopeRule) []string { return r.Secrets }),
		Networks:     intersectField(scopes, func(r ScopeRule) []string { return r.Networks }),
	}
}

// intersectField computes the intersection of a single slice field across scopes.
// Empty slices are treated as unrestricted (allow all).
func intersectField(scopes []Scope, extract func(ScopeRule) []string) []string {
	var result []string
	initialized := false

	for _, s := range scopes {
		vals := extract(s.Rules)
		if len(vals) == 0 {
			continue // unrestricted, skip
		}
		if !initialized {
			result = make([]string, len(vals))
			copy(result, vals)
			initialized = true
			continue
		}
		result = filterIntersect(result, vals)
	}
	return result
}

// filterIntersect keeps only items from a that also appear in b.
func filterIntersect(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	var out []string
	for _, v := range a {
		if _, ok := set[v]; ok {
			out = append(out, v)
		}
	}
	return out
}

// AllowsFile checks if the given path matches any of the file patterns.
// An empty FilePatterns slice means unrestricted (all files allowed).
func (r *ScopeRule) AllowsFile(path string) bool {
	if len(r.FilePatterns) == 0 {
		return true
	}
	for _, pattern := range r.FilePatterns {
		if matched, err := filepath.Match(pattern, path); err == nil && matched {
			return true
		}
	}
	return false
}

// AllowsTool checks if the given tool is in the allowed Tools slice.
// An empty Tools slice means unrestricted (all tools allowed).
func (r *ScopeRule) AllowsTool(tool string) bool {
	if len(r.Tools) == 0 {
		return true
	}
	for _, t := range r.Tools {
		if t == tool {
			return true
		}
	}
	return false
}

// AllowsOperation checks if the given operation is in the allowed Operations slice.
// An empty Operations slice means unrestricted (all operations allowed).
func (r *ScopeRule) AllowsOperation(op string) bool {
	if len(r.Operations) == 0 {
		return true
	}
	for _, o := range r.Operations {
		if o == op {
			return true
		}
	}
	return false
}
