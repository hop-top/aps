package core

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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
	Squads        []string             `yaml:"squads,omitempty"` // squad IDs this profile belongs to
}

// WorkspaceLink associates a profile with a workspace
type WorkspaceLink struct {
	Name  string `yaml:"name"`
	Scope string `yaml:"scope"` // "global" or "profile"
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

// GetAgentsDir returns the root directory for agents (~/.agents)
func GetAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agents"), nil
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
	return SaveProfile(profile)
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
	return SaveProfile(profile)
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
