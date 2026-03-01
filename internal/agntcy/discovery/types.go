package discovery

// DiscoveryResult represents an agent discovered via the AGNTCY Directory.
type DiscoveryResult struct {
	Name         string                 `json:"name"`
	DID          string                 `json:"did,omitempty"`
	Endpoint     string                 `json:"endpoint"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Record       map[string]interface{} `json:"record,omitempty"`
}
