package org

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/core"
	coreorg "hop.top/aps/internal/core/org"
	kitcli "hop.top/kit/go/console/cli"
)

// Snapshot format names accepted by --snapshot-format. Deliberately a
// LOCAL flag, not kit's global --format: the global enumerates
// table|json|yaml, while snapshots add tree and mermaid — the same
// collision --manifest-format solved for `aps profile export`.
const (
	snapshotFormatJSON    = "json"
	snapshotFormatYAML    = "yaml"
	snapshotFormatTree    = "tree"
	snapshotFormatMermaid = "mermaid"
)

// Scope type names recorded in the snapshot document.
const (
	scopeAll   = "all"
	scopeRoot  = "root"
	scopeSquad = "squad"
)

// snapshotDoc is the json/yaml snapshot envelope.
type snapshotDoc struct {
	CapturedAt string         `json:"captured_at" yaml:"captured_at"`
	Scope      snapshotScope  `json:"scope"       yaml:"scope"`
	Nodes      []snapshotNode `json:"nodes"       yaml:"nodes"`
	Edges      []snapshotEdge `json:"edges"       yaml:"edges"`
}

// snapshotScope records which selection produced the snapshot.
type snapshotScope struct {
	Type  string `json:"type"            yaml:"type"`
	Root  string `json:"root,omitempty"  yaml:"root,omitempty"`
	Squad string `json:"squad,omitempty" yaml:"squad,omitempty"`
}

// snapshotNode is one profile in the snapshot, sorted by id.
type snapshotNode struct {
	ID          string   `json:"id"                   yaml:"id"`
	DisplayName string   `json:"display_name"         yaml:"display_name"`
	Type        string   `json:"type"                 yaml:"type"`
	ReportsTo   string   `json:"reports_to,omitempty" yaml:"reports_to,omitempty"`
	Channels    []string `json:"channels"             yaml:"channels"`
	Squads      []string `json:"squads,omitempty"     yaml:"squads,omitempty"`
}

// snapshotEdge is one manager→report link where both endpoints are in
// scope, sorted by manager then report.
type snapshotEdge struct {
	Manager string `json:"manager" yaml:"manager"`
	Report  string `json:"report"  yaml:"report"`
}

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture a point-in-time organigram",
		Long: `Capture a point-in-time organigram of the reporting
hierarchy. The scope defaults to every profile on disk (--all);
--root <profile-id> limits it to a profile and its transitive
reports, and --squad <id> to the profiles whose squads list contains
the given squad id. The scope flags are mutually exclusive.

The rendering is selected with the LOCAL --snapshot-format flag
(json|yaml|tree|mermaid, default tree); the global --format flag is
ignored here because snapshots support formats the global enum does
not. json/yaml emit a {captured_at, scope, nodes, edges} document
with nodes and edges sorted by id; tree emits an ASCII forest (roots
first, reports indented, humans marked); mermaid emits a flowchart TD
with manager --> report edges. tree and mermaid carry no timestamp,
so identical state yields identical bytes.

The global --output <file> flag writes the rendering atomically
(temp file + rename) instead of stdout ("-" or empty means stdout).
Cyclic reports_to data still renders — traversals
are visited-set guarded — but the command exits non-zero with a
cycle error.

Read-only: no profile state is mutated; --output only writes the
operator-directed local file. Idempotent.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rootID, _ := cmd.Flags().GetString("root")
			squadID, _ := cmd.Flags().GetString("squad")
			snapFormat, _ := cmd.Flags().GetString("snapshot-format")
			outPath, _ := cmd.Flags().GetString("output")
			return runSnapshot(rootID, squadID, snapFormat, outPath)
		},
	}

	cmd.Flags().Bool("all", false, "Snapshot every profile on disk (default scope)")
	cmd.Flags().String("root", "", "Snapshot a profile and its transitive reports")
	cmd.Flags().String("squad", "", "Snapshot the members of a squad")
	cmd.Flags().String("snapshot-format", snapshotFormatTree, "Snapshot rendering (json|yaml|tree|mermaid)")
	cmd.MarkFlagsMutuallyExclusive("all", "root", "squad")

	// Read-only: loads profile state; --output is operator-directed
	// local output, not managed state. Safely repeatable.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func runSnapshot(rootID, squadID, snapFormat, outPath string) error {
	profiles, _, err := loadAllProfiles()
	if err != nil {
		return err
	}

	scope, selected, err := selectScope(profiles, rootID, squadID)
	if err != nil {
		return err
	}

	var rendered string
	switch snapFormat {
	case snapshotFormatJSON, snapshotFormatYAML:
		rendered, err = renderSnapshotDoc(snapFormat, scope, selected)
	case snapshotFormatTree:
		rendered = renderSnapshotTree(selected)
	case snapshotFormatMermaid:
		rendered = renderSnapshotMermaid(selected)
	default:
		return fmt.Errorf("unknown snapshot format %q (supported: json, yaml, tree, mermaid)", snapFormat)
	}
	if err != nil {
		return err
	}

	if outPath != "" && outPath != "-" {
		if err := writeFileAtomic(outPath, []byte(rendered)); err != nil {
			return err
		}
	} else if _, err := os.Stdout.WriteString(rendered); err != nil {
		return fmt.Errorf("writing snapshot: %w", err)
	}

	return scopeCycleError(selected)
}

// selectScope resolves the scope flags into the sorted profile
// selection the snapshot covers. Exactly one scope applies; the
// mutual-exclusion of the flags is enforced by cobra.
func selectScope(profiles []core.Profile, rootID, squadID string) (snapshotScope, []core.Profile, error) {
	switch {
	case rootID != "":
		g := coreorg.Build(profiles)
		subtree := g.TransitiveReports(rootID, 0)
		found := false
		for _, p := range profiles {
			if p.ID == rootID {
				subtree = append(subtree, p)
				found = true
				break
			}
		}
		if !found {
			return snapshotScope{}, nil, fmt.Errorf("unknown profile %q for --root", rootID)
		}
		sortProfiles(subtree)
		return snapshotScope{Type: scopeRoot, Root: rootID}, subtree, nil
	case squadID != "":
		var members []core.Profile
		for _, p := range profiles {
			if p.IsMemberOfSquad(squadID) {
				members = append(members, p)
			}
		}
		if len(members) == 0 {
			return snapshotScope{}, nil, fmt.Errorf("no profiles are members of squad %q", squadID)
		}
		sortProfiles(members)
		return snapshotScope{Type: scopeSquad, Squad: squadID}, members, nil
	default:
		sortProfiles(profiles)
		return snapshotScope{Type: scopeAll}, profiles, nil
	}
}

// sortProfiles orders profiles by id in place.
func sortProfiles(profiles []core.Profile) {
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].ID < profiles[j].ID })
}

// scopeCycleError returns a non-nil error when the selected profiles
// contain a reporting cycle (including self-references), so every
// snapshot format exits non-zero on cyclic data after emitting its
// (finite) rendering.
func scopeCycleError(selected []core.Profile) error {
	for _, f := range coreorg.Build(selected).Validate() {
		if f.Kind == coreorg.FindingCycle || f.Kind == coreorg.FindingSelfReference {
			return fmt.Errorf("organigram contains a reporting cycle: %s", f.Message)
		}
	}
	return nil
}

// snapshotEdges derives the manager→report edges among the selected
// profiles. Edges whose manager is out of scope are dropped — the
// dangling reference is `aps org check`'s finding, not the
// snapshot's. selected must be sorted by id, which makes the result
// sorted by (manager, report) after the final sort.
func snapshotEdges(selected []core.Profile) []snapshotEdge {
	inScope := make(map[string]bool, len(selected))
	for _, p := range selected {
		inScope[p.ID] = true
	}
	edges := []snapshotEdge{}
	for _, p := range selected {
		if p.ReportsTo != "" && inScope[p.ReportsTo] {
			edges = append(edges, snapshotEdge{Manager: p.ReportsTo, Report: p.ID})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Manager != edges[j].Manager {
			return edges[i].Manager < edges[j].Manager
		}
		return edges[i].Report < edges[j].Report
	})
	return edges
}

// renderSnapshotDoc renders the json/yaml snapshot envelope. Only
// captured_at varies between runs over identical state.
func renderSnapshotDoc(snapFormat string, scope snapshotScope, selected []core.Profile) (string, error) {
	nodes := make([]snapshotNode, 0, len(selected))
	for _, p := range selected {
		nodes = append(nodes, snapshotNode{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			Type:        p.EffectiveType(),
			ReportsTo:   p.ReportsTo,
			Channels:    coreorg.Channels(p),
			Squads:      p.Squads,
		})
	}
	doc := snapshotDoc{
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Scope:      scope,
		Nodes:      nodes,
		Edges:      snapshotEdges(selected),
	}
	if snapFormat == snapshotFormatJSON {
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshaling snapshot: %w", err)
		}
		return string(data) + "\n", nil
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshaling snapshot: %w", err)
	}
	return string(data), nil
}

// renderSnapshotTree renders the selection as an ASCII forest: roots
// first (profiles whose manager is empty or out of scope), reports
// indented two spaces per level, one `display name (id)` per line
// with a `(human)` marker for human profiles. Cyclic members have no
// in-scope root; they render in a trailing visited-set-guarded pass
// so the walk always terminates.
func renderSnapshotTree(selected []core.Profile) string {
	byID := make(map[string]core.Profile, len(selected))
	inScope := make(map[string]bool, len(selected))
	for _, p := range selected {
		byID[p.ID] = p
		inScope[p.ID] = true
	}
	children := make(map[string][]string, len(selected))
	for _, p := range selected { // sorted by id, so child lists are sorted
		if p.ReportsTo != "" && inScope[p.ReportsTo] {
			children[p.ReportsTo] = append(children[p.ReportsTo], p.ID)
		}
	}

	var b strings.Builder
	visited := make(map[string]bool, len(selected))
	var walk func(id string, depth int)
	walk = func(id string, depth int) {
		if visited[id] {
			return
		}
		visited[id] = true
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteString(treeLabel(byID[id]))
		b.WriteString("\n")
		for _, child := range children[id] {
			walk(child, depth+1)
		}
	}
	for _, p := range selected {
		if p.ReportsTo == "" || !inScope[p.ReportsTo] {
			walk(p.ID, 0)
		}
	}
	// Cycle members are reachable from no root; render them last.
	for _, p := range selected {
		walk(p.ID, 0)
	}
	return b.String()
}

// treeLabel formats one tree line: `display name (id)`, falling back
// to the id when the display name is empty, plus a human marker.
func treeLabel(p core.Profile) string {
	name := p.DisplayName
	if name == "" {
		name = p.ID
	}
	label := fmt.Sprintf("%s (%s)", name, p.ID)
	if p.EffectiveType() == core.ProfileTypeHuman {
		label += " (human)"
	}
	return label
}

// renderSnapshotMermaid renders the selection as a mermaid flowchart:
// one node declaration per profile (sorted by id, labels carry a
// ` (human)` suffix for humans), then one `manager --> report` line
// per in-scope edge. Edges are a finite set, so cyclic data renders
// without special handling.
func renderSnapshotMermaid(selected []core.Profile) string {
	ids := make([]string, 0, len(selected))
	for _, p := range selected {
		ids = append(ids, p.ID)
	}
	nodeIDs := mermaidNodeIDs(ids)

	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for _, p := range selected {
		name := p.DisplayName
		if name == "" {
			name = p.ID
		}
		if p.EffectiveType() == core.ProfileTypeHuman {
			name += " (human)"
		}
		fmt.Fprintf(&b, "    %s[%q]\n", nodeIDs[p.ID], name)
	}
	for _, e := range snapshotEdges(selected) {
		fmt.Fprintf(&b, "    %s --> %s\n", nodeIDs[e.Manager], nodeIDs[e.Report])
	}
	return b.String()
}

// mermaidNodeIDs sanitizes profile ids into mermaid-safe node ids:
// every non-alphanumeric rune becomes `_`, and ids that collide after
// sanitization get a numeric suffix (`_2`, `_3`, …) in sorted-input
// order so the mapping is deterministic.
func mermaidNodeIDs(ids []string) map[string]string {
	used := make(map[string]bool, len(ids))
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		sanitized := sanitizeMermaidID(id)
		candidate := sanitized
		for i := 2; used[candidate]; i++ {
			candidate = fmt.Sprintf("%s_%d", sanitized, i)
		}
		used[candidate] = true
		out[id] = candidate
	}
	return out
}

// sanitizeMermaidID maps every rune outside [a-zA-Z0-9] to `_`. An
// empty input yields "_" so the node id is never empty.
func sanitizeMermaidID(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}

// writeFileAtomic writes data to path via a temp file in the same
// directory followed by a rename, so readers never observe a partial
// snapshot.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".org-snapshot-*")
	if err != nil {
		return fmt.Errorf("creating temp snapshot file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing snapshot file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming snapshot into place: %w", err)
	}
	return nil
}
