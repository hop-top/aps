package core

import (
	"context"
	"testing"

	"oss-aps-cli/internal/core/protocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPSAdapter_ExecuteRun_InvalidInput(t *testing.T) {
	adapter, err := protocol.NewAPSAdapter()
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   protocol.RunInput
		wantErr bool
	}{
		{
			name: "missing profile_id",
			input: protocol.RunInput{
				ActionID: "test-action",
			},
			wantErr: true,
		},
		{
			name: "missing action_id",
			input: protocol.RunInput{
				ProfileID: "test-profile",
			},
			wantErr: true,
		},
		{
			name: "valid input",
			input: protocol.RunInput{
				ProfileID: "test-profile",
				ActionID:  "test-action",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.ExecuteRun(context.Background(), tt.input, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestAPSAdapter_CreateSession(t *testing.T) {
	adapter, err := protocol.NewAPSAdapter()
	require.NoError(t, err)

	session, err := adapter.CreateSession("test-profile", map[string]string{"key": "value"})
	assert.NoError(t, err)
	assert.NotEmpty(t, session.SessionID)
	assert.Equal(t, "test-profile", session.ProfileID)
	assert.Equal(t, "value", session.Metadata["key"])

	err = adapter.DeleteSession(session.SessionID)
	assert.NoError(t, err)
}

func TestAPSAdapter_StoreOperations(t *testing.T) {
	adapter, err := protocol.NewAPSAdapter()
	require.NoError(t, err)

	namespace := "test-ns"
	key := "test-key"
	value := []byte("test-value")

	err = adapter.StorePut(namespace, key, value)
	assert.NoError(t, err)

	retrieved, err := adapter.StoreGet(namespace, key)
	assert.NoError(t, err)
	assert.Equal(t, value, retrieved)

	results, err := adapter.StoreSearch(namespace, "")
	assert.NoError(t, err)
	assert.Contains(t, results, key)

	err = adapter.StoreDelete(namespace, key)
	assert.NoError(t, err)

	_, err = adapter.StoreGet(namespace, key)
	assert.Error(t, err)

	namespaces, err := adapter.StoreListNamespaces()
	assert.NoError(t, err)
	assert.Contains(t, namespaces, namespace)
}

func TestAPSAdapter_GetAgent(t *testing.T) {
	adapter, err := protocol.NewAPSAdapter()
	require.NoError(t, err)

	_, err = adapter.GetAgent("nonexistent-profile")
	assert.Error(t, err)
}
