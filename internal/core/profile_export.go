package core

import (
	"os"
	"path/filepath"
	"strings"
)

// SanitizePathsForExport replaces absolute home directory paths with ~ placeholders.
func SanitizePathsForExport(profile *Profile) *Profile {
	p := *profile // shallow copy
	home, err := os.UserHomeDir()
	if err != nil {
		return &p
	}

	if p.SSH.KeyPath != "" && filepath.IsAbs(p.SSH.KeyPath) {
		if rel, err := filepath.Rel(home, p.SSH.KeyPath); err == nil {
			p.SSH.KeyPath = filepath.Join("~", rel)
		}
	}

	if p.Identity != nil && p.Identity.KeyPath != "" && filepath.IsAbs(p.Identity.KeyPath) {
		identity := *p.Identity // shallow copy
		if rel, err := filepath.Rel(home, identity.KeyPath); err == nil {
			identity.KeyPath = filepath.Join("~", rel)
		}
		p.Identity = &identity
	}

	return &p
}

// RestorePathsFromImport expands ~ placeholders back to absolute paths.
func RestorePathsFromImport(profile *Profile) *Profile {
	p := *profile
	home, _ := os.UserHomeDir()

	if strings.HasPrefix(p.SSH.KeyPath, "~/") {
		p.SSH.KeyPath = filepath.Join(home, p.SSH.KeyPath[2:])
	}

	if p.Identity != nil && strings.HasPrefix(p.Identity.KeyPath, "~/") {
		identity := *p.Identity
		identity.KeyPath = filepath.Join(home, identity.KeyPath[2:])
		p.Identity = &identity
	}

	return &p
}
