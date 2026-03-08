package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// baseCtx returns a ProfileContext suitable for most resolver tests.
func baseCtx() ProfileContext {
	return ProfileContext{
		ID:          "prof-abc",
		DisplayName: "Test User",
		Email:       "test@example.com",
		ConfigDir:   "/home/test/.config/aps",
		DataDir:     "/home/test/.local/share/aps",
	}
}

func TestResolve_ServiceSplit(t *testing.T) {
	b := Bundle{
		Name: "svc-test",
		Services: []ServiceEntry{
			{Name: "alpha", Adapter: "a", Start: "always"},
			{Name: "beta", Adapter: "b", Start: "on-demand"},
			{Name: "gamma", Adapter: "c", Start: "always"},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	rb, err := Resolve(b, reg, baseCtx())
	require.NoError(t, err)

	require.Len(t, rb.AlwaysServices, 2)
	require.Len(t, rb.OnDemandServices, 1)
	assert.Equal(t, "alpha", rb.AlwaysServices[0].Name)
	assert.Equal(t, "gamma", rb.AlwaysServices[1].Name)
	assert.Equal(t, "beta", rb.OnDemandServices[0].Name)
}

func TestResolve_EnvExpansion(t *testing.T) {
	b := Bundle{
		Name: "env-test",
		Env: map[string]string{
			"MY_PROFILE": "${PROFILE_ID}",
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	ctx := baseCtx()
	ctx.ID = "my-profile-id"

	rb, err := Resolve(b, reg, ctx)
	require.NoError(t, err)

	assert.Equal(t, "my-profile-id", rb.Env["MY_PROFILE"])
}

func TestResolve_CommandExpansion(t *testing.T) {
	b := Bundle{
		Name: "cmd-test",
		Requires: []BinaryRequirement{
			{
				Binary:  "sh",
				Command: "sh --user=${PROFILE_EMAIL}",
				Missing: "warn",
			},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	ctx := baseCtx()
	ctx.Email = "user@corp.com"

	rb, err := Resolve(b, reg, ctx)
	require.NoError(t, err)

	require.Len(t, rb.BinaryResults, 1)
	assert.Equal(t, "sh --user=user@corp.com", rb.BinaryResults[0].Command)
}

func TestResolve_BinaryMissing_Skip(t *testing.T) {
	b := Bundle{
		Name: "skip-test",
		Requires: []BinaryRequirement{
			{Binary: "definitely-not-a-real-binary-xyz", Missing: "skip"},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	rb, err := Resolve(b, reg, baseCtx())
	require.NoError(t, err)

	require.Len(t, rb.BinaryResults, 1)
	assert.True(t, rb.BinaryResults[0].Skipped)
	assert.Empty(t, rb.Errors)
}

func TestResolve_BinaryMissing_Warn(t *testing.T) {
	b := Bundle{
		Name: "warn-test",
		Requires: []BinaryRequirement{
			{Binary: "definitely-not-a-real-binary-xyz", Missing: "warn"},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	rb, err := Resolve(b, reg, baseCtx())
	require.NoError(t, err)

	assert.Empty(t, rb.Errors)
	assert.NotEmpty(t, rb.Warnings)
}

func TestResolve_BinaryMissing_Error(t *testing.T) {
	b := Bundle{
		Name: "error-test",
		Requires: []BinaryRequirement{
			{Binary: "definitely-not-a-real-binary-xyz", Missing: "error"},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	rb, err := Resolve(b, reg, baseCtx())
	require.NoError(t, err) // Resolve itself does not error; errors accumulate in rb.Errors

	assert.NotEmpty(t, rb.Errors)
}

func TestResolve_BinaryBlocked(t *testing.T) {
	b := Bundle{
		Name: "blocked-test",
		Requires: []BinaryRequirement{
			{Binary: "sh", Blocked: true, Message: "sh is not allowed"},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	rb, err := Resolve(b, reg, baseCtx())
	require.NoError(t, err)

	require.Len(t, rb.BinaryResults, 1)
	assert.True(t, rb.BinaryResults[0].Blocked)
	assert.NotEmpty(t, rb.Errors)
}

func TestResolve_ScopeUnion(t *testing.T) {
	b := Bundle{
		Name: "scope-test",
		Scope: BundleScope{
			Operations: []string{"git:read", "git:write"},
			Networks:   []string{"github.com"},
		},
	}
	reg, err := NewRegistry()
	require.NoError(t, err)

	ctx := baseCtx()
	ctx.Scope = BundleScope{
		Operations: []string{"git:write", "shell:run"}, // git:write is a duplicate
		Networks:   []string{"api.github.com"},
	}

	rb, err := Resolve(b, reg, ctx)
	require.NoError(t, err)

	// Operations: union of {git:read, git:write} + {git:write, shell:run} = 3 unique
	assert.Len(t, rb.Scope.Operations, 3)
	assert.Contains(t, rb.Scope.Operations, "git:read")
	assert.Contains(t, rb.Scope.Operations, "git:write")
	assert.Contains(t, rb.Scope.Operations, "shell:run")

	// Networks: union of {github.com} + {api.github.com} = 2 unique
	assert.Len(t, rb.Scope.Networks, 2)
	assert.Contains(t, rb.Scope.Networks, "github.com")
	assert.Contains(t, rb.Scope.Networks, "api.github.com")
}

func TestResolve_Inheritance(t *testing.T) {
	// Use the built-in "developer" bundle as the parent.
	// Create a child bundle that overrides only the Description.
	child := Bundle{
		Name:        "child-dev",
		Description: "Child of developer",
		Extends:     "developer",
	}

	reg, err := NewRegistry()
	require.NoError(t, err)

	rb, err := Resolve(child, reg, baseCtx())
	require.NoError(t, err)

	// Child identity is preserved.
	assert.Equal(t, "child-dev", rb.Bundle.Name)
	assert.Equal(t, "Child of developer", rb.Bundle.Description)

	// Parent's requires are inherited (developer has several requires entries).
	assert.NotEmpty(t, rb.BinaryResults)

	// Parent's services are inherited.
	assert.NotEmpty(t, rb.AlwaysServices)
}
