// Command servicemd renders service alias and adapter-catalogue doc
// fragments from the internal/core alias maps and doc metadata. Invoked by
// the cog markers embedded in docs/MESSENGER_SETUP_QUICK_REF.md,
// docs/dev/messenger-architecture.md, and docs/user/tickets.md (see
// `make docs-gen`); never run in production paths.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"hop.top/aps/internal/core"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// Fragment names accepted on the command line, one per generated table.
const (
	fragmentMessageAliases    = "message-aliases"
	fragmentTicketAliases     = "ticket-aliases"
	fragmentPersistedAdapters = "persisted-adapters"
	fragmentTicketAdapters    = "ticket-adapters"
)

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintf(stderr, "usage: servicemd %s|%s|%s|%s\n",
			fragmentMessageAliases, fragmentTicketAliases, fragmentPersistedAdapters, fragmentTicketAdapters)
		return 2
	}
	out, err := render(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	_, _ = fmt.Fprint(stdout, out)
	return 0
}

func render(fragment string) (string, error) {
	switch fragment {
	case fragmentMessageAliases:
		return messageAliases(), nil
	case fragmentTicketAliases:
		return ticketAliases(), nil
	case fragmentPersistedAdapters:
		return persistedAdapters(), nil
	case fragmentTicketAdapters:
		return ticketAdapters()
	default:
		return "", fmt.Errorf("unknown fragment %q", fragment)
	}
}

// aliasRow pairs a service type alias with the adapter it resolves.
type aliasRow struct {
	alias   string
	adapter string
}

// aliasesFor returns the aliases resolving to the given canonical service
// type, sorted by alias name.
func aliasesFor(canonicalType string) []aliasRow {
	aliases := core.ServiceTypeAliases()
	rows := make([]aliasRow, 0, len(aliases))
	for alias, target := range aliases {
		typ, adapter, ok := strings.Cut(target, " ")
		if !ok || typ != canonicalType {
			continue
		}
		rows = append(rows, aliasRow{alias: alias, adapter: adapter})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].alias < rows[j].alias })
	return rows
}

// messageAliases renders the quick-ref message adapter alias table: one row
// per known message adapter, "(none)" when the adapter has no alias.
func messageAliases() string {
	aliasByAdapter := make(map[string]string)
	for _, row := range aliasesFor("message") {
		aliasByAdapter[row.adapter] = row.alias
	}
	var b strings.Builder
	b.WriteString("| Alias | Canonical config |\n")
	b.WriteString("| --- | --- |\n")
	for _, adapter := range core.MessageAdapterCatalogue() {
		aliasCell := "(none)"
		if alias, ok := aliasByAdapter[adapter]; ok {
			aliasCell = "`" + alias + "`"
		}
		config := fmt.Sprintf("`type: message`, `adapter: %s`", adapter)
		if note := core.MessageAdapterDocNote(adapter); note != "" {
			config += " " + note
		}
		fmt.Fprintf(&b, "| %s | %s |\n", aliasCell, config)
	}
	return b.String()
}

// ticketAliases renders the quick-ref ticket alias table.
func ticketAliases() string {
	var b strings.Builder
	b.WriteString("| Alias | Canonical config |\n")
	b.WriteString("| --- | --- |\n")
	for _, row := range aliasesFor("ticket") {
		config := fmt.Sprintf("`type: ticket`, `adapter: %s`", row.adapter)
		if note := core.TicketAliasDocNote(row.alias); note != "" {
			config += " " + note
		}
		fmt.Fprintf(&b, "| `%s` | %s |\n", row.alias, config)
	}
	return b.String()
}

// persistedAdapters renders the messenger-architecture alias persistence
// table: the explicit --type/--adapter form first, then one row per message
// alias.
func persistedAdapters() string {
	var b strings.Builder
	b.WriteString("| Input | Persisted type | Persisted adapter |\n")
	b.WriteString("| --- | --- | --- |\n")
	b.WriteString("| `--type message --adapter telegram` | `message` | `telegram` |\n")
	for _, row := range aliasesFor("message") {
		fmt.Fprintf(&b, "| `--type %s` | `message` | `%s` |\n", row.alias, row.adapter)
	}
	return b.String()
}

// ticketAdapters renders the docs/user/tickets.md adapter table from the
// ticket aliases plus their doc judgment metadata.
func ticketAdapters() (string, error) {
	var b strings.Builder
	b.WriteString("| Adapter alias | Canonical service | Inbound payload | Maturity |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range aliasesFor("ticket") {
		meta, ok := core.TicketAdapterDocMetaFor(row.alias)
		if !ok {
			return "", fmt.Errorf("ticket alias %q has no doc metadata row", row.alias)
		}
		fmt.Fprintf(&b, "| `%s` | `type: ticket`, `adapter: %s` | %s | %s |\n",
			row.alias, row.adapter, meta.Inbound, meta.Maturity)
	}
	return b.String(), nil
}
