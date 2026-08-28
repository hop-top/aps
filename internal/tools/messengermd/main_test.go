package main

import (
	"fmt"
	"strings"
	"testing"

	adapters "hop.top/aps/internal/adapters/messenger"
	messenger "hop.top/aps/internal/core/messenger"
)

var fragmentNames = []string{
	fragmentOverviewNav,
	fragmentQuickrefAliases,
	fragmentUserSupport,
	fragmentArchSupport,
	fragmentPatternsTable,
	fragmentCapabilityMatrix,
}

// metaRowCount counts platform metadata rows matching a predicate, so the
// expected fragment row counts are pinned to the driving data instead of
// literals.
func metaRowCount(match func(messenger.PlatformMeta) bool) int {
	count := 0
	for _, row := range messenger.AllPlatformMeta() {
		if match(row) {
			count++
		}
	}
	return count
}

// bodyRows returns the fragment's table body rows (header and separator
// stripped).
func bodyRows(t *testing.T, fragment string) []string {
	t.Helper()
	out, err := render(fragment)
	if err != nil {
		t.Fatalf("render(%q): %v", fragment, err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("render(%q) produced no table: %q", fragment, out)
	}
	if !strings.HasPrefix(lines[0], "| ") || !strings.Contains(lines[1], "---") {
		t.Fatalf("render(%q) missing header/separator: %q", fragment, lines[:2])
	}
	return lines[2:]
}

// cells splits one markdown table row into its cell values.
func cells(t *testing.T, row string) []string {
	t.Helper()
	trimmed := strings.TrimSuffix(strings.TrimPrefix(row, "| "), " |")
	return strings.Split(trimmed, " | ")
}

func TestFragmentRowCountsPinnedToMeta(t *testing.T) {
	isMessage := func(row messenger.PlatformMeta) bool { return row.Alias != "" || row.Ingress != "" }
	isAliased := func(row messenger.PlatformMeta) bool { return row.Alias != "" }

	wantRows := map[string]int{
		fragmentOverviewNav:      metaRowCount(isMessage),
		fragmentQuickrefAliases:  metaRowCount(isMessage),
		fragmentUserSupport:      metaRowCount(isMessage),
		fragmentCapabilityMatrix: metaRowCount(isMessage),
		fragmentArchSupport:      metaRowCount(isAliased),
		fragmentPatternsTable:    metaRowCount(isAliased),
	}
	for fragment, want := range wantRows {
		if got := len(bodyRows(t, fragment)); got != want {
			t.Errorf("%s has %d body rows, want %d", fragment, got, want)
		}
	}
}

func TestFragmentHeaders(t *testing.T) {
	wantHeaders := map[string]string{
		fragmentOverviewNav:      "| Platform | Alias | Channel control | Current ingress | Signature validation |",
		fragmentQuickrefAliases:  "| Alias | Canonical config |",
		fragmentUserSupport:      "| Adapter alias | Channel ID format | Typical token source | Current support |",
		fragmentArchSupport:      "| Adapter | Normalize support | Denormalize support | Service maturity |",
		fragmentPatternsTable:    "| Adapter alias | Canonical config | Incoming payload support | Reply shape | Notes |",
		fragmentCapabilityMatrix: "| Platform | Ingress modes | Delivery modes | Threads | Attachments | Reactions |",
	}
	for fragment, want := range wantHeaders {
		out, err := render(fragment)
		if err != nil {
			t.Fatalf("render(%q): %v", fragment, err)
		}
		if got := strings.SplitN(out, "\n", 2)[0]; got != want {
			t.Errorf("%s header = %q, want %q", fragment, got, want)
		}
	}
}

func TestFragmentOutputStable(t *testing.T) {
	for _, fragment := range fragmentNames {
		first, err := render(fragment)
		if err != nil {
			t.Fatalf("render(%q): %v", fragment, err)
		}
		second, err := render(fragment)
		if err != nil {
			t.Fatalf("render(%q) second pass: %v", fragment, err)
		}
		if first != second {
			t.Errorf("%s output is not stable across renders", fragment)
		}
	}
}

// TestPatternsTableIncludesTeams pins the drift fix: the patterns doc table
// historically lacked the teams row; the fragment must carry it.
func TestPatternsTableIncludesTeams(t *testing.T) {
	meta, ok := messenger.PlatformMetaFor(messenger.PlatformTeams)
	if !ok {
		t.Fatal("no platform meta for teams")
	}
	var teamsRow string
	for _, row := range bodyRows(t, fragmentPatternsTable) {
		if strings.HasPrefix(row, "| `"+meta.Alias+"` |") {
			teamsRow = row
			break
		}
	}
	if teamsRow == "" {
		t.Fatal("patterns-table has no teams row")
	}
	got := cells(t, teamsRow)
	want := []string{
		"`" + meta.Alias + "`",
		fmt.Sprintf("`type: message`, `adapter: %s`", meta.Platform),
		meta.Normalize,
		meta.Reply,
		emptyCell, // teams has no Notes in the platform meta
	}
	if len(got) != len(want) {
		t.Fatalf("teams row has %d cells, want %d: %q", len(got), len(want), teamsRow)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("teams row cell %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestCapabilityMatrixMatchesProviderMetadata checks every capability row
// against the live Metadata() of the platform's first-class provider, and
// that provider-less platforms only carry the documented Support string.
func TestCapabilityMatrixMatchesProviderMetadata(t *testing.T) {
	rowsByDisplay := map[string][]string{}
	for _, row := range bodyRows(t, fragmentCapabilityMatrix) {
		values := cells(t, row)
		if len(values) != 6 {
			t.Fatalf("capability row has %d cells, want 6: %q", len(values), row)
		}
		rowsByDisplay[values[0]] = values
	}

	providers := firstClassProviders()
	for _, meta := range messageAdapters() {
		values, ok := rowsByDisplay[meta.Display]
		if !ok {
			t.Errorf("capability-matrix has no row for %s", meta.Display)
			continue
		}
		provider, ok := providers[meta.Platform]
		if !ok {
			want := []string{meta.Display, meta.Support, emptyCell, emptyCell, emptyCell, emptyCell}
			for i := range want {
				if values[i] != want[i] {
					t.Errorf("%s (no first-class provider) cell %d = %q, want %q",
						meta.Display, i, values[i], want[i])
				}
			}
			continue
		}
		md := provider.Metadata()
		booleans := map[int]bool{3: md.SupportsThreads, 4: md.SupportsAttachments, 5: md.SupportsReactions}
		for i, capability := range booleans {
			want := "No"
			if capability {
				want = "Yes"
			}
			if values[i] != want {
				t.Errorf("%s capability cell %d = %q, want %q from provider Metadata()",
					meta.Display, i, values[i], want)
			}
		}
		for _, mode := range md.IngressModes {
			if !strings.Contains(values[1], "`"+string(mode)+"`") {
				t.Errorf("%s ingress cell %q missing mode %q", meta.Display, values[1], mode)
			}
		}
		for _, mode := range md.DeliveryModes {
			if !strings.Contains(values[2], "`"+string(mode)+"`") {
				t.Errorf("%s delivery cell %q missing mode %q", meta.Display, values[2], mode)
			}
		}
	}
}

// TestCapabilityMatrixTeamsRowLive asserts the teams row cell by cell
// against a freshly constructed TeamsProvider's Metadata().
func TestCapabilityMatrixTeamsRowLive(t *testing.T) {
	md := adapters.NewTeamsProvider(adapters.TeamsProviderConfig{}).Metadata()
	meta, ok := messenger.PlatformMetaFor(messenger.PlatformTeams)
	if !ok {
		t.Fatal("no platform meta for teams")
	}

	toCells := func(booleans ...bool) []string {
		out := make([]string, len(booleans))
		for i, value := range booleans {
			if value {
				out[i] = "Yes"
			} else {
				out[i] = "No"
			}
		}
		return out
	}
	capabilityCells := toCells(md.SupportsThreads, md.SupportsAttachments, md.SupportsReactions)

	var teamsRow string
	for _, row := range bodyRows(t, fragmentCapabilityMatrix) {
		if strings.HasPrefix(row, "| "+meta.Display+" |") {
			teamsRow = row
			break
		}
	}
	if teamsRow == "" {
		t.Fatal("capability-matrix has no Teams row")
	}
	values := cells(t, teamsRow)
	if len(values) != 6 {
		t.Fatalf("Teams row has %d cells, want 6: %q", len(values), teamsRow)
	}
	for i, want := range capabilityCells {
		if got := values[3+i]; got != want {
			t.Errorf("Teams capability cell %d = %q, want %q from TeamsProvider.Metadata()", 3+i, got, want)
		}
	}
	for _, mode := range md.IngressModes {
		if !strings.Contains(values[1], "`"+string(mode)+"`") {
			t.Errorf("Teams ingress cell %q missing mode %q", values[1], mode)
		}
	}
	for _, mode := range md.DeliveryModes {
		if !strings.Contains(values[2], "`"+string(mode)+"`") {
			t.Errorf("Teams delivery cell %q missing mode %q", values[2], mode)
		}
	}
}

func TestRunUnknownFragmentExitsNonZero(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := run([]string{"nope"}, &stdout, &stderr); code == 0 {
		t.Fatal("run with unknown fragment returned exit code 0")
	}
	if !strings.Contains(stderr.String(), "unknown fragment") {
		t.Errorf("stderr = %q, want unknown fragment error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunUsageExitsNonZero(t *testing.T) {
	for _, args := range [][]string{nil, {"overview-nav", "extra"}} {
		var stdout, stderr strings.Builder
		if code := run(args, &stdout, &stderr); code == 0 {
			t.Errorf("run(%q) returned exit code 0", args)
		}
		if !strings.Contains(stderr.String(), "usage:") {
			t.Errorf("run(%q) stderr = %q, want usage", args, stderr.String())
		}
	}
}

func TestRunRendersFragments(t *testing.T) {
	for _, fragment := range fragmentNames {
		var stdout, stderr strings.Builder
		if code := run([]string{fragment}, &stdout, &stderr); code != 0 {
			t.Fatalf("run(%q) exit code %d, stderr %q", fragment, code, stderr.String())
		}
		want, err := render(fragment)
		if err != nil {
			t.Fatalf("render(%q): %v", fragment, err)
		}
		if stdout.String() != want {
			t.Errorf("run(%q) stdout differs from render output", fragment)
		}
	}
}
