package a2a

import (
	"context"
	"fmt"

	a2a "github.com/a2aproject/a2a-go/a2a"
	a2aclient "github.com/a2aproject/a2a-go/a2aclient"

	"hop.top/aps/internal/core"
)

// Resolver handles resolving Agent Cards for profiles
type Resolver struct {
	storage *Storage
}

// NewResolver creates a new Agent Card resolver
func NewResolver(storage *Storage) *Resolver {
	return &Resolver{
		storage: storage,
	}
}

// ResolveProfile resolves Agent Card for a target profile
func (r *Resolver) ResolveProfile(ctx context.Context, targetProfileID string) (*a2a.AgentCard, error) {
	if targetProfileID == "" {
		return nil, ErrInvalidConfig
	}

	profile, err := core.LoadProfile(targetProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %s: %w", targetProfileID, err)
	}

	if !core.ProfileHasCapability(profile, "a2a") {
		return nil, ErrA2ANotEnabled
	}

	card, err := r.storage.GetAgentCard(targetProfileID)
	if err == nil {
		return card, nil
	}

	if err != ErrAgentCardNotFound {
		return nil, fmt.Errorf("failed to get cached agent card: %w", err)
	}

	generatedCard, err := GenerateAgentCardFromProfile(profile)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent card: %w", err)
	}

	if err := r.storage.SaveAgentCard(targetProfileID, generatedCard); err != nil {
		return nil, fmt.Errorf("failed to cache agent card: %w", err)
	}

	return generatedCard, nil
}

// ResolveFromCard creates a client from a given Agent Card
func (r *Resolver) ResolveFromCard(ctx context.Context, card *a2a.AgentCard) (*a2aclient.Client, error) {
	if card == nil {
		return nil, ErrInvalidAgentCard("card cannot be nil")
	}

	client, err := a2aclient.NewFromCard(ctx, card)
	if err != nil {
		return nil, ErrClientFailed("create from card", err)
	}

	return client, nil
}

// InvalidateCache removes cached Agent Card for a profile
func (r *Resolver) InvalidateCache(profileID string) error {
	if profileID == "" {
		return ErrInvalidConfig
	}

	return r.storage.DeleteAgentCard(profileID)
}
