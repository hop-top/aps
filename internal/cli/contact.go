package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/globals"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	coreadapter "hop.top/aps/internal/core/adapter"
	kitcli "hop.top/kit/go/console/cli"
)

func init() {
	rootCmd.AddCommand(newContactCmd())
}

func newContactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contact",
		Aliases: []string{"contacts"},
		Short:   "Manage contacts via adapter",
		Long: `Manage contacts through the contacts adapter.

Dispatches to the configured contacts adapter backend
(e.g. cardamum for CardDAV).`,
	}

	cmd.AddCommand(newContactListCmd())
	cmd.AddCommand(newContactShowCmd())
	cmd.AddCommand(newContactAddCmd())
	cmd.AddCommand(newContactUpdateCmd())
	cmd.AddCommand(newContactFindCmd())
	cmd.AddCommand(newContactNoteCmd())
	cmd.AddCommand(newContactDeleteCmd())

	return cmd
}

func contactExec(
	action string,
	inputs map[string]string,
	profile string,
) error {
	profileEmail := ""
	if profile != "" {
		p, err := core.LoadProfile(profile)
		if err != nil {
			return fmt.Errorf("load profile: %w", err)
		}
		profileEmail = p.Email
	}

	mgr := coreadapter.NewManager()
	out, err := mgr.ExecAction(
		context.Background(),
		"contacts",
		action,
		inputs,
		profileEmail,
	)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// contactSummaryRow is the row shape for `aps contact list`.
type contactSummaryRow struct {
	ID          string `table:"ID,priority=9" json:"id" yaml:"id"`
	Name        string `table:"NAME,priority=8" json:"name,omitempty" yaml:"name,omitempty"`
	Email       string `table:"EMAIL,priority=7" json:"email,omitempty" yaml:"email,omitempty"`
	Org         string `table:"ORG,priority=5" json:"org,omitempty" yaml:"org,omitempty"`
	Phone       string `table:"PHONE,priority=4" json:"phone,omitempty" yaml:"phone,omitempty"`
	Addressbook string `table:"ADDRESSBOOK,priority=3" json:"addressbook,omitempty" yaml:"addressbook,omitempty"`
}

// cardamumCard mirrors the JSON shape `cardamum cards list --json`
// emits — see adapters/contacts/backends/cardamum/list.sh.
type cardamumCard struct {
	ID            string `json:"id"`
	AddressbookID string `json:"addressbook_id"`
	Vcard         string `json:"vcard"`
}

func newContactListCmd() *cobra.Command {
	var addressbook, org string
	var hasEmail bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		Long: `List every contact reachable through the configured contacts
adapter (e.g. cardamum/CardDAV) under the active profile. The
adapter is invoked once per call and the resulting cards are
projected into a uniform row shape: ID, Name (FN), Email, Org,
Phone, Addressbook.

Filters: --addressbook scopes to a single addressbook ID; --org
filters by ORG value; --has-email keeps cards that have at least
one EMAIL property. Output respects the local --format flag
(table|json|yaml). The --profile global selects which profile's
adapter configuration is used.

Read-only: queries the upstream addressbook; no contact state is
mutated.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			profile := globals.Profile()
			inputs := map[string]string{}
			if addressbook != "" {
				inputs["addressbook"] = addressbook
			}
			raw, err := contactExecCapture("list", inputs, profile)
			if err != nil {
				return err
			}
			rows, err := contactRowsFromCardamum(raw)
			if err != nil {
				return err
			}
			pred := listing.All(
				listing.MatchString(
					func(r contactSummaryRow) string { return r.Org }, org),
				listing.BoolFlag(
					cmd.Flags().Changed("has-email"),
					func(r contactSummaryRow) bool { return r.Email != "" },
					hasEmail),
			)
			rows = listing.Filter(rows, pred)
			sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
			format, _ := cmd.Flags().GetString("format")
			return listing.RenderList(os.Stdout, format, rows)
		},
	}
	// T-0648 batch 8 — local --profile dropped; the contact subcommand
	// reads globals.Profile() (set by the inherited global
	// persistent flag, including its -p shorthand).
	cmd.Flags().StringVar(&addressbook, "addressbook", "", "Addressbook ID")
	cmd.Flags().StringVar(&org, "org", "", "Filter to a single ORG value")
	cmd.Flags().BoolVar(&hasEmail, "has-email", false,
		"Filter on whether the contact has an email")
	// `contact list` reads the configured contacts adapter and projects
	// the rows; safe to retry.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

// contactExecCapture is a sibling of contactExec that returns the raw
// adapter stdout instead of writing it. Used by `contact list` so it
// can post-process JSON before rendering.
func contactExecCapture(
	action string,
	inputs map[string]string,
	profile string,
) (string, error) {
	profileEmail := ""
	if profile != "" {
		p, err := core.LoadProfile(profile)
		if err != nil {
			return "", fmt.Errorf("load profile: %w", err)
		}
		profileEmail = p.Email
	}
	mgr := coreadapter.NewManager()
	return mgr.ExecAction(
		context.Background(), "contacts", action,
		inputs, profileEmail,
	)
}

// contactRowsFromCardamum parses the JSON envelope from cardamum's
// `cards list --json` and projects each card into a contactSummaryRow
// by lifting the well-known vCard fields (FN, EMAIL, ORG, TEL).
//
// Empty input or a single-item array of empty strings returns an
// empty slice — listing.RenderList handles the empty case.
func contactRowsFromCardamum(raw string) ([]contactSummaryRow, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []contactSummaryRow{}, nil
	}
	var cards []cardamumCard
	if err := json.Unmarshal([]byte(raw), &cards); err != nil {
		return nil, fmt.Errorf("parse cardamum json: %w", err)
	}
	rows := make([]contactSummaryRow, 0, len(cards))
	for _, c := range cards {
		row := contactSummaryRow{
			ID:          c.ID,
			Addressbook: c.AddressbookID,
		}
		row.Name = vcardField(c.Vcard, "FN")
		row.Email = vcardField(c.Vcard, "EMAIL")
		row.Org = vcardField(c.Vcard, "ORG")
		row.Phone = vcardField(c.Vcard, "TEL")
		rows = append(rows, row)
	}
	return rows, nil
}

// vcardField scans a raw vCard body for the first line whose property
// name (before any `;` parameter or `:`) matches key, and returns the
// value (everything after the first `:`). Returns "" when not found.
//
// vCard line continuations (lines starting with a space) are folded
// into the previous line per RFC 6350 §3.2 — sufficient for the
// header fields we surface (FN/EMAIL/ORG/TEL rarely span lines).
func vcardField(body, key string) string {
	upper := strings.ToUpper(key)
	// Unfold continuation lines (CRLF + space) — replace with empty
	// so a wrapped value re-joins.
	body = strings.ReplaceAll(body, "\r\n ", "")
	body = strings.ReplaceAll(body, "\n ", "")
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		colon := strings.Index(line, ":")
		if colon == -1 {
			continue
		}
		head := line[:colon]
		// Property name is before the first ';' parameter.
		if semi := strings.Index(head, ";"); semi != -1 {
			head = head[:semi]
		}
		if strings.ToUpper(head) == upper {
			return line[colon+1:]
		}
	}
	return ""
}

func newContactShowCmd() *cobra.Command {
	// T-0648 batch 8 — local --profile dropped; read the global via
	// root.Viper (inherited persistent flag, including its -p shorthand).
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show contact detail",
		Long: `Show the full vCard body for a single contact identified by id
via the configured contacts adapter. Output is the raw adapter
response (typically vCard text) rather than the projected row shape
used by aps contact list. The --profile global selects which
profile's adapter configuration handles the request.

Read-only: queries the upstream addressbook; idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("show",
				map[string]string{"id": args[0]},
				globals.Profile())
		},
	}
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func newContactAddCmd() *cobra.Command {
	var name, org, phone, note, addressbook string
	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Add a new contact",
		Long: `Add a new contact to the configured contacts adapter under the
active profile. The email argument is required; optional fields
(--name, --org, --phone, --note, --addressbook) populate the
corresponding vCard properties on the new card.

The contact id is minted by the upstream provider (e.g. cardamum
returns a freshly-issued UID), so --dry-run is opted out — aps
cannot preview an ID the provider has not yet assigned. Each
invocation creates a fresh card; not idempotent. Use aps contact
find to look up an existing card before adding to avoid duplicates.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			inputs := map[string]string{"email": args[0]}
			if name != "" {
				inputs["name"] = name
			}
			if org != "" {
				inputs["org"] = org
			}
			if phone != "" {
				inputs["phone"] = phone
			}
			if note != "" {
				inputs["note"] = note
			}
			if addressbook != "" {
				inputs["addressbook"] = addressbook
			}
			return contactExec("add", inputs, globals.Profile())
		},
	}
	// T-0648 batch 8 — local --profile dropped; read via root.Viper.
	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&org, "org", "", "Organization")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&note, "note", "", "Note")
	cmd.Flags().StringVar(&addressbook, "addressbook", "", "Addressbook ID")
	// `contact add` mints a new contact each call (write-shared into the
	// upstream addressbook, no caller-side dedupe).
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — add proxies to the bundled contact provider; the
	// provider mints an ID on its side, so preview has nothing local to
	// inspect without speaking the provider's wire protocol.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "add proxies to a bundled contact provider that mints a fresh ID on its side; previewing without the provider call would invent IDs the provider never assigned."); err != nil {
		panic(err)
	}
	return cmd
}

func newContactUpdateCmd() *cobra.Command {
	var name, email, org, phone, note string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update contact fields",
		Long: `Update one or more vCard fields on an existing contact via the
configured contacts adapter. The id argument identifies the card;
only flags that are explicitly passed are applied (--name, --email,
--org, --phone, --note). Unset flags leave the existing value
unchanged.

Idempotent at the field level: repeating the call with the same
payload converges. --dry-run is opted out because previewing the
diff would require the same provider round-trip that performs the
update.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			inputs := map[string]string{"id": args[0]}
			if name != "" {
				inputs["name"] = name
			}
			if email != "" {
				inputs["email"] = email
			}
			if org != "" {
				inputs["org"] = org
			}
			if phone != "" {
				inputs["phone"] = phone
			}
			if note != "" {
				inputs["note"] = note
			}
			return contactExec("update", inputs, globals.Profile())
		},
	}
	// T-0648 batch 8 — local --profile dropped; read via root.Viper.
	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&org, "org", "", "Organization")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&note, "note", "", "Note")
	// `contact update` overwrites the named fields on an existing card;
	// repeating with the same payload converges.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — update proxies to the bundled contact provider; the
	// preview would need a provider round-trip to read current values.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "update proxies to a bundled contact provider; previewing the diff would require the same provider round-trip that performs the update."); err != nil {
		panic(err)
	}
	return cmd
}

func newContactFindCmd() *cobra.Command {
	// T-0648 batch 8 — local --profile dropped; read via root.Viper.
	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Search contacts",
		Long: `Search contacts via the configured contacts adapter. The query
argument is forwarded verbatim to the adapter, which decides what
fields to match (name, email, org, etc.). Output is the raw adapter
response.

The --profile global selects which profile's adapter configuration
handles the search. Read-only: idempotent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("find",
				map[string]string{"query": args[0]},
				globals.Profile())
		},
	}
	kitcli.SetSideEffect(cmd, kitcli.SideEffectRead)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	return cmd
}

func newContactNoteCmd() *cobra.Command {
	// T-0648 batch 8 — local --profile dropped; read via root.Viper.
	cmd := &cobra.Command{
		Use:   "note <id> <text>",
		Short: "Append note to contact",
		Long: `Append a note to an existing contact identified by id via the
configured contacts adapter. The remaining positional arguments are
joined with single spaces into the note text. Each call appends a
fresh entry — not idempotent.

The note is stored in the upstream addressbook's NOTE field (or
adapter-specific equivalent). --dry-run is opted out because the
preview would only restate the text the user already typed.`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("note",
				map[string]string{
					"id":   args[0],
					"text": strings.Join(args[1:], " "),
				},
				globals.Profile())
		},
	}
	// `contact note` appends each call; not naturally idempotent.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyNo)
	// T-0656 — note proxies to the bundled contact provider's note
	// endpoint; preview cannot show an appended note without doing the
	// append.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "note appends a fresh entry via the bundled contact provider; previewing would only restate the text the user already typed."); err != nil {
		panic(err)
	}
	return cmd
}

func newContactDeleteCmd() *cobra.Command {
	// T-0648 batch 8 — local --profile dropped; read via root.Viper.
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a contact",
		Long: `Delete a contact from the configured contacts adapter by id. The
operation forwards to the upstream addressbook's delete endpoint;
the local effect on aps state is none, but the upstream card is
removed.

Delete-by-id is idempotent at the wire level: repeating the call
with a missing id is a no-op on the provider side. --dry-run is
opted out because the preview would only restate the id argument.
Pair with aps contact find to confirm the target id before
running.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("delete",
				map[string]string{"id": args[0]},
				globals.Profile())
		},
	}
	// `contact delete` is delete-by-id (idempotent). Hold the
	// write-shared tier (rather than destructive-shared) until the
	// tree-wide T-0653/T-0657 step lands the kit confirm gate alongside
	// the matching e2e --confirm=yes updates.
	kitcli.SetSideEffect(cmd, kitcli.SideEffectWriteShared)
	kitcli.SetIdempotency(cmd, kitcli.IdempotencyYes)
	// T-0656 — delete proxies to the bundled contact provider; preview
	// would only restate the contact ID the user already passed.
	kitcli.OptOutDryRun(cmd)
	if err := kitcli.SetDryRunRationale(cmd, "delete proxies to the bundled contact provider's delete endpoint; previewing would only restate the contact ID the user already passed."); err != nil {
		panic(err)
	}
	return cmd
}
