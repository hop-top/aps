package a2a

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultStorageConfig(t *testing.T) {
	config := DefaultStorageConfig()

	require.NotNil(t, config)
	assert.NotEmpty(t, config.BasePath)
	assert.Contains(t, config.BasePath, ".agents")
	assert.Contains(t, config.BasePath, "a2a")
}

func TestValidateProfileConfig(t *testing.T) {
	tests := []struct {
		name            string
		enabled         bool
		protocolBinding string
		securityScheme  string
		isolationTier   string
		wantErr         bool
	}{
		{
			name:            "valid config",
			enabled:         true,
			protocolBinding: "jsonrpc",
			securityScheme:  "apikey",
			isolationTier:   "process",
			wantErr:         false,
		},
		{
			name:            "missing protocol binding",
			enabled:         true,
			protocolBinding: "",
			securityScheme:  "apikey",
			isolationTier:   "process",
			wantErr:         true,
		},
		{
			name:            "invalid protocol binding",
			enabled:         true,
			protocolBinding: "invalid",
			securityScheme:  "apikey",
			isolationTier:   "process",
			wantErr:         true,
		},
		{
			name:            "invalid security scheme",
			enabled:         true,
			protocolBinding: "jsonrpc",
			securityScheme:  "invalid",
			isolationTier:   "process",
			wantErr:         true,
		},
		{
			name:            "invalid isolation tier",
			enabled:         true,
			protocolBinding: "jsonrpc",
			securityScheme:  "apikey",
			isolationTier:   "invalid",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileConfig(tt.enabled, tt.protocolBinding, tt.securityScheme, tt.isolationTier)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProfileConfig_Disabled(t *testing.T) {
	err := ValidateProfileConfig(false, "", "", "")
	assert.NoError(t, err, "disabled config should always validate")
}
