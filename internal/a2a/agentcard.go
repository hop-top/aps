package a2a

import (
	"encoding/json"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/skills"
)

// GenerateAgentCardFromProfile generates an A2A Agent Card from an APS profile
func GenerateAgentCardFromProfile(profile *core.Profile) (*a2a.AgentCard, error) {
	if profile.A2A == nil || !profile.A2A.Enabled {
		return nil, ErrA2ANotEnabled
	}

	listenAddr := getOrDefault(profile.A2A.ListenAddr, "127.0.0.1:8081")
	protocolBinding := getOrDefault(profile.A2A.ProtocolBinding, "jsonrpc")
	transport := mapProtocolBindingToTransport(protocolBinding)

	agentSkills := generateAgentSkills(profile)

	card := &a2a.AgentCard{
		Name:               profile.DisplayName,
		Description:        fmt.Sprintf("APS Profile: %s", profile.DisplayName),
		Version:            "1.0.0",
		ProtocolVersion:    "0.3.4",
		URL:                fmt.Sprintf("http://%s", listenAddr),
		PreferredTransport: transport,
		Capabilities: a2a.AgentCapabilities{
			Streaming: false,
		},
		Skills:          agentSkills,
		SecuritySchemes: nil,
		Provider: &a2a.AgentProvider{
			Org: "APS",
			URL: "https://github.com/oss-aps-cli",
		},
	}

	if err := validateAgentCard(card); err != nil {
		return nil, fmt.Errorf("invalid agent card: %w", err)
	}

	return card, nil
}

// GenerateAgentCardForProfile generates an Agent Card for a profile by ID
func GenerateAgentCardForProfile(profileID string) (*a2a.AgentCard, error) {
	profile, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	return GenerateAgentCardFromProfile(profile)
}

// generateAgentSkills generates agent skills from profile capabilities and Agent Skills
func generateAgentSkills(profile *core.Profile) []a2a.AgentSkill {
	skillsList := make([]a2a.AgentSkill, 0)

	// Add Agent Skills (from the skills system)
	agentSkills := GenerateSkillCapabilities(profile.ID)
	skillsList = append(skillsList, agentSkills...)

	// Add profile capabilities as skills
	for _, cap := range profile.Capabilities {
		skillsList = append(skillsList, a2a.AgentSkill{
			ID:          cap,
			Name:        cap,
			Description: fmt.Sprintf("APS capability: %s", cap),
			Examples:    []string{"Execute " + cap},
		})
	}

	// If no skills at all, add default
	if len(skillsList) == 0 {
		skillsList = append(skillsList, a2a.AgentSkill{
			ID:          "execute",
			Name:        "execute",
			Description: "Execute commands in isolated environment",
			Examples:    []string{"Execute shell commands"},
		})
	}

	return skillsList
}

// GenerateSkillCapabilities generates A2A AgentSkill entries from the skills registry
func GenerateSkillCapabilities(profileID string) []a2a.AgentSkill {
	result := make([]a2a.AgentSkill, 0)

	// Create registry and discover skills
	registry := skills.NewRegistry(profileID, []string{}, false)
	if err := registry.Discover(); err != nil {
		// If discovery fails, return empty list
		return result
	}

	// Convert each skill to A2A AgentSkill format
	for _, skill := range registry.List() {
		scriptList, _ := skill.ListScripts()

		// Create examples from scripts
		examples := make([]string, 0)
		for _, script := range scriptList {
			examples = append(examples, fmt.Sprintf("Execute %s", script))
		}

		// If no scripts, provide a generic example
		if len(examples) == 0 {
			examples = append(examples, fmt.Sprintf("Use %s skill", skill.Name))
		}

		agentSkill := a2a.AgentSkill{
			ID:          skill.Name,
			Name:        skill.Name,
			Description: skill.Description,
			Examples:    examples,
		}

		result = append(result, agentSkill)
	}

	return result
}

// generateAgentInterfaces generates agent interface configurations
func generateAgentInterfaces(profile *core.Profile, listenAddr, protocolBinding string) []a2a.AgentInterface {
	interfaces := make([]a2a.AgentInterface, 0)

	interfaces = append(interfaces, a2a.AgentInterface{
		Transport: a2a.TransportProtocol(protocolBinding),
		URL:       fmt.Sprintf("http://%s", listenAddr),
	})

	return interfaces
}

// getOrDefault returns value or default
func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// mapProtocolBindingToTransport maps config protocol binding to SDK transport constant
func mapProtocolBindingToTransport(binding string) a2a.TransportProtocol {
	switch binding {
	case "jsonrpc":
		return a2a.TransportProtocolJSONRPC
	case "grpc":
		return a2a.TransportProtocolGRPC
	case "http", "http+json":
		return a2a.TransportProtocolHTTPJSON
	default:
		return a2a.TransportProtocolJSONRPC
	}
}

// validateAgentCard validates that required fields are set
func validateAgentCard(card *a2a.AgentCard) error {
	if card.Name == "" {
		return ErrInvalidAgentCard("name is required")
	}

	if card.URL == "" {
		return ErrInvalidAgentCard("url is required")
	}

	if len(card.Skills) == 0 {
		return ErrInvalidAgentCard("at least one skill is required")
	}

	return nil
}

// SerializeAgentCardToJSON serializes an Agent Card to JSON
func SerializeAgentCardToJSON(card *a2a.AgentCard) ([]byte, error) {
	return json.MarshalIndent(card, "", "  ")
}
