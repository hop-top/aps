package scope

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Rule defines access boundaries across multiple dimensions.
type Rule struct {
	FilePatterns []string `json:"file_patterns,omitempty" yaml:"file_patterns,omitempty"`
	Operations   []string `json:"operations,omitempty" yaml:"operations,omitempty"`
	Tools        []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	Secrets      []string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Networks     []string `json:"networks,omitempty" yaml:"networks,omitempty"`
}

// Scope binds a Rule to an owner entity.
type Scope struct {
	OwnerType string `json:"owner_type" yaml:"owner_type"` // "profile", "squad", "workspace"
	OwnerID   string `json:"owner_id" yaml:"owner_id"`
	Rules     Rule   `json:"rules" yaml:"rules"`
}

// Validate checks required fields.
func (s *Scope) Validate() error {
	if s.OwnerType == "" {
		return fmt.Errorf("owner_type is required")
	}
	if s.OwnerID == "" {
		return fmt.Errorf("owner_id is required")
	}
	switch s.OwnerType {
	case "profile", "squad", "workspace":
	default:
		return fmt.Errorf("invalid owner_type: %q", s.OwnerType)
	}
	return nil
}

// AllowsFile returns true if path matches any file pattern (empty = unrestricted).
func (r *Rule) AllowsFile(path string) bool {
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

// AllowsTool checks if tool is in the allowed list (empty = unrestricted).
func (r *Rule) AllowsTool(tool string) bool {
	return allowsItem(r.Tools, tool)
}

// AllowsOperation checks if op is in the allowed list (empty = unrestricted).
func (r *Rule) AllowsOperation(op string) bool {
	return allowsItem(r.Operations, op)
}

// AllowsSecret checks if secret is in the allowed list (empty = unrestricted).
func (r *Rule) AllowsSecret(secret string) bool {
	return allowsItem(r.Secrets, secret)
}

// AllowsNetwork checks if network is in the allowed list (empty = unrestricted).
func (r *Rule) AllowsNetwork(network string) bool {
	return allowsItem(r.Networks, network)
}

func allowsItem(allowed []string, item string) bool {
	if len(allowed) == 0 {
		return true
	}
	lower := strings.ToLower(item)
	for _, a := range allowed {
		if strings.ToLower(a) == lower {
			return true
		}
	}
	return false
}

// Intersect returns the most restrictive combination of two rules.
func Intersect(a, b Rule) Rule {
	return Rule{
		FilePatterns: intersectSlices(a.FilePatterns, b.FilePatterns),
		Operations:   intersectSlices(a.Operations, b.Operations),
		Tools:        intersectSlices(a.Tools, b.Tools),
		Secrets:      intersectSlices(a.Secrets, b.Secrets),
		Networks:     intersectSlices(a.Networks, b.Networks),
	}
}

// IntersectAll returns the most restrictive combination of multiple rules.
func IntersectAll(rules ...Rule) Rule {
	if len(rules) == 0 {
		return Rule{}
	}
	result := rules[0]
	for _, r := range rules[1:] {
		result = Intersect(result, r)
	}
	return result
}

// intersectSlices returns the intersection of two slices using case-insensitive
// comparison. If either slice is empty (unrestricted), the other is returned.
func intersectSlices(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	set := make(map[string]struct{}, len(b))
	for _, v := range b {
		set[strings.ToLower(v)] = struct{}{}
	}
	var out []string
	for _, v := range a {
		if _, ok := set[strings.ToLower(v)]; ok {
			out = append(out, v)
		}
	}
	return out
}
