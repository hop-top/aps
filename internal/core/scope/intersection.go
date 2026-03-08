package scope

import "hop.top/aps/internal/core"

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
// effective scope from profile + workspace rules.
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

	// Squad scopes loaded from squad manager; not yet wired.
	return Resolve(profileScope, nil, workspaceScope), nil
}
