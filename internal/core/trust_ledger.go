package core

import "time"

// ValidRoles enumerates the recognized profile roles.
var ValidRoles = []string{"owner", "assignee", "evaluator", "auditor"}

// TrustDomains enumerates the recognized trust scoring domains.
var TrustDomains = []string{
	"hooks", "skills", "commands", "plugins", "general",
}

// TrustLedger holds trust scores and an append-only history for a profile.
type TrustLedger struct {
	Scores  map[string]float64 `yaml:"scores,omitempty"`  // domain → score
	History []TrustEntry       `yaml:"history,omitempty"` // append-only
}

// TrustEntry records a single trust-relevant event.
type TrustEntry struct {
	TaskRef    string            `yaml:"task_ref"`
	Domain     string            `yaml:"domain"`
	Difficulty string            `yaml:"difficulty,omitempty"` // XS, S, M, L, XL
	Timestamp  time.Time         `yaml:"timestamp"`
	Delta      float64           `yaml:"delta"`
	Breakdown  []TrustBreakdown  `yaml:"breakdown,omitempty"`
}

// TrustBreakdown provides sub-scoring detail within a TrustEntry.
type TrustBreakdown struct {
	Label string  `yaml:"label"`
	Value float64 `yaml:"value"`
}

// HasRole returns true if the profile has the given role.
func (p *Profile) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// AddRole adds a role to the profile (deduplicated).
func (p *Profile) AddRole(role string) {
	if !p.HasRole(role) {
		p.Roles = append(p.Roles, role)
	}
}

// RemoveRole removes a role from the profile.
func (p *Profile) RemoveRole(role string) {
	roles := make([]string, 0, len(p.Roles))
	for _, r := range p.Roles {
		if r != role {
			roles = append(roles, r)
		}
	}
	p.Roles = roles
}

// IsValidRole returns true if role is in ValidRoles.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// IsValidTrustDomain returns true if domain is in TrustDomains.
func IsValidTrustDomain(domain string) bool {
	for _, d := range TrustDomains {
		if d == domain {
			return true
		}
	}
	return false
}

// EnsureTrustLedger initialises the trust ledger if nil.
func (p *Profile) EnsureTrustLedger() {
	if p.TrustLedger == nil {
		p.TrustLedger = &TrustLedger{
			Scores: make(map[string]float64),
		}
	}
	if p.TrustLedger.Scores == nil {
		p.TrustLedger.Scores = make(map[string]float64)
	}
}

// RecordTrust appends a TrustEntry and updates the domain score.
func (p *Profile) RecordTrust(entry TrustEntry) {
	p.EnsureTrustLedger()
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	p.TrustLedger.History = append(p.TrustLedger.History, entry)
	p.TrustLedger.Scores[entry.Domain] += entry.Delta
}

// TrustScore returns the score for a domain (0 if unset).
func (p *Profile) TrustScore(domain string) float64 {
	if p.TrustLedger == nil || p.TrustLedger.Scores == nil {
		return 0
	}
	return p.TrustLedger.Scores[domain]
}

// TrustHistory returns history entries, optionally filtered by domain.
func (p *Profile) TrustHistory(domain string) []TrustEntry {
	if p.TrustLedger == nil {
		return nil
	}
	if domain == "" {
		return p.TrustLedger.History
	}
	var out []TrustEntry
	for _, e := range p.TrustLedger.History {
		if e.Domain == domain {
			out = append(out, e)
		}
	}
	return out
}
