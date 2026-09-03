// Command messengermd renders messenger doc-table fragments from the
// platform metadata in internal/core/messenger, the live provider
// capability metadata in internal/adapters/messenger, and the conversation
// store constants and turn shape in internal/core/messenger. Invoked by the
// cog markers embedded in the messenger docs (docs/MESSENGERS_OVERVIEW.md,
// docs/user/messengers.md, docs/user/conversations.md,
// docs/dev/messenger-architecture.md, docs/agent/messenger-patterns.md);
// never run in production paths.
package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	adapters "hop.top/aps/internal/adapters/messenger"
	messenger "hop.top/aps/internal/core/messenger"
)

// Fragment names, one per doc table the renderer replaces.
const (
	fragmentOverviewNav       = "overview-nav"
	fragmentUserSupport       = "user-support"
	fragmentArchSupport       = "arch-support"
	fragmentPatternsTable     = "patterns-table"
	fragmentCapabilityMatrix  = "capability-matrix"
	fragmentConversationStore = "conversation-store"
	fragmentConversationTurn  = "conversation-turn"
)

const usage = "usage: messengermd " + fragmentOverviewNav + "|" + fragmentUserSupport +
	"|" + fragmentArchSupport + "|" + fragmentPatternsTable + "|" + fragmentCapabilityMatrix +
	"|" + fragmentConversationStore + "|" + fragmentConversationTurn

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
	case fragmentUserSupport:
		return userSupport(), nil
	case fragmentArchSupport:
		return archSupport(), nil
	case fragmentPatternsTable:
		return patternsTable(), nil
	case fragmentCapabilityMatrix:
		return capabilityMatrix(), nil
	case fragmentConversationStore:
		return conversationStore(), nil
	case fragmentConversationTurn:
		return conversationTurn()
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

// conversationStore renders the docs/user/conversations.md store facts
// table from the conversation store constants: on-disk location, table,
// default prior-turn bound, and retention cap.
func conversationStore() string {
	location := strings.Join([]string{"<data-dir>", messenger.ConversationsDir, messenger.ConversationStoreFile}, "/")
	var b strings.Builder
	table(&b, "Fact", "Value")
	rowLine(&b, "Location", "`"+location+"`")
	rowLine(&b, "Table", "`"+messenger.ConversationTable+"`")
	rowLine(&b, "Prior turns per action run",
		fmt.Sprintf("`%d` by default; `--history-turns N` per service", messenger.DefaultPriorTurnLimit))
	rowLine(&b, "Turns kept per conversation",
		fmt.Sprintf("newest `%d`; older pruned on append", messenger.DefaultConversationRetention))
	return b.String()
}

// turnFieldMeanings documents each ConversationTurn JSON field. Keys must
// match the struct's JSON tags exactly; the test pins both directions so a
// new or renamed field cannot ship without a doc row.
var turnFieldMeanings = map[string]string{
	"seq":             "Append order in the store; ascending, unique per store",
	"conversation_id": "Outer place: service, platform, workspace, channel; plus sender for DMs and phone",
	"session_id":      "Conversation narrowed to a platform thread when the message carries one",
	"service_id":      "Service route that recorded the turn",
	"platform":        "Provider family, for example `sms` or `slack`",
	"profile_id":      "Profile whose action handled the message",
	"action_name":     "Action that ran",
	"direction": "`" + messenger.TurnDirectionInbound + "` (message received) or `" +
		messenger.TurnDirectionOutbound + "` (reply delivered)",
	"message_id":  "Platform message ID; outbound turns carry the inbound ID they answer",
	"channel_id":  "Platform channel, chat, or receiving number",
	"sender_id":   "Platform sender; the profile ID on outbound turns",
	"sender_name": "Sender display name when the platform provides one",
	"text":        "Message or reply text as routed; mentions and command prefixes kept",
	"attachments": "Normalized attachment metadata",
	"timestamp":   "Platform time for inbound turns, delivery time for outbound; UTC",
}

// turnField is one ConversationTurn field as seen through its JSON tag.
type turnField struct {
	Name     string
	Type     string
	Optional bool
}

// turnFields lists ConversationTurn's JSON fields in struct order.
func turnFields() ([]turnField, error) {
	rt := reflect.TypeOf(messenger.ConversationTurn{})
	fields := make([]turnField, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		docType, err := jsonType(f.Type)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}
		fields = append(fields, turnField{Name: name, Type: docType, Optional: strings.Contains(opts, "omitempty")})
	}
	return fields, nil
}

// jsonKindNames maps the Go kinds ConversationTurn uses to their JSON shape.
var jsonKindNames = map[reflect.Kind]string{
	reflect.Int:    "integer",
	reflect.Int64:  "integer",
	reflect.String: "string",
	reflect.Slice:  "array",
}

// jsonType names the JSON shape of a Go field type for the doc table.
func jsonType(t reflect.Type) (string, error) {
	if t == reflect.TypeOf(time.Time{}) {
		return "timestamp (RFC 3339, UTC)", nil
	}
	if name, ok := jsonKindNames[t.Kind()]; ok {
		return name, nil
	}
	return "", fmt.Errorf("unsupported field kind %s", t.Kind())
}

// conversationTurn renders the docs/user/conversations.md turn fields table
// from ConversationTurn's JSON tags; meanings come from turnFieldMeanings.
func conversationTurn() (string, error) {
	fields, err := turnFields()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	table(&b, "Field", "Type", "Optional", "Meaning")
	for _, f := range fields {
		meaning, ok := turnFieldMeanings[f.Name]
		if !ok {
			return "", fmt.Errorf("no documented meaning for turn field %q", f.Name)
		}
		rowLine(&b, "`"+f.Name+"`", f.Type, yesNo(f.Optional), meaning)
	}
	return b.String(), nil
}
