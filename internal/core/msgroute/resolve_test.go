package msgroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestTable(t *testing.T, platform string, cfg *Config) *Table {
	t.Helper()
	table, err := Load(cfg, Options{Platform: platform, DefaultProfile: "assistant"})
	require.NoError(t, err)
	return table
}

func TestResolve_ExactSenderKeyAcrossProviderForms(t *testing.T) {
	table := loadTestTable(t, "whatsapp", &Config{
		Routes: []Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})

	for _, sender := range []string{
		"whatsapp:+15551234567", // Twilio
		"15551234567",           // WhatsApp Cloud wa_id
		"+1 (555) 123-4567",     // formatted
	} {
		d := table.Resolve(sender)
		assert.Equal(t, "+15551234567", d.SenderKey, sender)
		assert.Equal(t, "acme", d.Route.Profile, sender)
		assert.Equal(t, "inbox", d.Route.Action, sender)
		assert.Equal(t, 0, d.Index, sender)
		assert.False(t, d.Terminal, sender)
		assert.Equal(t, "acme=inbox", d.Mapping(), sender)
	}
}

func TestResolve_GlobSenderKey(t *testing.T) {
	table := loadTestTable(t, "sms", &Config{
		Routes: []Route{
			{Match: "+1555*", Profile: "sales", Action: "inbox"},
			{Match: "*@acme.com", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})

	d := table.Resolve("+15559876543")
	assert.Equal(t, "sales", d.Route.Profile)
	assert.Equal(t, 0, d.Index)

	d = table.Resolve("Jane@Acme.com")
	assert.Equal(t, "acme", d.Route.Profile)
	assert.Equal(t, 1, d.Index)

	d = table.Resolve("+16005550000")
	assert.True(t, d.Terminal)
	assert.Equal(t, "triage", d.Route.Profile)
	assert.Equal(t, 2, d.Index)
}

func TestResolve_FirstMatchWinsInDeclaredOrder(t *testing.T) {
	table := loadTestTable(t, "sms", &Config{
		Routes: []Route{
			{Match: "+1555*", Profile: "broad", Action: "inbox"},
			{Match: "+15551234567", Profile: "narrow", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})
	d := table.Resolve("+15551234567")
	assert.Equal(t, "broad", d.Route.Profile, "glob declared first wins over a later exact route")
	assert.Equal(t, 0, d.Index)
}

func TestResolve_ContactOrgSelector(t *testing.T) {
	table := loadTestTable(t, "whatsapp", &Config{
		Contacts: &ContactsConfig{Entries: []Contact{
			{ID: "Jane", Name: "Jane Doe", Org: "Acme", Keys: []string{"whatsapp:+15551234567", "jane@acme.com"}},
			{ID: "bob", Org: "Globex-EU", Keys: []string{"+15559876543"}},
			{ID: "solo", Keys: []string{"+15550000000"}},
		}},
		Routes: []Route{
			{Match: "org:acme", Profile: "acme", Action: "inbox"},
			{Match: "org:globex-*", Profile: "globex", Action: "inbox"},
			{Match: "contact:SOLO", Profile: "vip", Action: "concierge"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})

	d := table.Resolve("15551234567")
	require.NotNil(t, d.Contact)
	assert.Equal(t, "Jane", d.Contact.ID)
	assert.Equal(t, "Acme", d.Contact.Org)
	assert.Equal(t, "acme", d.Route.Profile)
	assert.False(t, d.Terminal)

	d = table.Resolve("+1 555 987 6543")
	require.NotNil(t, d.Contact)
	assert.Equal(t, "bob", d.Contact.ID)
	assert.Equal(t, "globex", d.Route.Profile, "org glob is case-folded")

	d = table.Resolve("+15550000000")
	require.NotNil(t, d.Contact)
	assert.Equal(t, "vip", d.Route.Profile, "contact selector is case-folded")

	d = table.Resolve("+15551112222")
	assert.Nil(t, d.Contact)
	assert.True(t, d.Terminal)
	assert.Equal(t, "triage", d.Route.Profile)
}

func TestResolve_ContactWithoutOrgDoesNotMatchOrgSelector(t *testing.T) {
	table := loadTestTable(t, "sms", &Config{
		Contacts: &ContactsConfig{Entries: []Contact{
			{ID: "solo", Keys: []string{"+15550000000"}},
		}},
		Routes: []Route{
			{Match: "org:*", Profile: "org-only", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})
	d := table.Resolve("+15550000000")
	require.NotNil(t, d.Contact)
	assert.True(t, d.Terminal, "org:* must not match a contact with an empty org")
}

func TestResolve_TerminalFailSafeAlwaysResolves(t *testing.T) {
	table := loadTestTable(t, "sms", &Config{
		Routes: []Route{
			{Match: "+15551234567", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})
	for _, sender := range []string{"", "   ", "whatsapp:", "VERIFY", "+19998887777"} {
		d := table.Resolve(sender)
		assert.True(t, d.Terminal, "sender %q must fall through to the terminal route", sender)
		assert.Equal(t, "triage=triage", d.Mapping())
		assert.Equal(t, TerminalMatch, d.Route.Match)
	}
}

func TestResolve_NilTable(t *testing.T) {
	var table *Table
	d := table.Resolve("+15551234567")
	assert.Equal(t, "", d.Mapping())
	assert.False(t, d.Terminal)
	assert.Equal(t, -1, d.Index)
}

func TestDecision_Metadata(t *testing.T) {
	table := loadTestTable(t, "whatsapp", &Config{
		Contacts: &ContactsConfig{Entries: []Contact{
			{ID: "jane", Name: "Jane Doe", Org: "acme", Keys: []string{"+15551234567"}},
		}},
		Routes: []Route{
			{Match: "org:acme", Profile: "acme", Action: "inbox"},
			{Match: "unknown", Profile: "triage", Action: "triage"},
		},
	})

	known := table.Resolve("whatsapp:+15551234567").Metadata()
	assert.Equal(t, "+15551234567", known["sender_key"])
	assert.Equal(t, "org:acme", known["match"])
	assert.Equal(t, false, known["terminal"])
	assert.Equal(t, 0, known["route"])
	assert.Equal(t, map[string]any{"id": "jane", "name": "Jane Doe", "org": "acme"}, known["contact"])

	unknown := table.Resolve("+15550000000").Metadata()
	assert.Equal(t, true, unknown["terminal"])
	assert.Equal(t, TerminalMatch, unknown["match"])
	_, hasContact := unknown["contact"]
	assert.False(t, hasContact)
}
