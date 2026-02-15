package messenger

import (
	"fmt"
	"time"

	"oss-aps-cli/internal/core"
)

// Manager handles messenger-profile linking, channel mapping, and route resolution.
// All persistence is delegated to LinkStore instances per profile, so operations
// are file-based and do not require in-memory synchronization.
type Manager struct{}

// NewManager creates a new Manager.
func NewManager() *Manager {
	return &Manager{}
}

// LinkMessengerToProfile creates a new link between a messenger and a profile,
// with the given channel-to-action mappings.
//
// It enforces unique mapping: a channel ID can map to exactly one profile:action
// across ALL profiles for a given messenger. Returns ErrMappingConflict if
// a channel is already mapped elsewhere.
func (m *Manager) LinkMessengerToProfile(messengerName, profileID string, mappings map[string]string) error {
	if messengerName == "" {
		return fmt.Errorf("messenger name is required")
	}
	if profileID == "" {
		return fmt.Errorf("profile ID is required")
	}

	store := NewLinkStore(profileID)
	links, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load links for profile '%s': %w", profileID, err)
	}

	// Check if link already exists for this messenger
	for _, link := range links {
		if link.MessengerName == messengerName {
			return ErrLinkAlreadyExists(messengerName, profileID)
		}
	}

	// Validate mappings for cross-profile conflicts
	if err := m.validateMappingsAcrossProfiles(messengerName, profileID, mappings); err != nil {
		return err
	}

	now := time.Now().UTC()
	newLink := ProfileMessengerLink{
		ProfileID:      profileID,
		MessengerName:  messengerName,
		MessengerScope: "global",
		Enabled:        true,
		Mappings:       mappings,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := newLink.Validate(); err != nil {
		return err
	}

	links = append(links, newLink)
	return store.Save(links)
}

// UnlinkMessengerFromProfile removes the link between a messenger and a profile.
func (m *Manager) UnlinkMessengerFromProfile(messengerName, profileID string) error {
	store := NewLinkStore(profileID)
	links, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load links for profile '%s': %w", profileID, err)
	}

	found := false
	remaining := make([]ProfileMessengerLink, 0, len(links))
	for _, link := range links {
		if link.MessengerName == messengerName {
			found = true
			continue
		}
		remaining = append(remaining, link)
	}

	if !found {
		return ErrLinkNotFound(messengerName, profileID)
	}

	return store.Save(remaining)
}

// GetProfileLinks returns all messenger links for a profile.
func (m *Manager) GetProfileLinks(profileID string) ([]ProfileMessengerLink, error) {
	store := NewLinkStore(profileID)
	return store.Load()
}

// GetMessengerLinks returns all links across all profiles for a given messenger.
func (m *Manager) GetMessengerLinks(messengerName string) ([]ProfileMessengerLink, error) {
	profileIDs, err := core.ListProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	var result []ProfileMessengerLink
	for _, pid := range profileIDs {
		store := NewLinkStore(pid)
		links, err := store.Load()
		if err != nil {
			continue
		}
		for _, link := range links {
			if link.MessengerName == messengerName {
				result = append(result, link)
			}
		}
	}

	return result, nil
}

// AddMapping adds a channel-to-action mapping to an existing link.
// Enforces the unique mapping constraint across all profiles.
func (m *Manager) AddMapping(messengerName, profileID, channelID, action string) error {
	if channelID == "" {
		return fmt.Errorf("channel ID is required")
	}
	if action == "" {
		return fmt.Errorf("action is required")
	}

	store := NewLinkStore(profileID)
	links, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load links for profile '%s': %w", profileID, err)
	}

	idx := -1
	for i, link := range links {
		if link.MessengerName == messengerName {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrLinkNotFound(messengerName, profileID)
	}

	// Check cross-profile conflict for this single channel
	newMappings := map[string]string{channelID: action}
	if err := m.validateMappingsAcrossProfiles(messengerName, profileID, newMappings); err != nil {
		return err
	}

	// Also check within this profile's existing mappings for other links
	// (a channel should not be doubly mapped within the same link, but
	// updating an existing mapping within the same link is allowed)
	if links[idx].Mappings == nil {
		links[idx].Mappings = make(map[string]string)
	}
	links[idx].Mappings[channelID] = action
	links[idx].UpdatedAt = time.Now().UTC()

	return store.Save(links)
}

// RemoveMapping removes a channel mapping from an existing link.
func (m *Manager) RemoveMapping(messengerName, profileID, channelID string) error {
	if channelID == "" {
		return fmt.Errorf("channel ID is required")
	}

	store := NewLinkStore(profileID)
	links, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load links for profile '%s': %w", profileID, err)
	}

	idx := -1
	for i, link := range links {
		if link.MessengerName == messengerName {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrLinkNotFound(messengerName, profileID)
	}

	if _, exists := links[idx].Mappings[channelID]; !exists {
		return ErrUnknownChannel(messengerName, channelID)
	}

	delete(links[idx].Mappings, channelID)
	links[idx].UpdatedAt = time.Now().UTC()

	return store.Save(links)
}

// SetDefaultAction sets the default action for unmapped channels on a link.
func (m *Manager) SetDefaultAction(messengerName, profileID, action string) error {
	store := NewLinkStore(profileID)
	links, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load links for profile '%s': %w", profileID, err)
	}

	idx := -1
	for i, link := range links {
		if link.MessengerName == messengerName {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrLinkNotFound(messengerName, profileID)
	}

	links[idx].DefaultAction = action
	links[idx].UpdatedAt = time.Now().UTC()

	return store.Save(links)
}

// EnableLink enables a messenger-profile link.
func (m *Manager) EnableLink(messengerName, profileID string) error {
	return m.setLinkEnabled(messengerName, profileID, true)
}

// DisableLink disables a messenger-profile link.
func (m *Manager) DisableLink(messengerName, profileID string) error {
	return m.setLinkEnabled(messengerName, profileID, false)
}

// ResolveChannelRoute looks up the link and action for a given messenger+channel.
// It iterates all profiles that have links to the messenger and finds the one
// with a matching channel mapping. Returns the matching link, the resolved action
// string, and any error.
//
// Only enabled links are considered. If no mapping is found, returns ErrUnknownChannel.
func (m *Manager) ResolveChannelRoute(messengerName, channelID string) (*ProfileMessengerLink, string, error) {
	if messengerName == "" {
		return nil, "", fmt.Errorf("messenger name is required")
	}
	if channelID == "" {
		return nil, "", fmt.Errorf("channel ID is required")
	}

	profileIDs, err := core.ListProfiles()
	if err != nil {
		return nil, "", &MessengerError{
			Name:    messengerName,
			Message: "failed to list profiles for route resolution",
			Code:    ErrCodeRoutingFailed,
			Cause:   err,
		}
	}

	for _, pid := range profileIDs {
		store := NewLinkStore(pid)
		links, err := store.Load()
		if err != nil {
			continue
		}

		for i, link := range links {
			if link.MessengerName != messengerName {
				continue
			}
			if !link.Enabled {
				continue
			}

			action, found := link.GetActionForChannel(channelID)
			if found {
				return &links[i], action, nil
			}
		}
	}

	return nil, "", ErrUnknownChannel(messengerName, channelID)
}

// setLinkEnabled is the internal helper for EnableLink/DisableLink.
func (m *Manager) setLinkEnabled(messengerName, profileID string, enabled bool) error {
	store := NewLinkStore(profileID)
	links, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load links for profile '%s': %w", profileID, err)
	}

	idx := -1
	for i, link := range links {
		if link.MessengerName == messengerName {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrLinkNotFound(messengerName, profileID)
	}

	links[idx].Enabled = enabled
	links[idx].UpdatedAt = time.Now().UTC()

	return store.Save(links)
}

// validateMappingsAcrossProfiles checks that none of the proposed channel mappings
// conflict with existing mappings in other profiles for the same messenger.
// This enforces the unique mapping constraint: a channel ID can map to exactly
// one profile:action across ALL profiles for a messenger.
func (m *Manager) validateMappingsAcrossProfiles(messengerName, proposingProfileID string, proposedMappings map[string]string) error {
	if len(proposedMappings) == 0 {
		return nil
	}

	profileIDs, err := core.ListProfiles()
	if err != nil {
		return fmt.Errorf("failed to list profiles for conflict check: %w", err)
	}

	for _, pid := range profileIDs {
		if pid == proposingProfileID {
			continue
		}

		store := NewLinkStore(pid)
		links, err := store.Load()
		if err != nil {
			continue
		}

		for _, link := range links {
			if link.MessengerName != messengerName {
				continue
			}
			for channelID := range proposedMappings {
				if existingAction, exists := link.Mappings[channelID]; exists {
					return ErrMappingConflict(channelID, pid, existingAction)
				}
			}
		}
	}

	return nil
}
