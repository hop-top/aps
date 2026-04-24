package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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

func newContactListCmd() *cobra.Command {
	var profile, addressbook string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		RunE: func(_ *cobra.Command, _ []string) error {
			inputs := map[string]string{}
			if addressbook != "" {
				inputs["addressbook"] = addressbook
			}
			return contactExec("list", inputs, profile)
		},
	}
	cmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile ID")
	cmd.Flags().StringVar(&addressbook, "addressbook", "", "Addressbook ID")
	return cmd
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
