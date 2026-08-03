package core

import (
	"fmt"
	"strings"

	citescheme "hop.top/cite/scheme"
)

// ProfileURIScheme is the canonical scheme for aps profile URIs.
const ProfileURIScheme = "aps"

// ProfileURISpace is the canonical space for aps profile URIs.
const ProfileURISpace = "profile"

// URI returns the canonical URI for this profile (aps://profile/<id>).
// Cross-tool refs use this form (e.g. linking from tlc, ctxt, wsm).
func (p *Profile) URI() string {
	u := &citescheme.URI{Scheme: ProfileURIScheme, Namespace: ProfileURISpace, ID: p.ID}
	return u.String()
}

// ParseProfileRef accepts either a bare profile id ("noor") or a full
// aps://profile/<id> URI and returns the resolved profile id.
// Returns an error for empty input or refs with a non-aps scheme or
// non-profile space.
func ParseProfileRef(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("empty profile ref")
	}

	// Bare id form (no scheme separator). Parse rejects inputs without
	// a scheme, so handle bare ids before delegating.
	if !strings.Contains(s, "://") {
		return s, nil
	}

	u, err := citescheme.Parse(s)
	if err != nil {
		return "", fmt.Errorf("parse profile ref: %w", err)
	}

	if u.Scheme != ProfileURIScheme {
		return "", fmt.Errorf("invalid profile ref scheme %q (want %q)", u.Scheme, ProfileURIScheme)
	}
	if u.Namespace != ProfileURISpace {
		return "", fmt.Errorf("invalid profile ref space %q (want %q)", u.Namespace, ProfileURISpace)
	}
	return u.ID, nil
}
