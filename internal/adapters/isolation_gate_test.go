package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"hop.top/aps/internal/adapters/agentprotocol"
	"hop.top/aps/internal/adapters/messenger"
	"hop.top/aps/internal/adapters/ticket"
	"hop.top/aps/internal/core"
	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/core/protocol"
)

// isolationGateRunner drives one execution path that turns a profile's
// action into a running process. ran reports whether the path reported a
// completed run; msg carries the error text the path surfaced otherwise.
type isolationGateRunner func(t *testing.T, apsCore *protocol.APSAdapter, profileID, actionID string) (ran bool, msg string)

// isolationGatePaths lists every path that executes a profile action through
// the protocol core. A new path belongs here.
var isolationGatePaths = []struct {
	name string
	run  isolationGateRunner
}{
	{"protocol core", runViaProtocolCore},
	{"serve POST /v1/runs/wait", runViaServe("/v1/runs/wait", http.StatusOK)},
	{"serve POST /v1/runs", runViaServe("/v1/runs", http.StatusCreated)},
	{"messenger routing", runViaMessenger},
	{"ticket routing", runViaTicket},
}

func runViaProtocolCore(t *testing.T, apsCore *protocol.APSAdapter, profileID, actionID string) (bool, string) {
	state, err := apsCore.ExecuteRun(context.Background(),
		protocol.RunInput{ProfileID: profileID, ActionID: actionID}, nil)
	if err != nil {
		return false, err.Error()
	}
	require.NotNil(t, state)
	return state.Status == protocol.RunStatusCompleted, state.Error
}

func runViaServe(path string, wantStatus int) isolationGateRunner {
	return func(t *testing.T, apsCore *protocol.APSAdapter, profileID, actionID string) (bool, string) {
		mux := http.NewServeMux()
		require.NoError(t, agentprotocol.NewAgentProtocolAdapter().RegisterRoutes(mux, apsCore))

		body, err := json.Marshal(map[string]string{"agent_id": profileID, "action_id": actionID})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code >= http.StatusBadRequest {
			return false, w.Body.String()
		}
		var resp agentprotocol.RunResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp), w.Body.String())
		if resp.Status == "completed" {
			assert.Equal(t, wantStatus, w.Code)
			return true, ""
		}
		// A refused run is reported as a failed run carrying the message,
		// like any other start failure.
		assert.Equal(t, "failed", resp.Status, w.Body.String())
		return false, resp.Error
	}
}

type noopMessengerResolver struct{}

func (noopMessengerResolver) ResolveChannelRoute(string, string) (*msgtypes.ProfileMessengerLink, string, error) {
	return nil, "", nil
}

func runViaMessenger(t *testing.T, apsCore *protocol.APSAdapter, profileID, actionID string) (bool, string) {
	router := messenger.NewMessageRouterWithExecutor(noopMessengerResolver{}, messenger.NewNormalizer(), apsCore)
	result, err := router.ExecuteAction(context.Background(), profileID, actionID, &msgtypes.NormalizedMessage{
		ID:       "msg-1",
		Platform: "telegram",
		Sender:   msgtypes.Sender{ID: "sender-1"},
		Channel:  msgtypes.Channel{ID: "chan-1"},
	})
	if err != nil {
		return false, err.Error()
	}
	require.NotNil(t, result)
	if result.Status == "success" {
		return true, ""
	}
	return false, result.Output
}

func runViaTicket(t *testing.T, apsCore *protocol.APSAdapter, profileID, actionID string) (bool, string) {
	router := ticket.NewRouterWithExecutor(ticket.StaticRouteResolver{}, apsCore)
	result, err := router.ExecuteAction(context.Background(), profileID, actionID, &ticket.NormalizedTicket{
		ID:        "OPS-1",
		Adapter:   ticket.AdapterJira,
		Kind:      ticket.TicketKindIssue,
		ChannelID: "OPS",
		ThreadID:  "OPS-1",
		Author:    ticket.Actor{ID: "acc-1"},
	})
	if err != nil {
		return false, err.Error()
	}
	require.NotNil(t, result)
	if result.Status == ticket.StatusSuccess {
		return true, ""
	}
	return false, result.Output
}

// writeIsolationProfile writes a profile whose "hello" action creates a
// sentinel file, and returns the profile id and the sentinel path.
func writeIsolationProfile(t *testing.T, dataDir, id, isolationYAML string) string {
	t.Helper()
	profileDir := filepath.Join(dataDir, "profiles", id)
	require.NoError(t, os.MkdirAll(filepath.Join(profileDir, "actions"), 0o755))

	sentinel := filepath.Join(dataDir, "sentinel-"+id)
	yaml := "id: " + id + "\ndisplay_name: " + id + "\n" + isolationYAML
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "profile.yaml"), []byte(yaml), 0o600))
	script := "#!/bin/sh\ntouch '" + sentinel + "'\necho ran\n"
	require.NoError(t, os.WriteFile(filepath.Join(profileDir, "actions", "hello.sh"), []byte(script), 0o755))
	return sentinel
}

func TestIsolationGate_ExecutionPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh action scripts")
	}

	cases := []struct {
		name      string
		isolation string
		wantRan   bool
		wantMsg   string
	}{
		{"platform refused", "isolation:\n  level: platform\n", false, "platform isolation not yet implemented"},
		{"container refused", "isolation:\n  level: container\n  container:\n    image: alpine:3\n", false, "container isolation not yet implemented"},
		{"process runs", "isolation:\n  level: process\n", true, ""},
		{"unset level runs as process", "", true, ""},
	}

	for _, path := range isolationGatePaths {
		for _, tc := range cases {
			t.Run(path.name+"/"+tc.name, func(t *testing.T) {
				dataDir := t.TempDir()
				t.Setenv("APS_DATA_PATH", dataDir)
				t.Setenv("HOME", dataDir)
				apsCore, err := protocol.NewAPSAdapter()
				require.NoError(t, err)
				sentinel := writeIsolationProfile(t, dataDir, "gated", tc.isolation)

				ran, msg := path.run(t, apsCore, "gated", "hello")

				_, statErr := os.Stat(sentinel)
				if tc.wantRan {
					assert.True(t, ran, msg)
					assert.NoError(t, statErr, "action should have run")
					return
				}
				assert.False(t, ran)
				assert.Contains(t, msg, tc.wantMsg)
				assert.True(t, os.IsNotExist(statErr), "refused profile must not start the action")
			})
		}
	}
}

// An unknown level is rejected when the profile loads, so no path reaches
// execution with it.
func TestIsolationGate_UnknownLevelFailsAtLoad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh action scripts")
	}
	dataDir := t.TempDir()
	t.Setenv("APS_DATA_PATH", dataDir)
	t.Setenv("HOME", dataDir)
	sentinel := writeIsolationProfile(t, dataDir, "weird", "isolation:\n  level: quantum\n")

	_, err := core.LoadProfile("weird")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid isolation level")

	apsCore, err := protocol.NewAPSAdapter()
	require.NoError(t, err)
	for _, path := range isolationGatePaths {
		ran, _ := path.run(t, apsCore, "weird", "hello")
		assert.False(t, ran, path.name)
	}
	_, statErr := os.Stat(sentinel)
	assert.True(t, os.IsNotExist(statErr))
}
