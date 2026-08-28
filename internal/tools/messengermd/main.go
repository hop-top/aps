// Command messengermd renders messenger doc-table fragments from the
// platform metadata in internal/core/messenger and the live provider
// capability metadata in internal/adapters/messenger. Invoked by the cog
// markers embedded in the messenger docs (docs/MESSENGERS_OVERVIEW.md,
// docs/MESSENGER_SETUP_QUICK_REF.md, docs/user/messengers.md,
// docs/dev/messenger-architecture.md, docs/agent/messenger-patterns.md);
// never run in production paths.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	adapters "hop.top/aps/internal/adapters/messenger"
	messenger "hop.top/aps/internal/core/messenger"
)

// Fragment names, one per doc table the renderer replaces.
const (
	fragmentOverviewNav      = "overview-nav"
	fragmentQuickrefAliases  = "quickref-aliases"
	fragmentUserSupport      = "user-support"
	fragmentArchSupport      = "arch-support"
	fragmentPatternsTable    = "patterns-table"
	fragmentCapabilityMatrix = "capability-matrix"
)

const usage = "usage: messengermd " + fragmentOverviewNav + "|" + fragmentQuickrefAliases +
	"|" + fragmentUserSupport + "|" + fragmentArchSupport + "|" + fragmentPatternsTable +
	"|" + fragmentCapabilityMatrix

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, usage)
		return 2
	}
	fragment, err := render(args[0])
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	_, _ = fmt.Fprint(stdout, fragment)
	return 0
}

func render(name string) (string, error) {
	switch name {
	case fragmentOverviewNav:
		return overviewNav(), nil
	case fragmentQuickrefAliases:
		return quickrefAliases(), nil
	case fragmentUserSupport:
		return userSupport(), nil
	case fragmentArchSupport:
		return archSupport(), nil
	case fragmentPatternsTable:
		return patternsTable(), nil
	case fragmentCapabilityMatrix:
		return capabilityMatrix(), nil
	default:
		return "", fmt.Errorf("unknown fragment %q", name)
	}
}

// messageAdapters returns every platform that participates in the message
// runtime, in the deterministic AllPlatformMeta order: aliased message
// adapters plus alias-less ones that still ingest messages (email).
// Ticket-only platforms (github) carry no message ingress and are excluded.
func messageAdapters() []messenger.PlatformMeta {
	var rows []messenger.PlatformMeta
	for _, row := range messenger.AllPlatformMeta() {
		if row.Alias != "" || row.Ingress != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

// aliasedAdapters returns only the platforms addressable through a
// `aps service add --type <alias>` message alias.
func aliasedAdapters() []messenger.PlatformMeta {
	var rows []messenger.PlatformMeta
	for _, row := range messenger.AllPlatformMeta() {
		if row.Alias != "" {
			rows = append(rows, row)
		}
	}
	return rows
}

// firstClassProviders maps each platform to its first-class provider.
// Zero-value configs are safe here: Metadata() is static per provider and
// never touches transports.
func firstClassProviders() map[messenger.MessengerPlatform]messenger.MessageProvider {
	return map[messenger.MessengerPlatform]messenger.MessageProvider{
		messenger.PlatformTelegram: adapters.NewTelegramProvider(adapters.TelegramProviderConfig{}),
		messenger.PlatformSlack:    adapters.NewSlackProvider(adapters.SlackProviderConfig{}),
		messenger.PlatformTeams:    adapters.NewTeamsProvider(adapters.TeamsProviderConfig{}),
		messenger.PlatformDiscord:  adapters.NewDiscordProvider(adapters.DiscordProviderConfig{}),
		messenger.PlatformSMS:      messenger.NewSMSProvider(messenger.SMSProviderConfig{}, nil),
		messenger.PlatformWhatsApp: messenger.NewWhatsAppProvider(messenger.WhatsAppProviderConfig{}, nil),
	}
}

// aliasCell renders the alias column: the alias in backticks, or the
// canonical addressing form for alias-less message adapters.
func aliasCell(row messenger.PlatformMeta) string {
	if row.Alias == "" {
		return fmt.Sprintf("`--type message --adapter %s`", row.Platform)
	}
	return "`" + row.Alias + "`"
}

func canonicalConfig(row messenger.PlatformMeta) string {
	return fmt.Sprintf("`type: message`, `adapter: %s`", row.Platform)
}

const emptyCell = "—"

func orEmpty(value string) string {
	if value == "" {
		return emptyCell
	}
	return value
}

func codeList(items []string) string {
	if len(items) == 0 {
		return emptyCell
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = "`" + item + "`"
	}
	return strings.Join(quoted, ", ")
}

func ingressModeList(modes []messenger.IngressMode) string {
	items := make([]string, len(modes))
	for i, mode := range modes {
		items[i] = string(mode)
	}
	return codeList(items)
}

func deliveryModeList(modes []messenger.DeliveryMode) string {
	items := make([]string, len(modes))
	for i, mode := range modes {
		items[i] = string(mode)
	}
	return codeList(items)
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}
	return "No"
}

func table(b *strings.Builder, columns ...string) {
	b.WriteString("| " + strings.Join(columns, " | ") + " |\n")
	b.WriteString("|" + strings.Repeat(" --- |", len(columns)) + "\n")
}

func rowLine(b *strings.Builder, cells ...string) {
	b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
}

// overviewNav renders the docs/MESSENGERS_OVERVIEW.md "Quick Navigation"
// table body.
func overviewNav() string {
	var b strings.Builder
	table(&b, "Platform", "Alias", "Channel control", "Current ingress", "Signature validation")
	for _, row := range messageAdapters() {
		rowLine(&b, row.Display, aliasCell(row), row.ChannelControl, row.Ingress, row.Signature)
	}
	return b.String()
}

// quickrefAliases renders the docs/MESSENGER_SETUP_QUICK_REF.md message
// adapter alias table. Ticket aliases live in a separate table owned by
// the ticket docs.
func quickrefAliases() string {
	var b strings.Builder
	table(&b, "Alias", "Canonical config")
	for _, row := range messageAdapters() {
		if row.Alias == "" {
			rowLine(&b, "(none)",
				fmt.Sprintf("%s -- pass `--type message --adapter %s`", canonicalConfig(row), row.Platform))
			continue
		}
		rowLine(&b, "`"+row.Alias+"`", canonicalConfig(row))
	}
	return b.String()
}

// userSupport renders the docs/user/messengers.md "Supported Message
// Adapters" table body.
func userSupport() string {
	var b strings.Builder
	table(&b, "Adapter alias", "Channel ID format", "Typical token source", "Current support")
	for _, row := range messageAdapters() {
		rowLine(&b, aliasCell(row), row.ChannelID, row.AuthSource, row.Support)
	}
	return b.String()
}

// archSupport renders the docs/dev/messenger-architecture.md "Adapter
// Support" table body.
func archSupport() string {
	var b strings.Builder
	table(&b, "Adapter", "Normalize support", "Denormalize support", "Service maturity")
	for _, row := range aliasedAdapters() {
		rowLine(&b, row.Display, row.Normalize, row.Reply, row.Maturity)
	}
	return b.String()
}

// patternsTable renders the docs/agent/messenger-patterns.md "Supported
// Message Adapters" table body, one row per aliased platform.
func patternsTable() string {
	var b strings.Builder
	table(&b, "Adapter alias", "Canonical config", "Incoming payload support", "Reply shape", "Notes")
	for _, row := range aliasedAdapters() {
		rowLine(&b, "`"+row.Alias+"`", canonicalConfig(row), row.Normalize, row.Reply, orEmpty(row.Notes))
	}
	return b.String()
}

// capabilityMatrix renders the capability matrix sourced live from each
// first-class provider's Metadata(). Platforms without a first-class
// provider render only the documented Support string from the platform
// metadata; capability facts are never invented for them.
func capabilityMatrix() string {
	var b strings.Builder
	table(&b, "Platform", "Ingress modes", "Delivery modes", "Threads", "Attachments", "Reactions")
	providers := firstClassProviders()
	for _, row := range messageAdapters() {
		provider, ok := providers[row.Platform]
		if !ok {
			rowLine(&b, row.Display, row.Support, emptyCell, emptyCell, emptyCell, emptyCell)
			continue
		}
		md := provider.Metadata()
		rowLine(&b, row.Display,
			ingressModeList(md.IngressModes), deliveryModeList(md.DeliveryModes),
			yesNo(md.SupportsThreads), yesNo(md.SupportsAttachments), yesNo(md.SupportsReactions))
	}
	return b.String()
}
