package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookServer(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "hook-agent"

	// Create profile and action
	_, _, err := runAPS(t, home, "profile", "new", profileID)
	require.NoError(t, err)

	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	scriptPath := filepath.Join(actionsDir, "hook.sh")
	// Create a script that creates a file to prove it ran
	proofFile := filepath.Join(home, "proof.txt")
	err = os.WriteFile(scriptPath, []byte(fmt.Sprintf("#!/bin/sh\necho ran > %s", proofFile)), 0755)
	require.NoError(t, err)

	// Start server
	// We need a random port. We can rely on system picking one if we pass :0,
	// but we need to know it to send request.
	// APS CLI logs "listening on ..." to stderr/stdout.
	// We can parse it?
	// Or we can just pick a fixed high port (e.g. 18080) but that risks collision in parallel tests.
	// Better: Bind a listener, get port, close listener, pass port to APS. Race condition possible but rare.
	// Or use --addr 127.0.0.1:0 and parse output.

	// Let's use 127.0.0.1:0 and parse output.
	cmd := prepareAPS(t, home, nil, "webhook", "server", "--addr", "127.0.0.1:0", "--event-map", "test.event="+profileID+":hook", "--secret", "supersecret")

	// We need to capture stdout/stderr to find the port.
	// But cmd.Start() doesn't block. We need to read the output stream.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr // Logs go to stderr
	// cmd.Stdout = os.Stdout

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Wait for startup (poll stderr)
	var port string
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		logs := stderr.String()
		if idx := strings.Index(logs, "listening on 127.0.0.1:"); idx != -1 {
			// Extract port
			rest := logs[idx+len("listening on 127.0.0.1:"):]
			// find next newline
			if end := strings.IndexByte(rest, '\n'); end != -1 {
				port = rest[:end]
				port = strings.TrimSpace(port) // Trim newline
				break
			}
		}
	}
	require.NotEmpty(t, port, "Failed to find port in logs: "+stderr.String())

	baseURL := fmt.Sprintf("http://127.0.0.1:%s/webhook", port)

	// Test 1: Missing Signature (401)
	resp, err := http.Post(baseURL, "application/json", bytes.NewBufferString("{}"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Test 2: Invalid Signature (401)
	req, _ := http.NewRequest("POST", baseURL, bytes.NewBufferString("{}"))
	req.Header.Set("X-APS-Event", "test.event")
	req.Header.Set("X-APS-Signature", "sha256=invalid")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Test 3: Valid Signature, Missing Event (400)
	payload := []byte(`{"foo":"bar"}`)
	mac := hmac.New(sha256.New, []byte("supersecret"))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	req, _ = http.NewRequest("POST", baseURL, bytes.NewBuffer(payload))
	req.Header.Set("X-APS-Signature", "sha256="+sig)
	// No event header
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test 4: Valid Request (200) + Action Trigger
	req, _ = http.NewRequest("POST", baseURL, bytes.NewBuffer(payload))
	req.Header.Set("X-APS-Event", "test.event")
	req.Header.Set("X-APS-Signature", "sha256="+sig)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify action ran (check proof file)
	// Give it a moment to flush
	time.Sleep(100 * time.Millisecond)
	content, err := os.ReadFile(proofFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "ran")
}
