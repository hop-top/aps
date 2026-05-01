package e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startWebhookServer spawns `aps webhook server`, parses the chosen
// listen port from stderr, and returns the base URL + a teardown fn.
// Used by gap tests T-0164/0165/0166.
func startWebhookServer(t *testing.T, cmd *exec.Cmd, stderr *bytes.Buffer) string {
	t.Helper()
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		logs := stderr.String()
		if idx := strings.Index(logs, "addr=127.0.0.1:"); idx != -1 {
			rest := logs[idx+len("addr=127.0.0.1:"):]
			if end := strings.IndexAny(rest, " \t\n\r"); end != -1 {
				return fmt.Sprintf("http://127.0.0.1:%s/webhook", strings.TrimSpace(rest[:end]))
			}
		}
	}
	t.Fatalf("webhook server did not bind a port; stderr=%s", stderr.String())
	return ""
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// T-0164: closes US-0009 #3 — unmapped event returns 400.
// Existing TestWebhookServer covers MISSING X-APS-Event header (400);
// this exercises a present-but-unmapped event name.
func TestWebhook_UnmappedEventReturns400(t *testing.T) {
	t.Parallel()
	home := t.TempDir()

	cmd := prepareAPS(t, home, nil, "webhook", "server",
		"--addr", "127.0.0.1:0",
		"--event-map", "known.event=anyprofile:hook",
		"--secret", "s3cr3t")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	url := startWebhookServer(t, cmd, &stderr)

	body := []byte(`{"x":1}`)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("X-APS-Event", "unknown.event")
	req.Header.Set("X-APS-Signature", sign("s3cr3t", body))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	assert.Contains(t, buf.String(), "unknown.event",
		"error body should name the unmapped event")
}

// T-0165: closes US-0039 #4 — `webhook server --profile` auto-enables
// AND the live request is handled (200 + persisted enabled state).
// Existing TestWebhookServer_AutoEnable kills before sending a request.
func TestWebhook_ServerAutoEnableOnLiveRequest(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "auto-en-live"

	_, _, err := runAPS(t, home, "profile", "create", profileID)
	require.NoError(t, err)

	// Stage an action so the dispatched mapping has something to run.
	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	require.NoError(t, os.MkdirAll(actionsDir, 0755))
	proof := filepath.Join(home, "auto-proof.txt")
	require.NoError(t, os.WriteFile(filepath.Join(actionsDir, "hook.sh"),
		[]byte(fmt.Sprintf("#!/bin/sh\necho ran > %s", proof)), 0755))

	cmd := prepareAPS(t, home, nil, "webhook", "server",
		"--profile", profileID,
		"--addr", "127.0.0.1:0",
		"--event-map", "live.event="+profileID+":hook",
		"--secret", "topsecret")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	url := startWebhookServer(t, cmd, &stderr)

	assert.Contains(t, stderr.String(), "auto-enabling",
		"server should announce auto-enable on stderr")

	body := []byte(`{"live":true}`)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("X-APS-Event", "live.event")
	req.Header.Set("X-APS-Signature", sign("topsecret", body))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	time.Sleep(150 * time.Millisecond)
	content, err := os.ReadFile(proof)
	require.NoError(t, err, "action should have run after auto-enable")
	assert.Contains(t, string(content), "ran")

	// Persisted enabled state survives the live request.
	yaml, err := os.ReadFile(filepath.Join(home, ".local", "share", "aps",
		"profiles", profileID, "profile.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(yaml), "- webhooks")
}

// T-0166: per architecture.md — bus token missing → bus disabled with
// stderr warning; webhook still serves; no crash. Uses a real profile +
// action to isolate the bus-token-missing condition from action-exec
// failure (would otherwise mask as 500).
func TestWebhook_BusTokenMissingGracefulFallback(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	profileID := "no-bus-tok"

	// Stage profile + action up front (these stages also init the bus
	// with no token; capture warnings here too).
	_, createErr, err := runAPSWithEnv(t, home,
		map[string]string{"APS_BUS_TOKEN": "", "BUS_TOKEN": ""},
		"profile", "create", profileID)
	require.NoError(t, err)
	assert.Contains(t, createErr, "bus disabled",
		"profile create with no bus token must warn on stderr")

	actionsDir := filepath.Join(home, ".local", "share", "aps", "profiles", profileID, "actions")
	require.NoError(t, os.MkdirAll(actionsDir, 0755))
	proof := filepath.Join(home, "no-bus-proof.txt")
	require.NoError(t, os.WriteFile(filepath.Join(actionsDir, "hook.sh"),
		[]byte(fmt.Sprintf("#!/bin/sh\necho ran > %s", proof)), 0755))

	cmd := prepareAPS(t, home,
		map[string]string{"APS_BUS_TOKEN": "", "BUS_TOKEN": ""},
		"webhook", "server",
		"--addr", "127.0.0.1:0",
		"--event-map", "bus.event="+profileID+":hook",
		"--secret", "k")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	url := startWebhookServer(t, cmd, &stderr)

	assert.Contains(t, stderr.String(), "bus disabled",
		"server stderr should warn about disabled bus when token missing")

	body := []byte(`{}`)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("X-APS-Event", "bus.event")
	req.Header.Set("X-APS-Signature", sign("k", body))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "server must keep serving without bus token")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"webhook must still execute action when bus is disabled")

	time.Sleep(150 * time.Millisecond)
	content, err := os.ReadFile(proof)
	require.NoError(t, err, "action should still run when bus is disabled")
	assert.Contains(t, string(content), "ran")
}
