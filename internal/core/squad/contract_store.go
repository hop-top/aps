package squad

import (
	"fmt"
	"sync"
	"time"
)

// ContractStore provides in-memory CRUD for interaction contracts.
// Follows collaboration/registry.go pattern.
type ContractStore struct {
	mu        sync.RWMutex
	contracts map[string]*Contract
}

// NewContractStore creates an empty ContractStore.
func NewContractStore() *ContractStore {
	return &ContractStore{contracts: make(map[string]*Contract)}
}

// Create validates and stores a new contract.
func (s *ContractStore) Create(c Contract) error {
	if err := c.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.contracts[c.ID]; exists {
		return fmt.Errorf("contract already exists: %s", c.ID)
	}
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	cp := c
	s.contracts[c.ID] = &cp
	return nil
}

// Get returns a copy of the contract with the given ID.
func (s *ContractStore) Get(id string) (*Contract, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.contracts[id]
	if !ok {
		return nil, fmt.Errorf("contract not found: %s", id)
	}
	cp := *c
	return &cp, nil
}

// ListBySquad returns all contracts where squadID is provider or consumer.
func (s *ContractStore) ListBySquad(squadID string) []Contract {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Contract
	for _, c := range s.contracts {
		if c.ProviderSquad == squadID || c.ConsumerSquad == squadID {
			out = append(out, *c)
		}
	}
	return out
}

// ListAll returns all contracts.
func (s *ContractStore) ListAll() []Contract {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Contract, 0, len(s.contracts))
	for _, c := range s.contracts {
		out = append(out, *c)
	}
	return out
}

// Update validates and updates an existing contract.
func (s *ContractStore) Update(c Contract) error {
	if err := c.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.contracts[c.ID]
	if !ok {
		return fmt.Errorf("contract not found: %s", c.ID)
	}
	c.CreatedAt = existing.CreatedAt
	c.UpdatedAt = time.Now()
	cp := c
	s.contracts[c.ID] = &cp
	return nil
}

// Delete removes a contract by ID.
func (s *ContractStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.contracts[id]; !ok {
		return fmt.Errorf("contract not found: %s", id)
	}
	delete(s.contracts, id)
	return nil
}
