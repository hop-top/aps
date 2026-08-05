package org

import (
	"fmt"
	"sort"
	"strings"

	"hop.top/aps/internal/core"
)

// FindingKind classifies a hierarchy consistency problem reported by
// Validate.
type FindingKind string

const (
	// FindingCycle is a reporting cycle of two or more profiles.
	FindingCycle FindingKind = "cycle"
	// FindingDanglingReportsTo is a reports_to pointing at a profile
	// that does not exist.
	FindingDanglingReportsTo FindingKind = "dangling_reports_to"
	// FindingSelfReference is a profile that reports to itself. Self
	// loops are reported under this kind only, never as FindingCycle.
	FindingSelfReference FindingKind = "self_reference"
	// FindingUnknownType is a type value outside the write-strict set
	// ("", "agent", "human"); read paths tolerate it, Validate flags it.
	FindingUnknownType FindingKind = "unknown_type"
)

// Finding is one hierarchy consistency problem. ProfileIDs holds the
// affected profile ids: the full cycle path for FindingCycle
// (canonicalized to start at the smallest id), a single id otherwise.
type Finding struct {
	Kind       FindingKind
	ProfileIDs []string
	Message    string
}

// Validate checks the whole graph for consistency problems: cycles,
// dangling reports_to refs, self-references and unknown type values.
// Findings are sorted by kind, then by affected profile ids, so output
// is deterministic regardless of input order.
func (g *Graph) Validate() []Finding {
	var findings []Finding
	for _, id := range g.ids {
		p := g.byID[id]
		switch p.ReportsTo {
		case "":
			// root, nothing to check
		case id:
			findings = append(findings, Finding{
				Kind:       FindingSelfReference,
				ProfileIDs: []string{id},
				Message:    fmt.Sprintf("profile %q reports to itself", id),
			})
		default:
			if _, ok := g.byID[p.ReportsTo]; !ok {
				findings = append(findings, Finding{
					Kind:       FindingDanglingReportsTo,
					ProfileIDs: []string{id},
					Message:    fmt.Sprintf("profile %q reports_to unknown profile %q", id, p.ReportsTo),
				})
			}
		}
		if err := p.ValidateType(); err != nil {
			findings = append(findings, Finding{
				Kind:       FindingUnknownType,
				ProfileIDs: []string{id},
				Message:    fmt.Sprintf("profile %q has unknown type %q (allowed: %q, %q or empty)", id, p.Type, core.ProfileTypeAgent, core.ProfileTypeHuman),
			})
		}
	}
	for _, cycle := range g.cycles() {
		findings = append(findings, Finding{
			Kind:       FindingCycle,
			ProfileIDs: cycle,
			Message:    fmt.Sprintf("reporting cycle: %s -> %s", strings.Join(cycle, " -> "), cycle[0]),
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return strings.Join(findings[i].ProfileIDs, "\x00") < strings.Join(findings[j].ProfileIDs, "\x00")
	})
	return findings
}

// cycles finds every reporting cycle of length >= 2 exactly once.
// Each cycle path is rotated to start at its smallest profile id so
// the result does not depend on where the walk entered the cycle.
func (g *Graph) cycles() [][]string {
	var out [][]string
	done := make(map[string]bool) // ids whose walk outcome is settled
	for _, start := range g.ids {
		if done[start] {
			continue
		}
		index := make(map[string]int) // id -> position in current path
		var path []string
		for cur := start; ; {
			p, ok := g.byID[cur]
			if !ok || done[cur] {
				break // dangling ref, root reached, or settled prefix
			}
			if at, seen := index[cur]; seen {
				if cycle := path[at:]; len(cycle) >= 2 {
					out = append(out, canonicalCycle(cycle))
				}
				break
			}
			index[cur] = len(path)
			path = append(path, cur)
			cur = p.ReportsTo
			if cur == "" {
				break
			}
		}
		for _, id := range path {
			done[id] = true
		}
	}
	return out
}

// canonicalCycle rotates a cycle path so its smallest id comes first.
func canonicalCycle(cycle []string) []string {
	smallest := 0
	for i, id := range cycle {
		if id < cycle[smallest] {
			smallest = i
		}
	}
	out := make([]string, 0, len(cycle))
	out = append(out, cycle[smallest:]...)
	out = append(out, cycle[:smallest]...)
	return out
}
