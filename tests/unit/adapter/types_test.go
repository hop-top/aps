package adapter_test

import (
	"testing"

	"hop.top/aps/internal/core/adapter"

	"github.com/stretchr/testify/assert"
)

func TestAdapterTypeValidation(t *testing.T) {
	tests := []struct {
		name  string
		typ   adapter.AdapterType
		valid bool
		impl  bool
	}{
		{"messenger is valid and implemented", adapter.AdapterTypeMessenger, true, true},
		{"protocol is valid and implemented", adapter.AdapterTypeProtocol, true, true},
		{"desktop is valid but not implemented", adapter.AdapterTypeDesktop, true, false},
		{"mobile is valid and implemented", adapter.AdapterTypeMobile, true, true},
		{"sense is valid but not implemented", adapter.AdapterTypeSense, true, false},
		{"actuator is valid but not implemented", adapter.AdapterTypeActuator, true, false},
		{"invalid type is not valid", adapter.AdapterType("invalid"), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, adapter.IsAdapterTypeValid(tt.typ))
			assert.Equal(t, tt.impl, adapter.IsAdapterTypeImplemented(tt.typ))
		})
	}
}

func TestImplementedAdapterTypes(t *testing.T) {
	impl := adapter.ImplementedAdapterTypes()
	assert.Contains(t, impl, adapter.AdapterTypeMessenger)
	assert.Contains(t, impl, adapter.AdapterTypeProtocol)
	assert.Contains(t, impl, adapter.AdapterTypeMobile)
	assert.NotContains(t, impl, adapter.AdapterTypeDesktop)
	assert.NotContains(t, impl, adapter.AdapterTypeSense)
}

func TestDefaultStrategyForType(t *testing.T) {
	tests := []struct {
		name     string
		typ      adapter.AdapterType
		expected adapter.LoadingStrategy
	}{
		{"messenger defaults to subprocess", adapter.AdapterTypeMessenger, adapter.StrategySubprocess},
		{"protocol defaults to builtin", adapter.AdapterTypeProtocol, adapter.StrategyBuiltin},
		{"desktop defaults to subprocess", adapter.AdapterTypeDesktop, adapter.StrategySubprocess},
		{"mobile defaults to subprocess", adapter.AdapterTypeMobile, adapter.StrategySubprocess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, adapter.DefaultStrategyForType(tt.typ))
		})
	}
}

func TestAdapterIsGlobal(t *testing.T) {
	globalAdapter := &adapter.Adapter{Scope: adapter.ScopeGlobal}
	profileAdapter := &adapter.Adapter{Scope: adapter.ScopeProfile}

	assert.True(t, globalAdapter.IsGlobal())
	assert.False(t, profileAdapter.IsGlobal())
	assert.False(t, globalAdapter.IsProfileScoped())
	assert.True(t, profileAdapter.IsProfileScoped())
}

func TestAdapterIsLinkedToProfile(t *testing.T) {
	a := &adapter.Adapter{
		Name:     "test",
		LinkedTo: []string{"profile1", "profile2"},
	}

	assert.True(t, a.IsLinkedToProfile("profile1"))
	assert.True(t, a.IsLinkedToProfile("profile2"))
	assert.False(t, a.IsLinkedToProfile("profile3"))
}
