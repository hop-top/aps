package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureIsolationEnforceable(t *testing.T) {
	cases := []struct {
		level   IsolationLevel
		wantErr string
	}{
		{"", ""},
		{IsolationProcess, ""},
		{IsolationPlatform, "platform isolation not yet implemented"},
		{IsolationContainer, "container isolation not yet implemented"},
		{"quantum", "unsupported isolation level: quantum"},
	}
	for _, tc := range cases {
		t.Run(string(tc.level), func(t *testing.T) {
			p := &Profile{ID: "p", Isolation: IsolationConfig{Level: tc.level}}
			err := p.EnsureIsolationEnforceable()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.wantErr, err.Error())
		})
	}
}
