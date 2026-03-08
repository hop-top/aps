package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"os/exec"
	"testing"
	"time"

	"hop.top/aps/internal/adapters/agentprotocol"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testServer struct {
	cmd    *exec.Cmd
	url    string
	cancel context.CancelFunc
}

func startTestServer(t *testing.T, home string, port string) *testServer {
	t.Helper()

	tmpDir := t.TempDir()
	stdoutPath := tmpDir + "/server-stdout.log"
	stderrPath := tmpDir + "/server-stderr.log"

	stdout, err := os.Create(stdoutPath)
	require.NoError(t, err)
	stderr, err := os.Create(stderrPath)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, apsBinary, "serve", "--addr", "127.0.0.1:"+port)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("HOME=%s", home),
		fmt.Sprintf("USERPROFILE=%s", home),
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Start()
	require.NoError(t, err, "Failed to start server")

	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()

		stdout.Close()
		stderr.Close()

		stdoutContent, _ := os.ReadFile(stdoutPath)
		stderrContent, _ := os.ReadFile(stderrPath)

		if len(stdoutContent) > 0 {
			t.Logf("Server stdout: %s", string(stdoutContent))
		}
		if len(stderrContent) > 0 {
			t.Logf("Server stderr: %s", string(stderrContent))
		}
	})

	maxRetries := 20
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			resp.Body.Close()
			time.Sleep(500 * time.Millisecond)

			return &testServer{
				cmd:    cmd,
				url:    "http://127.0.0.1:" + port,
				cancel: cancel,
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("Server failed to start after %d retries", maxRetries)
	return nil
}

func createTestProfileAndAction(t *testing.T, home, profileID, actionID, actionScript string) {
	t.Helper()

	stdout, _, err := runAPS(t, home, "profile", "new", profileID, "--display-name", profileID)
	require.NoError(t, err, "Failed to create profile %s: %v\n%s", profileID, err, stdout)
	assert.Contains(t, stdout, "created successfully")

	addTestAction(t, home, profileID, actionID, actionScript)
}

func addTestAction(t *testing.T, home, profileID, actionID, actionScript string) {
	t.Helper()

	actionsDir := home + "/.agents/profiles/" + profileID + "/actions"
	err := os.MkdirAll(actionsDir, 0755)
	require.NoError(t, err)

	actionPath := actionsDir + "/" + actionID + ".sh"
	err = os.WriteFile(actionPath, []byte(actionScript), 0755)
	require.NoError(t, err)
}

func TestAgentProtocol_UserStory1_StatelessRun(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18080")
	baseURL := server.url

	createTestProfileAndAction(t, home, "myagent", "hello", `#!/bin/sh
echo "Hello, World!"`)

	t.Run("POST /runs/wait returns action output", func(t *testing.T) {
		t.Parallel()
		body := map[string]interface{}{
			"agent_id":  "myagent",
			"action_id": "hello",
			"input":     map[string]interface{}{},
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", baseURL+"/v1/runs/wait", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result agentprotocol.RunResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.NotEmpty(t, result.RunID)
		assert.Equal(t, "completed", result.Status)
	})

	t.Run("Non-existent profile returns HTTP 404", func(t *testing.T) {
		t.Parallel()
		body := map[string]interface{}{
			"agent_id":  "non-existent-agent",
			"action_id": "test",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", baseURL+"/v1/runs/wait", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Failed action returns HTTP 200 with status: failed", func(t *testing.T) {
		t.Parallel()
		createTestProfileAndAction(t, home, "fail-agent", "fail-action", `#!/bin/sh
echo "This action fails" >&2
exit 1`)

		body := map[string]interface{}{
			"agent_id":  "fail-agent",
			"action_id": "fail-action",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", baseURL+"/v1/runs/wait", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result agentprotocol.RunResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "failed", result.Status)
		assert.NotEmpty(t, result.Error)
	})
}

func TestAgentProtocol_UserStory2_StreamingRun(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18081")
	baseURL := server.url

	createTestProfileAndAction(t, home, "stream-agent", "longrun", `#!/bin/sh
for i in 1 2 3; do
  echo "Progress: $i"
  sleep 0.5
done`)

	t.Run("POST /runs/stream receives SSE events", func(t *testing.T) {
		t.Parallel()
		body := map[string]interface{}{
			"agent_id":  "stream-agent",
			"action_id": "longrun",
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", baseURL+"/v1/runs/stream", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

		eventsReceived := 0
		reader := resp.Body
		buf := make([]byte, 2048)

		for {
			n, err := reader.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Logf("Read error: %v", err)
				break
			}
			data := string(buf[:n])
			if strings.Contains(data, "event:") && strings.Contains(data, "data:") {
				eventsReceived++
				t.Logf("Received SSE event: %s", data)
			}
		}

		assert.Greater(t, eventsReceived, 0, "Should receive at least one SSE event")
	})
}

func TestAgentProtocol_UserStory4_AgentDiscovery(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18082")
	baseURL := server.url

	stdout, _, err := runAPS(t, home, "profile", "new", "agent-a", "--display-name", "Agent A")
	require.NoError(t, err)
	assert.Contains(t, stdout, "created successfully")

	stdout, _, err = runAPS(t, home, "profile", "new", "agent-b", "--display-name", "Agent B")
	require.NoError(t, err)
	assert.Contains(t, stdout, "created successfully")

	t.Run("POST /agents/search returns both agents", func(t *testing.T) {
		t.Parallel()
		body := agentprotocol.AgentSearchRequest{}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", baseURL+"/v1/agents/search", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result agentprotocol.AgentSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Agents), 2)

		agentIDs := make(map[string]bool)
		for _, agent := range result.Agents {
			agentIDs[agent.ID] = true
		}
		assert.True(t, agentIDs["agent-a"], "Should include agent-a")
		assert.True(t, agentIDs["agent-b"], "Should include agent-b")
	})

	t.Run("GET /agents/agent-a/schemas returns JSON Schema", func(t *testing.T) {
		t.Parallel()
		addTestAction(t, home, "agent-a", "test", `#!/bin/sh
echo "test"`)

		req, _ := http.NewRequest("GET", baseURL+"/v1/agents/agent-a/schemas", nil)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "agent-a", result["agent_id"])
		assert.Contains(t, result, "schemas")
	})
}

func TestAgentProtocol_UserStory5_ThreadSessionManagement(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18083")
	baseURL := server.url

	createTestProfileAndAction(t, home, "thread-agent", "hello", `#!/bin/sh
echo "Hello from thread!"`)

	t.Run("POST /threads creates session", func(t *testing.T) {
		t.Parallel()
		body := agentprotocol.CreateThreadRequest{
			AgentID:  "thread-agent",
			Metadata: map[string]interface{}{"user": "test"},
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", baseURL+"/v1/threads", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result agentprotocol.ThreadResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.NotEmpty(t, result.ThreadID)
		assert.Equal(t, "thread-agent", result.AgentID)
		assert.Equal(t, "test", result.Metadata["user"])
	})
}

func TestAgentProtocol_StoreOperations(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18084")
	baseURL := server.url

	t.Run("Store put and get", func(t *testing.T) {
		t.Parallel()
		putBody := agentprotocol.StorePutRequest{
			Namespace: "test-ns",
			Key:       "test-key",
			Value:     "test-value",
		}
		jsonBody, _ := json.Marshal(putBody)

		client := &http.Client{Timeout: 5 * time.Second}

		putReq, _ := http.NewRequest("PUT", baseURL+"/v1/store", bytes.NewReader(jsonBody))
		putReq.Header.Set("Content-Type", "application/json")
		putResp, err := client.Do(putReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, putResp.StatusCode)

		getReq, _ := http.NewRequest("GET", baseURL+"/v1/store/test-ns/test-key", nil)
		getResp, err := client.Do(getReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, getResp.StatusCode)

		var getResult agentprotocol.StoreItem
		err = json.NewDecoder(getResp.Body).Decode(&getResult)
		require.NoError(t, err)

		assert.Equal(t, "test-ns", getResult.Namespace)
		assert.Equal(t, "test-key", getResult.Key)
		assert.Equal(t, "test-value", getResult.Value)
	})
}

func TestAgentProtocol_BackgroundRun(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18085")
	baseURL := server.url

	createTestProfileAndAction(t, home, "bg-agent", "quick", `#!/bin/sh
echo "Quick task done"`)

	t.Run("POST /runs/background starts run in background", func(t *testing.T) {
		t.Parallel()
		body := map[string]interface{}{
			"agent_id":  "bg-agent",
			"action_id": "quick",
		}
		jsonBody, _ := json.Marshal(body)

		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("POST", baseURL+"/v1/runs/background", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})
}

func TestAgentProtocol_ErrorHandling(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	server := startTestServer(t, home, "18086")
	baseURL := server.url

	t.Run("Invalid JSON returns 400", func(t *testing.T) {
		t.Parallel()
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("POST", baseURL+"/v1/runs/wait", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid run ID returns 404", func(t *testing.T) {
		t.Parallel()
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("GET", baseURL+"/v1/runs/non-existent-id", nil)
		resp, err := client.Do(req)

		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
