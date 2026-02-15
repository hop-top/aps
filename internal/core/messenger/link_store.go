package messenger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"oss-aps-cli/internal/core"
)

// LinkStore persists profile-messenger links to disk as JSON.
// Each profile has its own messenger-links.json file located at
// ~/.agents/profiles/{profileID}/messenger-links.json.
type LinkStore struct {
	profileID string
}

// NewLinkStore creates a new LinkStore for the given profile.
func NewLinkStore(profileID string) *LinkStore {
	return &LinkStore{profileID: profileID}
}

// Load reads all profile-messenger links from disk.
// Returns an empty slice if the file does not exist yet.
func (s *LinkStore) Load() ([]ProfileMessengerLink, error) {
	path, err := s.linksPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve links path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ProfileMessengerLink{}, nil
		}
		return nil, fmt.Errorf("failed to read messenger links for profile '%s': %w", s.profileID, err)
	}

	// Handle empty file gracefully
	if len(data) == 0 {
		return []ProfileMessengerLink{}, nil
	}

	var links []ProfileMessengerLink
	if err := json.Unmarshal(data, &links); err != nil {
		return nil, fmt.Errorf("failed to parse messenger links for profile '%s': %w", s.profileID, err)
	}

	return links, nil
}

// Save writes all profile-messenger links to disk atomically.
// The file is written with MarshalIndent for human readability.
func (s *LinkStore) Save(links []ProfileMessengerLink) error {
	path, err := s.linksPath()
	if err != nil {
		return fmt.Errorf("failed to resolve links path: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal messenger links: %w", err)
	}

	// Write to a temp file first, then rename for atomicity
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write messenger links: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on rename failure
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize messenger links: %w", err)
	}

	return nil
}

// linksPath returns the full path to the messenger-links.json file
// for this store's profile: ~/.agents/profiles/{profileID}/messenger-links.json
func (s *LinkStore) linksPath() (string, error) {
	profileDir, err := core.GetProfileDir(s.profileID)
	if err != nil {
		return "", err
	}
	return filepath.Join(profileDir, "messenger-links.json"), nil
}
