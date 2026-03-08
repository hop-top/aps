package scope

import (
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/bundle"
)

// ResolvedScope represents the effective scope after merging
// all applicable scopes. Most restrictive wins.
type ResolvedScope struct {
	ProfileScope   *Scope
	SquadScopes    []Scope
	WorkspaceScope *Scope
	Effective      Rule // final, most restrictive rule
}

// Resolve computes the effective scope for a profile, considering
// its own scope, squad scopes, and workspace scope.
func Resolve(profile *Scope, squads []Scope, workspace *Scope) *ResolvedScope {
	resolved := &ResolvedScope{
		ProfileScope:   profile,
		SquadScopes:    squads,
		WorkspaceScope: workspace,
	}

	var rules []Rule
	if profile != nil {
		rules = append(rules, profile.Rules)
	}
	for _, s := range squads {
		rules = append(rules, s.Rules)
	}
	if workspace != nil {
		rules = append(rules, workspace.Rules)
	}

	if len(rules) == 0 {
		resolved.Effective = Rule{}
		return resolved
	}

	resolved.Effective = IntersectAll(rules...)
	return resolved
}

// scopeFromConfig converts a core.ScopeConfig into a scope.Scope.
func scopeFromConfig(
	ownerType, ownerID string,
	cfg *core.ScopeConfig,
) *Scope {
	if cfg == nil {
		return nil
	}
	return &Scope{
		OwnerType: ownerType,
		OwnerID:   ownerID,
		Rules: Rule{
			FilePatterns: cfg.FilePatterns,
			Operations:   cfg.Operations,
			Tools:        cfg.Tools,
			Secrets:      cfg.Secrets,
			Networks:     cfg.Networks,
		},
	}
}

// ResolveForProfile loads a profile by ID and computes the
// effective scope from profile + bundle + workspace rules.
func ResolveForProfile(profileID string) (*ResolvedScope, error) {
	p, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, err
	}

	profileScope := scopeFromConfig("profile", profileID, p.Scope)

	var workspaceScope *Scope
	if p.Workspace != nil {
		workspaceScope = scopeFromConfig(
			"workspace", p.Workspace.Name,
			p.Workspace.ScopeRules,
		)
	}

	// T-0053 — Union-merge bundle scopes into the profile scope.
	resolved := Resolve(profileScope, nil, workspaceScope)
	resolved = mergeBundleScopes(resolved, p)
	return resolved, nil
}

// mergeBundleScopes resolves all bundles for the profile and union-merges their
// scope contributions into the ResolvedScope's effective rule.
func mergeBundleScopes(rs *ResolvedScope, p *core.Profile) *ResolvedScope {
	resolved, err := core.ResolveBundlesForProfile(p)
	if err != nil || len(resolved) == 0 {
		return rs
	}

	// Union-merge all bundle scopes into the effective rule.
	bundleOps := rs.Effective.Operations
	bundleFiles := rs.Effective.FilePatterns
	bundleNets := rs.Effective.Networks

	for _, rb := range resolved {
		bundleOps = bundleUnionStrings(bundleOps, rb.Scope.Operations)
		bundleFiles = bundleUnionStrings(bundleFiles, rb.Scope.FilePatterns)
		bundleNets = bundleUnionStrings(bundleNets, rb.Scope.Networks)
	}

	rs.Effective.Operations = bundleOps
	rs.Effective.FilePatterns = bundleFiles
	rs.Effective.Networks = bundleNets
	return rs
}

// bundleUnionStrings returns a deduplicated union of two string slices.
// This mirrors bundle.unionStrings without importing the unexported function.
func bundleUnionStrings(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(a)+len(b))
	var out []string
	for _, s := range append(a, b...) {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// BundleScopeToRule converts a bundle.BundleScope to a scope.Rule.
func BundleScopeToRule(bs bundle.BundleScope) Rule {
	return Rule{
		Operations:   bs.Operations,
		FilePatterns: bs.FilePatterns,
		Networks:     bs.Networks,
	}
}
