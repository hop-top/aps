package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"hop.top/aps/internal/events"
)

// ErrProfileHasActiveSessions is returned when DeleteProfile finds active
// sessions for the profile and force is not set. The wrapped error message
// lists the blocking session IDs so callers can render a useful message.
var ErrProfileHasActiveSessions = errors.New("profile has active sessions")

// activeSessionsForProfile is a test seam: it returns the IDs of sessions
// currently in the "active" state for the given profile. The default
// implementation reads the session registry JSON directly from disk to
// avoid an import cycle with internal/core/session. Tests may override
// this variable to inject a fake.
var activeSessionsForProfile = defaultActiveSessionsForProfile

// defaultActiveSessionsForProfile reads <data>/sessions/registry.json
// and returns the IDs of any session whose profile_id matches id and
// whose status is "active". A missing OR corrupt registry is treated as
// having no active sessions: a registry that cannot be parsed cannot
// reliably block deletion, and locking out deletes on a corrupt file is
// worse than allowing them. File-read errors other than not-exist
// (e.g. permission denied) still propagate.
func defaultActiveSessionsForProfile(id string) ([]string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "sessions", "registry.json")
	// #nosec G304 -- path is constructed from core.GetDataDir(), not user input
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session registry: %w", err)
	}
	// Decode loosely so we don't depend on the session package's types.
	var raw map[string]struct {
		ProfileID string `json:"profile_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		// Corrupt registry cannot reliably block deletion; treat as empty.
		// The registry will be rewritten on the next successful session
		// mutation via T1's write-through path.
		return nil, nil //nolint:nilerr // intentional: corrupt registry should not block delete
	}
	var active []string
	for sid, info := range raw {
		if info.ProfileID == id && info.Status == "active" {
			active = append(active, sid)
		}
	}
	return active, nil
}

const (
	ApsHomeDir = ".aps"
)

type IsolationLevel string

const (
	IsolationProcess   IsolationLevel = "process"
	IsolationPlatform  IsolationLevel = "platform"
	IsolationContainer IsolationLevel = "container"
)

// Profile represents an agent profile configuration
type Profile struct {
	ID           string             `yaml:"id"`
	DisplayName  string             `yaml:"display_name"`
	Email        string             `yaml:"email,omitempty"`
	Persona      Persona            `yaml:"persona,omitempty"`
	Capabilities []string           `yaml:"capabilities,omitempty"`
	Accounts     map[string]Account `yaml:"accounts,omitempty"`
	Preferences  Preferences        `yaml:"preferences,omitempty"`
	Limits       Limits             `yaml:"limits,omitempty"`
	Git          GitConfig          `yaml:"git,omitempty"`
	SSH          SSHConfig          `yaml:"ssh,omitempty"`
	Webhooks     WebhookConfig      `yaml:"webhooks,omitempty"`
	Isolation    IsolationConfig    `yaml:"isolation,omitempty"`
	A2A           *A2AConfig          `yaml:"a2a,omitempty"`
	ACP           *ACPConfig          `yaml:"acp,omitempty"`
	Mobile        *MobileAdapterConfig `yaml:"mobile,omitempty"`
	Workspace     *WorkspaceLink      `yaml:"workspace,omitempty"`
	Observability *ObservabilityConfig `yaml:"observability,omitempty"`
	Directory     *DirectoryConfig     `yaml:"directory,omitempty"`
	Identity      *IdentityConfig      `yaml:"identity,omitempty"`
	Trust         *TrustConfig         `yaml:"trust,omitempty"`
	Voice         *VoiceConfig         `yaml:"voice,omitempty"`
	Squads        []string             `yaml:"squads,omitempty"` // squad IDs this profile belongs to
	Scope         *ScopeConfig         `yaml:"scope,omitempty"`
	Roles         []string             `yaml:"roles,omitempty"`  // owner, assignee, evaluator, auditor
	TrustLedger   *TrustLedger         `yaml:"trust_ledger,omitempty"`
}

// ScopeConfig defines access boundaries for a profile.
type ScopeConfig struct {
	FilePatterns []string `yaml:"file_patterns,omitempty"`
	Operations   []string `yaml:"operations,omitempty"`
	Tools        []string `yaml:"tools,omitempty"`
	Secrets      []string `yaml:"secrets,omitempty"`
	Networks     []string `yaml:"networks,omitempty"`
}

// WorkspaceLink associates a profile with a workspace
type WorkspaceLink struct {
	Name       string       `yaml:"name"`
	Scope      string       `yaml:"scope"`
	ScopeRules *ScopeConfig `yaml:"scope_rules,omitempty"`
}

// A2AConfig holds A2A protocol configuration for a profile
type A2AConfig struct {
	ProtocolBinding string `yaml:"protocol_binding,omitempty"`
	ListenAddr      string `yaml:"listen_addr,omitempty"`
	PublicEndpoint  string `yaml:"public_endpoint,omitempty"`
	SecurityScheme  string `yaml:"security_scheme,omitempty"`
	IsolationTier   string `yaml:"isolation_tier,omitempty"`
}

// ACPConfig holds Agent Client Protocol configuration for a profile
type ACPConfig struct {
	Enabled    bool   `yaml:"enabled,omitempty"`
	Transport  string `yaml:"transport,omitempty"` // "stdio", "http", "ws"
	ListenAddr string `yaml:"listen_addr,omitempty"`
	Port       int    `yaml:"port,omitempty"`
}

// MobileAdapterConfig holds mobile adapter linking configuration for a profile
type MobileAdapterConfig struct {
	Enabled             bool     `yaml:"enabled"`
	Port                int      `yaml:"port,omitempty"`
	MaxDevices          int      `yaml:"max_devices,omitempty"`
	DefaultExpiry       string   `yaml:"default_expiry,omitempty"`
	QRExpiry            string   `yaml:"qr_expiry,omitempty"`
	ApprovalRequired    bool     `yaml:"approval_required,omitempty"`
	AllowedCapabilities []string `yaml:"allowed_capabilities,omitempty"`
}

type Persona struct {
	Tone  string `yaml:"tone,omitempty"`
	Style string `yaml:"style,omitempty"`
	Risk  string `yaml:"risk,omitempty"`
}

type Account struct {
	Username string `yaml:"username,omitempty"`
}

type Preferences struct {
	Language string `yaml:"language,omitempty"`
	Timezone string `yaml:"timezone,omitempty"`
	Shell    string `yaml:"shell,omitempty"`
}

type Limits struct {
	MaxConcurrency    int `yaml:"max_concurrency,omitempty"`
	MaxRuntimeMinutes int `yaml:"max_runtime_minutes,omitempty"`
}

type GitConfig struct {
	Enabled bool `yaml:"enabled,omitempty"`
}

type SSHConfig struct {
	Enabled bool   `yaml:"enabled,omitempty"`
	KeyPath string `yaml:"key_path,omitempty"`
}

type WebhookConfig struct {
	AllowedEvents []string `yaml:"allowed_events,omitempty"`
}

// ObservabilityConfig holds OpenTelemetry observability configuration
type ObservabilityConfig struct {
	Exporter     string  `yaml:"exporter,omitempty"`      // "otlp", "stdout", "none"
	Endpoint     string  `yaml:"endpoint,omitempty"`      // e.g. "http://localhost:4317"
	SamplingRate float64 `yaml:"sampling_rate,omitempty"` // 0.0–1.0
}

// DirectoryConfig holds AGNTCY Directory registration configuration
type DirectoryConfig struct {
	Endpoint    string `yaml:"endpoint,omitempty"`     // Directory service URL
	AutoRegister bool  `yaml:"auto_register,omitempty"`
	AutoRefresh  bool  `yaml:"auto_refresh,omitempty"`
}

// IdentityConfig holds DID-based agent identity configuration
type IdentityConfig struct {
	DID     string   `yaml:"did,omitempty"`
	KeyPath string   `yaml:"key_path,omitempty"`
	Badges  []string `yaml:"badges,omitempty"`
}

// TrustConfig holds inbound trust verification configuration
type TrustConfig struct {
	RequireIdentity bool     `yaml:"require_identity,omitempty"`
	AllowedIssuers  []string `yaml:"allowed_issuers,omitempty"`
}

// A2AClient represents an A2A client for profile-to-profile communication
type A2AClient struct {
	targetProfileID string
}

// GetID returns the profile ID. Implements hop.top/kit/go/runtime/domain.Entity.
func (p Profile) GetID() string { return p.ID }

// CreateA2AClient creates an A2A client for communicating with another profile
func (p *Profile) CreateA2AClient(targetProfileID string) (*A2AClient, error) {
	if targetProfileID == "" {
		return nil, fmt.Errorf("target profile ID cannot be empty")
	}

	return &A2AClient{
		targetProfileID: targetProfileID,
	}, nil
}

// GetTargetProfileID returns the target profile ID
func (c *A2AClient) GetTargetProfileID() string {
	return c.targetProfileID
}

type IsolationConfig struct {
	Level     IsolationLevel  `yaml:"level"`
	Strict    bool            `yaml:"strict"`
	Fallback  bool            `yaml:"fallback"`
	Platform  PlatformConfig  `yaml:"platform,omitempty"`
	Container ContainerConfig `yaml:"container,omitempty"`
}

type PlatformConfig struct {
	SandboxID string `yaml:"sandbox_id,omitempty"`
	Name      string `yaml:"name,omitempty"`
}

type ContainerConfig struct {
	Image      string             `yaml:"image,omitempty"`
	Network    string             `yaml:"network,omitempty"`
	Volumes    []string           `yaml:"volumes,omitempty"`
	Resources  ContainerResources `yaml:"resources,omitempty"`
	BuildSteps []BuildStep        `yaml:"build_steps,omitempty"`
	Packages   []string           `yaml:"packages,omitempty"`
}

type BuildStep struct {
	Type    string `yaml:"type"`
	Run     string `yaml:"run"`
	Content string `yaml:"content,omitempty"`
}

type ContainerResources struct {
	MemoryMB int `yaml:"memory_mb,omitempty"`
	CPUQuota int `yaml:"cpu_quota,omitempty"`
}

// GetAgentsDir returns the root directory for agents.
// Delegates to GetDataDir for XDG-compliant path resolution.
func GetAgentsDir() (string, error) {
	return GetDataDir()
}

// GetProfileDir returns the directory for a specific profile
func GetProfileDir(id string) (string, error) {
	agentsDir, err := GetAgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(agentsDir, "profiles", id), nil
}

// GetProfilePath returns the path to profile.yaml for a specific profile
func GetProfilePath(id string) (string, error) {
	profileDir, err := GetProfileDir(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(profileDir, "profile.yaml"), nil
}

// LoadProfile loads a profile from disk by ID
func LoadProfile(id string) (*Profile, error) {
	path, err := GetProfilePath(id)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile %s: %w", id, err)
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile %s: %w", id, err)
	}

	if profile.ID != id {
		return nil, fmt.Errorf("profile ID mismatch: path=%s, content=%s", id, profile.ID)
	}

	if err := profile.ValidateIsolation(); err != nil {
		return nil, fmt.Errorf("invalid isolation config for profile %s: %w", id, err)
	}

	return &profile, nil
}

// ValidateIsolation validates the isolation configuration
func (p *Profile) ValidateIsolation() error {
	if p.Isolation.Level == "" {
		p.Isolation.Level = IsolationProcess
		return nil
	}

	switch p.Isolation.Level {
	case IsolationProcess:
	case IsolationPlatform:
	case IsolationContainer:
	default:
		return fmt.Errorf("invalid isolation level: %s", p.Isolation.Level)
	}

	if p.Isolation.Level == IsolationContainer {
		if p.Isolation.Container.Image == "" {
			return fmt.Errorf("container isolation requires an image")
		}
	}

	return nil
}

// IsMemberOfSquad returns true if the profile belongs to the given squad.
func (p *Profile) IsMemberOfSquad(squadID string) bool {
	for _, s := range p.Squads {
		if s == squadID {
			return true
		}
	}
	return false
}

// AddSquad adds a squad ID to the profile's membership list (deduplicated).
func (p *Profile) AddSquad(squadID string) {
	if !p.IsMemberOfSquad(squadID) {
		p.Squads = append(p.Squads, squadID)
	}
}

// RemoveSquad removes a squad ID from the profile's membership list.
func (p *Profile) RemoveSquad(squadID string) {
	squads := make([]string, 0, len(p.Squads))
	for _, s := range p.Squads {
		if s != squadID {
			squads = append(squads, s)
		}
	}
	p.Squads = squads
}

// SaveProfile saves a profile to disk
func SaveProfile(profile *Profile) error {
	if profile.ID == "" {
		return fmt.Errorf("profile ID cannot be empty")
	}

	dir, err := GetProfileDir(profile.ID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	data, err := yaml.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	path := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile.yaml: %w", err)
	}

	return nil
}

// CreateProfile creates a new profile directory and default files
func CreateProfile(id string, config Profile) error {
	dir, err := GetProfileDir(id)
	if err != nil {
		return err
	}

	// Check if already exists
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return fmt.Errorf("profile '%s' already exists", id)
	}

	// Create structure
	if err := os.MkdirAll(filepath.Join(dir, "actions"), 0755); err != nil {
		return fmt.Errorf("failed to create profile directories: %w", err)
	}

	// Handle Force: if force is true and profile exists, we might need to remove it first or just overwrite?
	// Spec T013 just says "Implement aps profile new command handler with flags"
	// Spec 12.4 says "Refuse overwrite unless --force is provided"
	// CreateProfile returns error if exists.
	// But actually, we are in core package here. The logic logic is inside CreateProfile.

	// Set default shell if not provided
	if config.Preferences.Shell == "" {
		config.Preferences.Shell = DetectShell()
	}

	// Save profile.yaml
	config.ID = id
	if err := SaveProfile(&config); err != nil {
		return err
	}

	// Create default secrets.env
	secretsPath := filepath.Join(dir, "secrets.env")
	defaultSecrets := "# Add your secrets here. Format: KEY=VALUE\n# GITHUB_TOKEN=...\n"
	if err := os.WriteFile(secretsPath, []byte(defaultSecrets), 0600); err != nil {
		return fmt.Errorf("failed to create secrets.env: %w", err)
	}

	// Create default notes.md
	notesPath := filepath.Join(dir, "notes.md")
	defaultNotes := fmt.Sprintf("# Notes for %s\n\n- Created on %s\n", config.DisplayName, "today") // Date handling could be better but sufficient for now
	if err := os.WriteFile(notesPath, []byte(defaultNotes), 0644); err != nil {
		return fmt.Errorf("failed to create notes.md: %w", err)
	}

	// Create optional gitconfig if requested (logic handled by caller usually, but we can scaffold empty one if git enabled)
	if config.Git.Enabled {
		gitConfigPath := filepath.Join(dir, "gitconfig")
		defaultGitConfig := "[user]\n\tname = " + config.DisplayName + "\n\temail = agent@example.com\n"
		if err := os.WriteFile(gitConfigPath, []byte(defaultGitConfig), 0644); err != nil {
			return fmt.Errorf("failed to create gitconfig: %w", err)
		}
	}

	publish(context.Background(), string(events.TopicProfileCreated), "", events.ProfileCreatedPayload{
		ProfileID:    id,
		DisplayName:  config.DisplayName,
		Email:        config.Email,
		Capabilities: config.Capabilities,
	})

	return nil
}

// DeleteProfile removes a profile by id. It refuses to delete if any
// session for the profile is currently active, unless force is true.
// The returned error wraps ErrProfileHasActiveSessions and lists the
// blocking session IDs so callers can present a useful message.
//
// On success, DeleteProfile removes the profile directory under
// <data>/profiles/<id>. The workspace link and squad memberships live
// inside profile.yaml itself, so they are removed implicitly when the
// directory is deleted — there is no separate persisted store to
// reverse-clean. (If a future change introduces a persisted reverse
// index for either, this function must be updated to clean it up.)
//
// DeleteProfile is not transactional. If the directory removal fails
// part-way through, the caller must clean up manually. The function
// returns the first error encountered and aborts.
//
// In force mode, DeleteProfile does NOT terminate the active sessions —
// the caller has explicitly accepted the risk that they will be left
// referencing a missing profile. Terminating sessions is the CLI's
// concern, not core's.
func DeleteProfile(id string, force bool) error {
	if id == "" {
		return fmt.Errorf("delete profile: id cannot be empty")
	}

	// Precondition 1: profile must exist (LoadProfile errors otherwise).
	if _, err := LoadProfile(id); err != nil {
		return fmt.Errorf("delete profile %q: %w", id, err)
	}

	// Precondition 2: no active sessions, unless force.
	if !force {
		active, err := activeSessionsForProfile(id)
		if err != nil {
			return fmt.Errorf("delete profile %q: check active sessions: %w", id, err)
		}
		if len(active) > 0 {
			return fmt.Errorf("%w: %d active session(s) for profile %q: %s. Terminate them first or pass force=true",
				ErrProfileHasActiveSessions, len(active), id, strings.Join(active, ", "))
		}
	}

	// Removal: profile directory. Workspace link and squad memberships
	// live inside profile.yaml and are removed with it.
	dir, err := GetProfileDir(id)
	if err != nil {
		return fmt.Errorf("delete profile %q: %w", id, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete profile %q: remove directory: %w", id, err)
	}

	publish(context.Background(), string(events.TopicProfileDeleted), "", events.ProfileDeletedPayload{
		ProfileID: id,
	})
	return nil
}

// ListProfiles scans the profiles directory and returns a list of profile IDs
func ListProfiles() ([]string, error) {
	agentsDir, err := GetAgentsDir()
	if err != nil {
		return nil, err
	}
	profilesDir := filepath.Join(agentsDir, "profiles")

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var profiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if valid profile by checking for profile.yaml
			if _, err := os.Stat(filepath.Join(profilesDir, entry.Name(), "profile.yaml")); err == nil {
				profiles = append(profiles, entry.Name())
			}
		}
	}

	return profiles, nil
}

// AddCapabilityToProfile adds a capability to a profile (deduplicates)
func AddCapabilityToProfile(profileID, capName string) error {
	profile, err := LoadProfile(profileID)
	if err != nil {
		return err
	}

	if ProfileHasCapability(profile, capName) {
		return nil // already present
	}

	profile.Capabilities = append(profile.Capabilities, capName)
	if err := SaveProfile(profile); err != nil {
		return err
	}

	publish(context.Background(), string(events.TopicProfileUpdated), "", events.ProfileUpdatedPayload{
		ProfileID: profileID,
		Fields:    []string{"capabilities"},
	})
	return nil
}

// RemoveCapabilityFromProfile removes a capability from a profile
func RemoveCapabilityFromProfile(profileID, capName string) error {
	profile, err := LoadProfile(profileID)
	if err != nil {
		return err
	}

	found := false
	caps := make([]string, 0, len(profile.Capabilities))
	for _, c := range profile.Capabilities {
		if c == capName {
			found = true
			continue
		}
		caps = append(caps, c)
	}

	if !found {
		return fmt.Errorf("capability '%s' not found on profile '%s'", capName, profileID)
	}

	profile.Capabilities = caps
	if err := SaveProfile(profile); err != nil {
		return err
	}

	publish(context.Background(), string(events.TopicProfileUpdated), "", events.ProfileUpdatedPayload{
		ProfileID: profileID,
		Fields:    []string{"capabilities"},
	})
	return nil
}

// ProfileHasCapability checks if a profile has a specific capability
func ProfileHasCapability(profile *Profile, capName string) bool {
	for _, c := range profile.Capabilities {
		if c == capName {
			return true
		}
	}
	return false
}

// ExtractBundleNames splits a capabilities list into bundle names and individual capability names.
// Entries prefixed with "bundle:" are treated as bundle references; all others are individual capabilities.
//
// Example:
//
//	capabilities:
//	  - bundle:developer
//	  - github
//
// Returns bundles=["developer"], individual=["github"].
func ExtractBundleNames(capabilities []string) (bundles []string, individual []string) {
	const prefix = "bundle:"
	for _, cap := range capabilities {
		if strings.HasPrefix(cap, prefix) {
			name := strings.TrimPrefix(cap, prefix)
			if name != "" {
				bundles = append(bundles, name)
			}
		} else {
			individual = append(individual, cap)
		}
	}
	return bundles, individual
}

// ProfilesUsingCapability returns profile IDs that have the given capability
func ProfilesUsingCapability(capName string) ([]string, error) {
	profileIDs, err := ListProfiles()
	if err != nil {
		return nil, err
	}

	var result []string
	for _, id := range profileIDs {
		profile, err := LoadProfile(id)
		if err != nil {
			continue
		}
		if ProfileHasCapability(profile, capName) {
			result = append(result, id)
		}
	}
	return result, nil
}

// MigrateProfilesFromLegacy migrates profiles from ~/.agents/profiles/ to the XDG data directory.
// Returns count of migrated profiles and any error.
func MigrateProfilesFromLegacy() (int, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}

	legacyDir := filepath.Join(home, ".agents", "profiles")
	if _, err := os.Stat(legacyDir); os.IsNotExist(err) {
		return 0, nil // nothing to migrate
	}

	newDir, err := GetDataDir()
	if err != nil {
		return 0, err
	}
	newProfilesDir := filepath.Join(newDir, "profiles")

	if legacyDir == newProfilesDir {
		return 0, nil // same path, no migration needed
	}

	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		src := filepath.Join(legacyDir, entry.Name())
		dst := filepath.Join(newProfilesDir, entry.Name())

		if _, err := os.Stat(dst); err == nil {
			continue // already exists at destination
		}

		if err := os.MkdirAll(newProfilesDir, 0755); err != nil {
			return count, err
		}

		if err := os.Rename(src, dst); err != nil {
			return count, fmt.Errorf("failed to migrate profile %s: %w", entry.Name(), err)
		}
		count++
	}

	return count, nil
}
