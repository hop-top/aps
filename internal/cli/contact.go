package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/cli/listing"
	"hop.top/aps/internal/core"
	coreadapter "hop.top/aps/internal/core/adapter"
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
	var profile, addressbook, org string
	var hasEmail bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		RunE: func(cmd *cobra.Command, _ []string) error {
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
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	cmd.Flags().StringVar(&addressbook, "addressbook", "", "Addressbook ID")
	cmd.Flags().StringVar(&org, "org", "", "Filter to a single ORG value")
	cmd.Flags().BoolVar(&hasEmail, "has-email", false,
		"Filter on whether the contact has an email")
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
	var profile string
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show contact detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("show",
				map[string]string{"id": args[0]}, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	return cmd
}

func newContactAddCmd() *cobra.Command {
	var profile, name, org, phone, note, addressbook string
	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Add a new contact",
		Args:  cobra.ExactArgs(1),
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
			return contactExec("add", inputs, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&org, "org", "", "Organization")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&note, "note", "", "Note")
	cmd.Flags().StringVar(&addressbook, "addressbook", "", "Addressbook ID")
	return cmd
}

func newContactUpdateCmd() *cobra.Command {
	var profile, name, email, org, phone, note string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update contact fields",
		Args:  cobra.ExactArgs(1),
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
			return contactExec("update", inputs, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	cmd.Flags().StringVar(&name, "name", "", "Contact name")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&org, "org", "", "Organization")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	cmd.Flags().StringVar(&note, "note", "", "Note")
	return cmd
}

func newContactFindCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Search contacts",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("find",
				map[string]string{"query": args[0]}, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	return cmd
}

func newContactNoteCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "note <id> <text>",
		Short: "Append note to contact",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("note",
				map[string]string{
					"id":   args[0],
					"text": strings.Join(args[1:], " "),
				}, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	return cmd
}

func newContactDeleteCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return contactExec("delete",
				map[string]string{"id": args[0]}, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	return cmd
}
