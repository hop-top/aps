// Command adaptermd renders adapter doc fragments from the meta
// tables in internal/core/adapter. Invoked by the cog markers
// embedded in README.md, docs/architecture.md, and
// docs/dev/adapters.md; never run in production paths.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"hop.top/aps/internal/core/adapter"
)

const usage = "usage: adaptermd kinds-table|strategies-table|kinds-inline|kinds-list"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	switch args[0] {
	case "kinds-table":
		kindsTable(stdout)
	case "strategies-table":
		strategiesTable(stdout)
	case "kinds-inline":
		kindsInline(stdout)
	case "kinds-list":
		kindsList(stdout)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown fragment %q\n%s\n", args[0], usage)
		return 2
	}
	return 0
}

// sortedKinds returns the AdapterTypes keys in alphabetical order so
// every fragment renders deterministically across runs.
func sortedKinds() []adapter.AdapterType {
	kinds := make([]adapter.AdapterType, 0, len(adapter.AdapterTypes))
	for k := range adapter.AdapterTypes {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

// sortedStrategies returns the LoadingStrategies keys in alphabetical
// order so the strategy table renders deterministically across runs.
func sortedStrategies() []adapter.LoadingStrategy {
	strategies := make([]adapter.LoadingStrategy, 0, len(adapter.LoadingStrategies))
	for s := range adapter.LoadingStrategies {
		strategies = append(strategies, s)
	}
	sort.Slice(strategies, func(i, j int) bool { return strategies[i] < strategies[j] })
	return strategies
}

// title uppercases the first byte of an adapter kind key; kind keys
// are single lowercase ASCII words, so no display string is needed.
func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// kindsTable emits the adapter kinds table for docs/dev/adapters.md.
func kindsTable(w io.Writer) {
	_, _ = fmt.Fprintln(w, "| Type | Key | Description |")
	_, _ = fmt.Fprintln(w, "| :--- | :--- | :--- |")
	for _, k := range sortedKinds() {
		meta := adapter.AdapterTypes[k]
		_, _ = fmt.Fprintf(w, "| **%s** | `%s` | %s |\n", title(string(k)), k, meta.Description)
	}
}

// strategiesTable emits the loading-strategy table for
// docs/dev/adapters.md.
func strategiesTable(w io.Writer) {
	_, _ = fmt.Fprintln(w, "| Strategy | Description | Persistence |")
	_, _ = fmt.Fprintln(w, "| :--- | :--- | :--- |")
	for _, s := range sortedStrategies() {
		meta := adapter.LoadingStrategies[s]
		_, _ = fmt.Fprintf(w, "| **%s** | %s | %s |\n", meta.Display, meta.Description, meta.Persistence)
	}
}

// kindsInline emits the parenthesized kind list embedded in
// docs/architecture.md.
func kindsInline(w io.Writer) {
	keys := make([]string, 0, len(adapter.AdapterTypes))
	for _, k := range sortedKinds() {
		keys = append(keys, string(k))
	}
	_, _ = fmt.Fprintf(w, "(%s)\n", strings.Join(keys, " / "))
}

// kindsList emits a README-style bullet list of adapter kinds.
func kindsList(w io.Writer) {
	for _, k := range sortedKinds() {
		meta := adapter.AdapterTypes[k]
		_, _ = fmt.Fprintf(w, "- **%s**: %s\n", title(string(k)), meta.Description)
	}
}
