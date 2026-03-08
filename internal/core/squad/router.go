package squad

import "fmt"

// RouteRequest describes a routing request.
// Spec ref §173-183.
type RouteRequest struct {
	Domain       string          `json:"domain" yaml:"domain"`
	RequiredType SquadType       `json:"required_type,omitempty" yaml:"required_type,omitempty"`
	RequiredMode InteractionMode `json:"required_mode,omitempty" yaml:"required_mode,omitempty"`
}

// RouteResult is a squad matched by the router with a relevance score.
type RouteResult struct {
	Squad Squad
	Score int
}

// Router classifies and routes by squad type, not just capability.
type Router struct {
	manager *Manager
}

// NewRouter creates a Router backed by the given Manager.
func NewRouter(mgr *Manager) *Router {
	return &Router{manager: mgr}
}

// Route finds squads matching the request, scored by type alignment.
// Domain is required — rejects capability-only routing per spec.
func (r *Router) Route(req RouteRequest) ([]RouteResult, error) {
	if req.Domain == "" {
		return nil, fmt.Errorf("domain is required for routing")
	}

	var results []RouteResult
	for _, s := range r.manager.List() {
		if req.RequiredType != "" && s.Type != req.RequiredType {
			continue
		}
		if !s.matchesDomain(req.Domain) {
			continue
		}

		score := 10 // domain match base
		switch s.Type {
		case SquadTypeStream:
			score += 5
		case SquadTypeSubsystem:
			score += 3
		case SquadTypePlatform:
			score += 2
		}

		results = append(results, RouteResult{Squad: s, Score: score})
	}

	// Sort descending by score (simple insertion sort — small N)
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	return results, nil
}
