// Package org provides a pure reporting-hierarchy graph over aps
// profiles. It performs no I/O: callers load profiles themselves (e.g.
// via core.ListProfilesFull) and hand them to Build. On-disk data is
// untrusted, so every traversal is visited-set guarded and terminates
// on cyclic input, and unknown ids yield empty results rather than
// panics. All returned slices are sorted by profile id so output is
// deterministic regardless of input order.
package org

import (
	"fmt"
	"sort"
	"strings"

	"hop.top/aps/internal/core"
)

// CycleError reports a reporting-hierarchy cycle encountered while
// walking reports_to links. Path holds the profile ids along the walk
// from the first repeated profile onward, in walk order.
type CycleError struct {
	Path []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("reporting cycle: %s", strings.Join(e.Path, " -> "))
}

// Graph is an immutable reporting-hierarchy view over a set of
// profiles. Construct it with Build; the zero value is not usable.
type Graph struct {
	byID    map[string]core.Profile
	reports map[string][]string // manager id -> sorted direct-report ids
	ids     []string            // all known ids, sorted
}

// Build constructs a Graph from the given profiles. Duplicate ids are
// resolved defensively: the last occurrence wins, mirroring how a
// later write would overwrite an earlier profile on disk. Input order
// otherwise does not matter — all accessors sort by profile id.
func Build(profiles []core.Profile) *Graph {
	g := &Graph{
		byID:    make(map[string]core.Profile, len(profiles)),
		reports: make(map[string][]string),
	}
	for _, p := range profiles {
		g.byID[p.ID] = p
	}
	g.ids = make([]string, 0, len(g.byID))
	for id := range g.byID {
		g.ids = append(g.ids, id)
	}
	sort.Strings(g.ids)
	for _, id := range g.ids {
		if mgr := g.byID[id].ReportsTo; mgr != "" {
			g.reports[mgr] = append(g.reports[mgr], id)
		}
	}
	// Children were appended in sorted id order, so each list is sorted.
	return g
}

// Manager returns the profile the given profile reports to. The
// second return is false when the id is unknown, the profile is a
// root, or its reports_to target does not exist (dangling ref).
func (g *Graph) Manager(id string) (core.Profile, bool) {
	p, ok := g.byID[id]
	if !ok || p.ReportsTo == "" {
		return core.Profile{}, false
	}
	mgr, ok := g.byID[p.ReportsTo]
	return mgr, ok
}

// Chain walks reports_to links from the given profile up to its root,
// returning the profiles in walk order starting with the profile
// itself. An unknown id yields an empty chain and no error. A dangling
// reports_to target ends the chain silently — Validate is the surface
// that flags it. On a cycle, Chain returns the partial chain (each
// profile visited exactly once) together with a *CycleError.
func (g *Graph) Chain(id string) ([]core.Profile, error) {
	var chain []core.Profile
	visited := make(map[string]bool)
	for cur := id; cur != ""; {
		p, ok := g.byID[cur]
		if !ok {
			return chain, nil // unknown start or dangling reports_to
		}
		if visited[cur] {
			return chain, &CycleError{Path: cyclePathFrom(chain, cur)}
		}
		visited[cur] = true
		chain = append(chain, p)
		cur = p.ReportsTo
	}
	return chain, nil
}

// cyclePathFrom extracts the ids of chain from the first occurrence of
// repeated onward — the actual cycle portion of the walk.
func cyclePathFrom(chain []core.Profile, repeated string) []string {
	var path []string
	for i, p := range chain {
		if p.ID == repeated {
			for _, q := range chain[i:] {
				path = append(path, q.ID)
			}
			break
		}
	}
	return path
}

// DirectReports returns the profiles that report directly to the given
// id, sorted by profile id. Unknown ids yield an empty slice.
func (g *Graph) DirectReports(id string) []core.Profile {
	children := g.reports[id]
	if len(children) == 0 {
		return nil
	}
	out := make([]core.Profile, 0, len(children))
	for _, c := range children {
		out = append(out, g.byID[c])
	}
	return out
}

// TransitiveReports returns every profile below the given id in the
// reporting hierarchy, up to depth levels down. depth <= 0 means
// unlimited. The starting profile is never included, even when cyclic
// data makes it its own transitive report. Results are sorted by
// profile id; unknown ids yield an empty slice.
func (g *Graph) TransitiveReports(id string, depth int) []core.Profile {
	if _, ok := g.byID[id]; !ok {
		return nil
	}
	visited := map[string]bool{id: true}
	var out []core.Profile
	frontier := []string{id}
	for level := 0; len(frontier) > 0 && (depth <= 0 || level < depth); level++ {
		var next []string
		for _, cur := range frontier {
			for _, child := range g.reports[cur] {
				if visited[child] {
					continue
				}
				visited[child] = true
				out = append(out, g.byID[child])
				next = append(next, child)
			}
		}
		frontier = next
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Roots returns the profiles with an empty reports_to, sorted by
// profile id. Profiles whose reports_to points at a missing profile
// are NOT roots — they surface as dangling-ref findings in Validate.
func (g *Graph) Roots() []core.Profile {
	var out []core.Profile
	for _, id := range g.ids {
		if p := g.byID[id]; p.ReportsTo == "" {
			out = append(out, p)
		}
	}
	return out
}
