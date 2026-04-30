package core

import (
	"fmt"

	"hop.top/uri"
)

// ProfileURIScheme is the canonical scheme for aps profile URIs.
const ProfileURIScheme = "aps"

// ProfileURISpace is the canonical space for aps profile URIs.
const ProfileURISpace = "profile"

// URI returns the canonical URI for this profile (aps://profile/<id>).
// Cross-tool refs use this form (e.g. linking from tlc, ctxt, wsm).
func (p *Profile) URI() string {
	u := &uri.URI{Scheme: ProfileURIScheme, Space: ProfileURISpace, ID: p.ID}
	return u.String()
}

// ParseProfileRef accepts either a bare profile id ("noor") or a full
// aps://profile/<id> URI and returns the resolved profile id.
// Returns an error for empty input or refs with a non-aps scheme or
// non-profile space.
func ParseProfileRef(s string) (string, error) {
	u, err := uri.Parse(s)
	if err != nil {
		return "", err
	}

	// Bare id form: Parse returns URI{ID: s}.
	if u.Scheme == "" && u.Space == "" {
		if u.ID == "" {
			return "", fmt.Errorf("empty profile ref")
		}
		return u.ID, nil
	}

	if u.Scheme != "" && u.Scheme != ProfileURIScheme {
		return "", fmt.Errorf("invalid profile ref scheme %q (want %q)", u.Scheme, ProfileURIScheme)
	}
	if u.Space != ProfileURISpace {
		return "", fmt.Errorf("invalid profile ref space %q (want %q)", u.Space, ProfileURISpace)
	}
	return u.ID, nil
}
