package msgroute

// Config is the `routing:` block of a message service. It replaces the
// service's single default_action with a sender-keyed route table.
//
// Exactly one of File or Routes must be set. File points at an external YAML
// document with the same shape (contacts + routes, no nested file) so
// per-organization tables can live outside the service record.
type Config struct {
	File     string          `yaml:"file,omitempty" json:"file,omitempty"`
	Contacts *ContactsConfig `yaml:"contacts,omitempty" json:"contacts,omitempty"`
	Routes   []Route         `yaml:"routes,omitempty" json:"routes,omitempty"`
}

// Route is one ordered entry of a route table.
//
// Match grammar (evaluated against the normalized sender key unless a
// selector prefix is present):
//
//	unknown          terminal fail-safe; matches every sender; must be last
//	org:<pattern>    resolved contact organization (case-folded)
//	contact:<pattern> resolved contact ID (case-folded)
//	<pattern>        normalized sender key
//
// A pattern containing * ? or [ is a glob (path.Match semantics); anything
// else is an exact comparison. Profile defaults to the service profile.
type Route struct {
	Match   string `yaml:"match" json:"match"`
	Profile string `yaml:"profile,omitempty" json:"profile,omitempty"`
	Action  string `yaml:"action" json:"action"`
}

// ContactsConfig declares the contact snapshot consulted before matching.
// Path is a YAML document of the shape `contacts: [...]`; Entries are inline
// contacts. Both may be set; entries are appended after the file.
type ContactsConfig struct {
	Path    string    `yaml:"path,omitempty" json:"path,omitempty"`
	Entries []Contact `yaml:"entries,omitempty" json:"entries,omitempty"`
}

// Contact is one entry of the contact snapshot. Keys are sender identifiers
// in any provider form (Twilio `whatsapp:+1…`, bare E.164, email, platform
// user ID); they are normalized at load time.
type Contact struct {
	ID   string   `yaml:"id" json:"id"`
	Name string   `yaml:"name,omitempty" json:"name,omitempty"`
	Org  string   `yaml:"org,omitempty" json:"org,omitempty"`
	Keys []string `yaml:"keys" json:"keys"`
}

// contactsFile is the on-disk shape of a standalone contacts document.
type contactsFile struct {
	Contacts []Contact `yaml:"contacts"`
}

// TerminalMatch is the reserved match keyword for the fail-safe route.
const TerminalMatch = "unknown"

// Selector prefixes recognized in Route.Match.
const (
	orgSelector     = "org:"
	contactSelector = "contact:"
)
