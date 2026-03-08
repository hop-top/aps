package squad

// InheritedConfig represents configuration inherited from squad to member profile.
type InheritedConfig struct {
	Domain            string    `json:"domain" yaml:"domain"`
	SquadType         SquadType `json:"squad_type" yaml:"squad_type"`
	GoldenPathDefined bool      `json:"golden_path_defined" yaml:"golden_path_defined"`
	Capabilities      []string  `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
}

// GetInheritedConfigForProfile returns the merged config from all squads a profile belongs to.
// If a profile belongs to multiple squads, capabilities are merged (union),
// and the first squad's domain/type take precedence.
func GetInheritedConfigForProfile(mgr *Manager, profileID string) (*InheritedConfig, error) {
	squads := mgr.GetSquadsForProfile(profileID)
	if len(squads) == 0 {
		return nil, nil
	}

	config := &InheritedConfig{
		Domain:            squads[0].Domain,
		SquadType:         squads[0].Type,
		GoldenPathDefined: squads[0].GoldenPathDefined,
	}

	// Merge capabilities from all squads (deduplicated)
	seen := map[string]bool{}
	for _, s := range squads {
		// If any squad has golden path, mark true
		if s.GoldenPathDefined {
			config.GoldenPathDefined = true
		}
	}
	_ = seen // capabilities will be added when squads gain capability fields

	return config, nil
}
