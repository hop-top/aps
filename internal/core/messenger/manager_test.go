package messenger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupTestProfile creates a minimal profile directory structure under the data
// directory (APS_DATA_PATH) so that core.GetProfileDir, core.ListProfiles,
// and LinkStore all resolve correctly.
func setupTestProfile(t *testing.T, home, profileID string) {
	t.Helper()
	profileDir := filepath.Join(home, "profiles", profileID)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		t.Fatalf("failed to create profile dir: %v", err)
	}
	profileYAML := "id: " + profileID + "\ndisplay_name: Test Profile\ncapabilities: []\n"
	if err := os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(profileYAML), 0644); err != nil {
		t.Fatalf("failed to write profile.yaml: %v", err)
	}
}

// readLinksFile reads and parses the messenger-links.json for a profile.
func readLinksFile(t *testing.T, home, profileID string) []ProfileMessengerLink {
	t.Helper()
	path := filepath.Join(home, "profiles", profileID, "messenger-links.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("failed to read links file: %v", err)
	}
	var links []ProfileMessengerLink
	if err := json.Unmarshal(data, &links); err != nil {
		t.Fatalf("failed to parse links file: %v", err)
	}
	return links
}

func TestManager_LinkMessengerToProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	mappings := map[string]string{
		"chan-1": "test-profile=handle-msg",
	}

	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", mappings)
	if err != nil {
		t.Fatalf("LinkMessengerToProfile failed: %v", err)
	}

	// Verify link file was created
	links := readLinksFile(t, home, "test-profile")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].MessengerName != "my-telegram" {
		t.Errorf("MessengerName = %q, want %q", links[0].MessengerName, "my-telegram")
	}
	if links[0].ProfileID != "test-profile" {
		t.Errorf("ProfileID = %q, want %q", links[0].ProfileID, "test-profile")
	}
	if !links[0].Enabled {
		t.Error("expected link to be enabled by default")
	}
	if links[0].Mappings["chan-1"] != "test-profile=handle-msg" {
		t.Errorf("Mappings[chan-1] = %q, want %q", links[0].Mappings["chan-1"], "test-profile=handle-msg")
	}
}

func TestManager_LinkMessengerToProfile_AlreadyExists(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("first link failed: %v", err)
	}

	err = mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err == nil {
		t.Fatal("expected error for duplicate link, got nil")
	}
	if !IsLinkNotFound(err) {
		// The error should be ErrLinkAlreadyExists, not ErrLinkNotFound
		me, ok := err.(*MessengerError)
		if !ok || me.Code != ErrCodeLinkAlreadyExists {
			t.Errorf("expected ErrCodeLinkAlreadyExists, got %v", err)
		}
	}
}

func TestManager_LinkMessengerToProfile_ValidationErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)

	mgr := NewManager()

	tests := []struct {
		name      string
		messenger string
		profile   string
		wantErr   string
	}{
		{
			name:      "empty messenger name",
			messenger: "",
			profile:   "test-profile",
			wantErr:   "messenger name is required",
		},
		{
			name:      "empty profile ID",
			messenger: "telegram",
			profile:   "",
			wantErr:   "profile ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.LinkMessengerToProfile(tt.messenger, tt.profile, nil)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestManager_UnlinkMessengerFromProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	// Link first
	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Unlink
	err = mgr.UnlinkMessengerFromProfile("my-telegram", "test-profile")
	if err != nil {
		t.Fatalf("unlink failed: %v", err)
	}

	// Verify link was removed
	links := readLinksFile(t, home, "test-profile")
	if len(links) != 0 {
		t.Errorf("expected 0 links after unlink, got %d", len(links))
	}
}

func TestManager_UnlinkMessengerFromProfile_NotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.UnlinkMessengerFromProfile("nonexistent", "test-profile")
	if err == nil {
		t.Fatal("expected error for unlinking nonexistent messenger, got nil")
	}
	if !IsLinkNotFound(err) {
		t.Errorf("expected IsLinkNotFound, got %v", err)
	}
}

func TestManager_MappingConflict(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "profile-a")
	setupTestProfile(t, home, "profile-b")

	mgr := NewManager()

	// Link profile-a with channel-1
	mappingsA := map[string]string{
		"channel-1": "profile-a=handle",
	}
	err := mgr.LinkMessengerToProfile("shared-telegram", "profile-a", mappingsA)
	if err != nil {
		t.Fatalf("link profile-a failed: %v", err)
	}

	// Link profile-b with same channel-1 should conflict
	mappingsB := map[string]string{
		"channel-1": "profile-b=handle",
	}
	err = mgr.LinkMessengerToProfile("shared-telegram", "profile-b", mappingsB)
	if err == nil {
		t.Fatal("expected mapping conflict error, got nil")
	}
	if !IsMappingConflict(err) {
		t.Errorf("expected IsMappingConflict, got %v", err)
	}
}

func TestManager_MappingConflict_DifferentChannels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "profile-a")
	setupTestProfile(t, home, "profile-b")

	mgr := NewManager()

	// Link profile-a with channel-1
	mappingsA := map[string]string{
		"channel-1": "profile-a=handle",
	}
	err := mgr.LinkMessengerToProfile("shared-telegram", "profile-a", mappingsA)
	if err != nil {
		t.Fatalf("link profile-a failed: %v", err)
	}

	// Link profile-b with channel-2 should succeed (different channels)
	mappingsB := map[string]string{
		"channel-2": "profile-b=handle",
	}
	err = mgr.LinkMessengerToProfile("shared-telegram", "profile-b", mappingsB)
	if err != nil {
		t.Fatalf("link profile-b with different channel should succeed: %v", err)
	}
}

func TestManager_AddMapping(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	// Create link with no mappings
	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Add mapping
	err = mgr.AddMapping("my-telegram", "test-profile", "chan-new", "test-profile=process")
	if err != nil {
		t.Fatalf("AddMapping failed: %v", err)
	}

	// Verify
	links := readLinksFile(t, home, "test-profile")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Mappings["chan-new"] != "test-profile=process" {
		t.Errorf("mapping not added: got %q", links[0].Mappings["chan-new"])
	}
}

func TestManager_AddMapping_LinkNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.AddMapping("nonexistent", "test-profile", "chan-1", "test-profile=handle")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsLinkNotFound(err) {
		t.Errorf("expected IsLinkNotFound, got %v", err)
	}
}

func TestManager_AddMapping_ValidationErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	tests := []struct {
		name      string
		channelID string
		action    string
		wantErr   string
	}{
		{
			name:      "empty channel ID",
			channelID: "",
			action:    "test-profile=handle",
			wantErr:   "channel ID is required",
		},
		{
			name:      "empty action",
			channelID: "chan-1",
			action:    "",
			wantErr:   "action is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.AddMapping("my-telegram", "test-profile", tt.channelID, tt.action)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestManager_RemoveMapping(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	mappings := map[string]string{
		"chan-1": "test-profile=handle",
		"chan-2": "test-profile=deploy",
	}
	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", mappings)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Remove chan-1
	err = mgr.RemoveMapping("my-telegram", "test-profile", "chan-1")
	if err != nil {
		t.Fatalf("RemoveMapping failed: %v", err)
	}

	// Verify chan-1 removed but chan-2 remains
	links := readLinksFile(t, home, "test-profile")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if _, exists := links[0].Mappings["chan-1"]; exists {
		t.Error("chan-1 should have been removed")
	}
	if links[0].Mappings["chan-2"] != "test-profile=deploy" {
		t.Error("chan-2 should still exist")
	}
}

func TestManager_RemoveMapping_UnknownChannel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", map[string]string{
		"chan-1": "test-profile=handle",
	})
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	err = mgr.RemoveMapping("my-telegram", "test-profile", "nonexistent-chan")
	if err == nil {
		t.Fatal("expected error for removing nonexistent channel, got nil")
	}
	if !IsUnknownChannel(err) {
		t.Errorf("expected IsUnknownChannel, got %v", err)
	}
}

func TestManager_SetDefaultAction(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Set default action
	err = mgr.SetDefaultAction("my-telegram", "test-profile", "test-profile=catch-all")
	if err != nil {
		t.Fatalf("SetDefaultAction failed: %v", err)
	}

	// Verify
	links := readLinksFile(t, home, "test-profile")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].DefaultAction != "test-profile=catch-all" {
		t.Errorf("DefaultAction = %q, want %q", links[0].DefaultAction, "test-profile=catch-all")
	}
}

func TestManager_SetDefaultAction_LinkNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.SetDefaultAction("nonexistent", "test-profile", "test-profile=fallback")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsLinkNotFound(err) {
		t.Errorf("expected IsLinkNotFound, got %v", err)
	}
}

func TestManager_EnableDisable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Verify starts enabled
	links := readLinksFile(t, home, "test-profile")
	if !links[0].Enabled {
		t.Error("expected link to start enabled")
	}

	// Disable
	err = mgr.DisableLink("my-telegram", "test-profile")
	if err != nil {
		t.Fatalf("DisableLink failed: %v", err)
	}

	links = readLinksFile(t, home, "test-profile")
	if links[0].Enabled {
		t.Error("expected link to be disabled")
	}

	// Enable
	err = mgr.EnableLink("my-telegram", "test-profile")
	if err != nil {
		t.Fatalf("EnableLink failed: %v", err)
	}

	links = readLinksFile(t, home, "test-profile")
	if !links[0].Enabled {
		t.Error("expected link to be re-enabled")
	}
}

func TestManager_EnableDisable_LinkNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	tests := []struct {
		name string
		fn   func(string, string) error
	}{
		{"EnableLink", mgr.EnableLink},
		{"DisableLink", mgr.DisableLink},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn("nonexistent", "test-profile")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !IsLinkNotFound(err) {
				t.Errorf("expected IsLinkNotFound, got %v", err)
			}
		})
	}
}

func TestManager_ResolveChannelRoute(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "profile-a")

	mgr := NewManager()

	mappings := map[string]string{
		"chan-1": "profile-a=handle-msg",
	}
	err := mgr.LinkMessengerToProfile("my-telegram", "profile-a", mappings)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	tests := []struct {
		name          string
		messenger     string
		channelID     string
		wantAction    string
		wantErr       bool
		wantErrCheck  func(error) bool
	}{
		{
			name:       "known channel resolves",
			messenger:  "my-telegram",
			channelID:  "chan-1",
			wantAction: "profile-a=handle-msg",
		},
		{
			name:         "unknown channel returns error",
			messenger:    "my-telegram",
			channelID:    "unknown-chan",
			wantErr:      true,
			wantErrCheck: IsUnknownChannel,
		},
		{
			name:      "empty messenger name returns error",
			messenger: "",
			channelID: "chan-1",
			wantErr:   true,
		},
		{
			name:      "empty channel ID returns error",
			messenger: "my-telegram",
			channelID: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link, action, err := mgr.ResolveChannelRoute(tt.messenger, tt.channelID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErrCheck != nil && !tt.wantErrCheck(err) {
					t.Errorf("error check failed for %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if action != tt.wantAction {
				t.Errorf("action = %q, want %q", action, tt.wantAction)
			}
			if link == nil {
				t.Fatal("expected non-nil link")
			}
			if link.ProfileID != "profile-a" {
				t.Errorf("link.ProfileID = %q, want %q", link.ProfileID, "profile-a")
			}
		})
	}
}

func TestManager_ResolveChannelRoute_DisabledLink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	mappings := map[string]string{
		"chan-1": "test-profile=handle-msg",
	}
	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", mappings)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	// Disable the link
	err = mgr.DisableLink("my-telegram", "test-profile")
	if err != nil {
		t.Fatalf("disable failed: %v", err)
	}

	// Resolve should fail because link is disabled
	_, _, err = mgr.ResolveChannelRoute("my-telegram", "chan-1")
	if err == nil {
		t.Fatal("expected error when link is disabled, got nil")
	}
	if !IsUnknownChannel(err) {
		t.Errorf("expected IsUnknownChannel for disabled link, got %v", err)
	}
}

func TestManager_ResolveChannelRoute_WithDefaultAction(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	err := mgr.LinkMessengerToProfile("my-telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("link failed: %v", err)
	}

	err = mgr.SetDefaultAction("my-telegram", "test-profile", "test-profile=catch-all")
	if err != nil {
		t.Fatalf("set default failed: %v", err)
	}

	// Resolve an unmapped channel should use default action
	link, action, err := mgr.ResolveChannelRoute("my-telegram", "any-channel")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if action != "test-profile=catch-all" {
		t.Errorf("action = %q, want %q", action, "test-profile=catch-all")
	}
	if link.ProfileID != "test-profile" {
		t.Errorf("link.ProfileID = %q, want %q", link.ProfileID, "test-profile")
	}
}

func TestManager_GetProfileLinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "test-profile")

	mgr := NewManager()

	// No links initially
	links, err := mgr.GetProfileLinks("test-profile")
	if err != nil {
		t.Fatalf("GetProfileLinks failed: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}

	// Add two links
	err = mgr.LinkMessengerToProfile("telegram", "test-profile", nil)
	if err != nil {
		t.Fatalf("link telegram failed: %v", err)
	}
	err = mgr.LinkMessengerToProfile("slack", "test-profile", nil)
	if err != nil {
		t.Fatalf("link slack failed: %v", err)
	}

	links, err = mgr.GetProfileLinks("test-profile")
	if err != nil {
		t.Fatalf("GetProfileLinks failed: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links, got %d", len(links))
	}
}

func TestManager_GetMessengerLinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APS_DATA_PATH", home)
	setupTestProfile(t, home, "profile-a")
	setupTestProfile(t, home, "profile-b")

	mgr := NewManager()

	// Link same messenger to two different profiles with different channels
	err := mgr.LinkMessengerToProfile("shared-telegram", "profile-a", map[string]string{
		"chan-1": "profile-a=handle",
	})
	if err != nil {
		t.Fatalf("link profile-a failed: %v", err)
	}

	err = mgr.LinkMessengerToProfile("shared-telegram", "profile-b", map[string]string{
		"chan-2": "profile-b=handle",
	})
	if err != nil {
		t.Fatalf("link profile-b failed: %v", err)
	}

	links, err := mgr.GetMessengerLinks("shared-telegram")
	if err != nil {
		t.Fatalf("GetMessengerLinks failed: %v", err)
	}
	if len(links) != 2 {
		t.Errorf("expected 2 links across profiles, got %d", len(links))
	}
}
