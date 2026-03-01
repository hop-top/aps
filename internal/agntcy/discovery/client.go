package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"hop.top/aps/internal/core"
)

// Client wraps AGNTCY Directory operations.
// When the dir/client SDK is available, this will wrap its gRPC client.
// For now, it provides the interface and local record management.
type Client struct {
	endpoint string
}

// NewClient creates a new Directory client.
func NewClient(cfg *core.DirectoryConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("directory config is nil")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://dir.agntcy.org"
	}

	return &Client{
		endpoint: endpoint,
	}, nil
}

// Register generates an OASF record and registers it with the Directory.
func (c *Client) Register(ctx context.Context, profile *core.Profile) (map[string]interface{}, error) {
	record, err := GenerateOASFRecord(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OASF record: %w", err)
	}

	if err := ValidateOASFRecord(record); err != nil {
		return nil, fmt.Errorf("invalid OASF record: %w", err)
	}

	// TODO: When github.com/agntcy/dir/client is available, call Push() here.
	// For now, return the generated record for local inspection.
	return record, nil
}

// Deregister removes a profile's record from the Directory.
func (c *Client) Deregister(ctx context.Context, profileID string) error {
	if profileID == "" {
		return fmt.Errorf("profile ID is required")
	}

	// TODO: When github.com/agntcy/dir/client is available, call Delete() here.
	return nil
}

// Discover queries the Directory for agents matching a capability.
func (c *Client) Discover(ctx context.Context, capability string) ([]DiscoveryResult, error) {
	if capability == "" {
		return nil, fmt.Errorf("capability query is required")
	}

	// TODO: When github.com/agntcy/dir/client is available, call Search() here.
	return []DiscoveryResult{}, nil
}

// Show retrieves the OASF record for a specific profile from the Directory.
func (c *Client) Show(ctx context.Context, profileID string) (map[string]interface{}, error) {
	if profileID == "" {
		return nil, fmt.Errorf("profile ID is required")
	}

	// TODO: When github.com/agntcy/dir/client is available, call Pull() here.
	// For now, generate from local profile.
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	return GenerateOASFRecord(profile)
}

// Close closes the client connection.
func (c *Client) Close() error {
	return nil
}

// FormatRecord returns a JSON representation of an OASF record.
func FormatRecord(record map[string]interface{}) (string, error) {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
