package msgroute

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ContactSource resolves a normalized sender key to a contact. The YAML
// snapshot is the first-class implementation; other backends (address-book
// adapters, directories) can satisfy the same interface.
type ContactSource interface {
	// LookupContact returns the contact owning key, or false when unknown.
	LookupContact(key string) (Contact, bool)
}

// contactIndex is the in-memory ContactSource built from a ContactsConfig.
type contactIndex struct {
	contacts []Contact
	byKey    map[string]int
}

var _ ContactSource = (*contactIndex)(nil)

func (c *contactIndex) LookupContact(key string) (Contact, bool) {
	if c == nil {
		return Contact{}, false
	}
	i, ok := c.byKey[key]
	if !ok {
		return Contact{}, false
	}
	return c.contacts[i], true
}

func (c *contactIndex) list() []Contact {
	out := make([]Contact, len(c.contacts))
	copy(out, c.contacts)
	return out
}

// loadContacts reads the file (when set), appends inline entries, normalizes
// every key for the platform, and indexes them. Validation problems are
// aggregated; a non-nil index is returned alongside the error so route
// compilation can still tell whether a contacts source is declared.
func loadContacts(cfg *ContactsConfig, opts Options) (*contactIndex, error) {
	index := &contactIndex{byKey: map[string]int{}}
	var problems []error

	var entries []Contact
	if p := strings.TrimSpace(cfg.Path); p != "" {
		fromFile, err := readContactsFile(p, opts.BaseDir)
		if err != nil {
			problems = append(problems, err)
		}
		entries = append(entries, fromFile...)
	}
	entries = append(entries, cfg.Entries...)

	seenIDs := map[string]int{}
	for i, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			problems = append(problems, fmt.Errorf("contact %d: id is required", i))
			continue
		}
		if prev, dup := seenIDs[strings.ToLower(id)]; dup {
			problems = append(problems, fmt.Errorf("contact %d: duplicate contact id %q (first at contact %d)", i, id, prev))
			continue
		}
		seenIDs[strings.ToLower(id)] = i

		normalized := Contact{
			ID:   id,
			Name: strings.TrimSpace(entry.Name),
			Org:  strings.TrimSpace(entry.Org),
		}
		for _, raw := range entry.Keys {
			key := NormalizeSenderKey(opts.Platform, raw)
			if key == "" {
				continue
			}
			if owner, taken := index.byKey[key]; taken {
				problems = append(problems, fmt.Errorf("contact %q: key %q already belongs to contact %q", id, key, index.contacts[owner].ID))
				continue
			}
			index.byKey[key] = len(index.contacts)
			normalized.Keys = append(normalized.Keys, key)
		}
		if len(normalized.Keys) == 0 {
			problems = append(problems, fmt.Errorf("contact %q: at least one key is required", id))
			continue
		}
		index.contacts = append(index.contacts, normalized)
	}

	return index, errors.Join(problems...)
}

func readContactsFile(value, baseDir string) ([]Contact, error) {
	resolved, err := resolvePath(value, baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve contacts path: %w", err)
	}
	data, err := os.ReadFile(resolved) //nolint:gosec // operator-declared path from service config
	if err != nil {
		return nil, fmt.Errorf("read contacts %s: %w", resolved, err)
	}
	var doc contactsFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse contacts %s: %w", resolved, err)
	}
	return doc.Contacts, nil
}
