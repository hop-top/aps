package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry_LoadsBuiltins(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	all := reg.List()
	names := make([]string, len(all))
	for i, b := range all {
		names[i] = b.Name
	}

	assert.Contains(t, names, "developer")
	assert.Contains(t, names, "reader")
	assert.Contains(t, names, "ops")
	assert.Contains(t, names, "comms")
	assert.Contains(t, names, "mobile")
	assert.Contains(t, names, "agntcy")
}

func TestRegistry_Get_Found(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	b, err := reg.Get("developer")
	require.NoError(t, err)
	assert.Equal(t, "developer", b.Name)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	_, err = reg.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRegistry_List_ReturnsAll(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	all := reg.List()
	assert.GreaterOrEqual(t, len(all), 6)
}

func TestRegistry_Validate_Valid(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	b := &Bundle{
		Name:        "test-valid",
		Description: "A valid test bundle",
		Requires: []BinaryRequirement{
			{Binary: "git", Missing: "warn", DenyPolicy: "strip"},
		},
	}

	err = reg.Validate(b)
	assert.NoError(t, err)
}

func TestRegistry_Validate_EmptyName(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	b := &Bundle{
		Name:        "",
		Description: "No name",
	}

	err = reg.Validate(b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestRegistry_Validate_InvalidMissingPolicy(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	b := &Bundle{
		Name: "test-bad-missing",
		Requires: []BinaryRequirement{
			{Binary: "git", Missing: "ignore"}, // not a valid policy
		},
	}

	err = reg.Validate(b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid missing policy")
}

func TestRegistry_Validate_InvalidDenyPolicy(t *testing.T) {
	reg, err := NewRegistry()
	require.NoError(t, err)

	b := &Bundle{
		Name: "test-bad-deny",
		Requires: []BinaryRequirement{
			{Binary: "git", DenyPolicy: "block"}, // not a valid deny_policy
		},
	}

	err = reg.Validate(b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid deny_policy")
}
