# Voice Test Coverage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the three untested areas in `internal/voice` — `BackendManager.Start`, audio I/O loops, and the WebSocket upgrade failure paths — bringing coverage from 70% to ≥90%.

**Architecture:** All tests live in the existing `package voice_test` files alongside existing tests. No new files created unless a test file is missing. Tests use `net/http/httptest`, `github.com/gorilla/websocket`, OS pipes and `net.Pipe()` to exercise real I/O without external processes. `BackendManager.Start` is tested by pointing the backend binary at `/bin/echo` (always present) for the happy path and at a nonexistent path for the error path.

**Tech Stack:** Go `testing`, `testify/assert`, `gorilla/websocket`, `net/http/httptest`, standard `net` package.

---

### Task 1: BackendManager.Start — happy path and error paths

**Files:**
- Modify: `internal/voice/backend_test.go`

The goal is to cover `Start` (currently 0%) and `Stop`'s kill branch (currently 50%).

**Step 1: Write failing test — external URL is a no-op**

Add to `internal/voice/backend_test.go`:

```go
func TestBackendManager_Start_ExternalURL_NoOp(t *testing.T) {
	m := voice.NewBackendManager(voice.GlobalBackendConfig{})
	err := m.Start(&voice.BackendConfig{URL: "ws://remote:8998"})
	assert.NoError(t, err)
	assert.False(t, m.IsRunning()) // no local process started
}
```

Run: `go test ./internal/voice/... -run TestBackendManager_Start_ExternalURL_NoOp -v`
Expected: PASS (this path already returns nil — confirms the no-op branch is reachable and IsRunning stays false).

**Step 2: Write failing test — compatible fallback returns error**

```go
func TestBackendManager_Start_Compatible_ReturnsError(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "auto",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManager(cfg)
	err := m.Start(nil)
	assert.ErrorContains(t, err, "no voice backend binary configured")
}
```

Run: `go test ./internal/voice/... -run TestBackendManager_Start_Compatible_ReturnsError -v`
Expected: PASS (the `compatible` branch already returns this error — confirms the path is exercised).

**Step 3: Write failing test — unknown backend type returns error**

```go
func TestBackendManager_Start_UnknownType_ReturnsError(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "moshi-mlx",
		Backends:       map[string]voice.BackendBinConfig{}, // no entry for moshi-mlx
	}
	m := voice.NewBackendManager(cfg)
	err := m.Start(nil)
	assert.ErrorContains(t, err, `no binary configured for backend type "moshi-mlx"`)
}
```

Run: `go test ./internal/voice/... -run TestBackendManager_Start_UnknownType_ReturnsError -v`
Expected: PASS.

**Step 4: Write failing test — Start launches process and IsRunning becomes true**

```go
func TestBackendManager_Start_LaunchesProcess(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "echo-backend",
		Backends: map[string]voice.BackendBinConfig{
			"echo-backend": {Bin: "/bin/sleep", Args: []string{"10"}},
		},
	}
	m := voice.NewBackendManager(cfg)
	err := m.Start(nil)
	assert.NoError(t, err)
	assert.True(t, m.IsRunning())
	// cleanup
	assert.NoError(t, m.Stop())
	assert.False(t, m.IsRunning())
}
```

Run: `go test ./internal/voice/... -run TestBackendManager_Start_LaunchesProcess -v`
Expected: FAIL with "IsRunning should be true" (process not tracked after start — the test should fail first if Start was 0%). If it already passes, move on.

**Step 5: Write failing test — Stop kills the running process**

The `Stop` happy path (killing a running process) is also uncovered. This is exercised by `TestBackendManager_Start_LaunchesProcess` above via the cleanup. No separate test needed.

**Step 6: Run all backend tests**

```bash
go test ./internal/voice/... -run TestBackendManager -v
```

Expected: All PASS.

**Step 7: Commit**

```bash
git add internal/voice/backend_test.go
git commit -m "test(voice): cover BackendManager.Start launch, external URL no-op, and error paths"
```

---

### Task 2: Web adapter — WebSocket upgrade failure

**Files:**
- Modify: `internal/voice/adapter_web_test.go`

`ServeHTTP` is at 70% — the websocket upgrade failure branch is not exercised.

**Step 1: Write the failing test**

```go
func TestWebAdapter_UpgradeFailure_DoesNotPanic(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	_, _ = adapter.Accept()
	defer adapter.Close()

	// Send a plain HTTP GET to /ws (not a WebSocket upgrade) — upgrade will fail
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)

	// Upgrade fails gracefully — no session emitted, no panic
	// gorilla returns 400 for non-upgrade requests
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

Add import `"net/http"` to the file if not present.

Run: `go test ./internal/voice/... -run TestWebAdapter_UpgradeFailure -v`
Expected: FAIL (code might differ or test might not compile yet due to missing import).

**Step 2: Fix imports and run again until it fails for the right reason**

Ensure the test file imports `"net/http"` and `"net/http/httptest"`. Run again — expected: FAIL with wrong status or panic.

**Step 3: Run to confirm it passes**

The code already handles upgrade failure by returning early. Run:
```bash
go test ./internal/voice/... -run TestWebAdapter -v
```
Expected: All PASS.

**Step 4: Commit**

```bash
git add internal/voice/adapter_web_test.go
git commit -m "test(voice): cover WebSocket upgrade failure path in WebAdapter"
```

---

### Task 3: Twilio adapter — upgrade failure

**Files:**
- Modify: `internal/voice/adapter_twilio_test.go`

Same gap as Task 2 but for TwilioAdapter.

**Step 1: Write the failing test**

```go
func TestTwilioAdapter_UpgradeFailure_DoesNotPanic(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	_, _ = adapter.Accept()
	defer adapter.Close()

	req := httptest.NewRequest(http.MethodGet, "/twilio/media-stream", nil)
	w := httptest.NewRecorder()
	adapter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

Run: `go test ./internal/voice/... -run TestTwilioAdapter_UpgradeFailure -v`
Expected: FAIL (for right reason — wrong status or panic).

**Step 2: Run to confirm it passes**

```bash
go test ./internal/voice/... -run TestTwilioAdapter -v
```
Expected: All PASS.

**Step 3: Commit**

```bash
git add internal/voice/adapter_twilio_test.go
git commit -m "test(voice): cover WebSocket upgrade failure path in TwilioAdapter"
```

---

### Task 4: Audio I/O — TUI adapter read/write loops

**Files:**
- Modify: `internal/voice/adapter_tui_test.go`

`readLoop` is 66.7%, `writeLoop` is 33.3%, `AudioIn`/`AudioOut`/`TextOut` are 0%.

**Step 1: Write the failing test — send audio frame through TUI session**

```go
func TestTUISession_AudioFrameRoundTrip(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "v.sock")
	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)
	sessions, _ := adapter.Accept()
	defer adapter.Close()

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	// send a frame from client → session.AudioIn
	frame := []byte("pcm-frame-data")
	_, err = conn.Write(frame)
	assert.NoError(t, err)

	received := <-sess.AudioIn()
	assert.Equal(t, frame, received)
	sess.Close()
}
```

Run: `go test ./internal/voice/... -run TestTUISession_AudioFrameRoundTrip -v`
Expected: FAIL (AudioIn is not exported directly; may need to verify the channel is accessible).

**Step 2: Write the failing test — session sends audio out to client**

```go
func TestTUISession_AudioOut_WritesToConn(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "v.sock")
	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)
	sessions, _ := adapter.Accept()
	defer adapter.Close()

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("response-audio")
	sess.AudioOut() <- frame

	buf := make([]byte, len(frame))
	_, err = io.ReadFull(conn, buf)
	assert.NoError(t, err)
	assert.Equal(t, frame, buf)
	sess.Close()
}
```

Add import `"io"` to the file.

Run: `go test ./internal/voice/... -run TestTUISession_AudioOut -v`
Expected: FAIL (AudioOut channel not drained to conn, or conn reads nothing).

**Step 3: Confirm both pass**

```bash
go test ./internal/voice/... -run TestTUISession -v
```
Expected: All PASS.

**Step 4: Commit**

```bash
git add internal/voice/adapter_tui_test.go
git commit -m "test(voice): cover TUI session audio in/out frame paths"
```

---

### Task 5: Audio I/O — Web adapter read/write loops

**Files:**
- Modify: `internal/voice/adapter_web_test.go`

`readLoop` is 83.3% (error exit not hit), `writeLoop` is 20% (only entry, never sends text or binary), `AudioIn`/`AudioOut`/`TextOut` are 0%.

**Step 1: Write the failing test — binary frame in**

```go
func TestWebSession_AudioIn_ReceivesFrame(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("audio-in")
	err = conn.WriteMessage(websocket.BinaryMessage, frame)
	assert.NoError(t, err)

	received := <-sess.AudioIn()
	assert.Equal(t, frame, received)
	sess.Close()
}
```

Run: `go test ./internal/voice/... -run TestWebSession_AudioIn -v`
Expected: FAIL (AudioIn channel not fed).

**Step 2: Write the failing test — binary frame out**

```go
func TestWebSession_AudioOut_SendsFrame(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("audio-out")
	sess.AudioOut() <- frame

	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, frame, msg)
	sess.Close()
}
```

Run: `go test ./internal/voice/... -run TestWebSession_AudioOut -v`
Expected: FAIL.

**Step 3: Write the failing test — text out**

```go
func TestWebSession_TextOut_SendsTextFrame(t *testing.T) {
	adapter := voice.NewWebAdapter("profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	sess.TextOut() <- "hello agent"

	msgType, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, msgType)
	assert.Equal(t, "hello agent", string(msg))
	sess.Close()
}
```

Run: `go test ./internal/voice/... -run TestWebSession_TextOut -v`
Expected: FAIL.

**Step 4: Run all web adapter tests**

```bash
go test ./internal/voice/... -run TestWeb -v
```
Expected: All PASS.

**Step 5: Commit**

```bash
git add internal/voice/adapter_web_test.go
git commit -m "test(voice): cover WebSession audio in/out and text out paths"
```

---

### Task 6: Audio I/O — Twilio adapter write loop

**Files:**
- Modify: `internal/voice/adapter_twilio_test.go`

`writeLoop` is 33.3% — only the loop entry is hit. The binary write path and error exit are not exercised.

**Step 1: Write the failing test — audio in from Twilio**

```go
func TestTwilioSession_AudioIn_ReceivesFrame(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/twilio/media-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("mulaw-audio")
	err = conn.WriteMessage(websocket.BinaryMessage, frame)
	assert.NoError(t, err)

	received := <-sess.AudioIn()
	assert.Equal(t, frame, received)
	sess.Close()
}
```

Run: `go test ./internal/voice/... -run TestTwilioSession_AudioIn -v`
Expected: FAIL.

**Step 2: Write the failing test — audio out to Twilio**

```go
func TestTwilioSession_AudioOut_SendsFrame(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	sessions, _ := adapter.Accept()
	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/twilio/media-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions

	frame := []byte("response-mulaw")
	sess.AudioOut() <- frame

	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, frame, msg)
	sess.Close()
}
```

Run: `go test ./internal/voice/... -run TestTwilioSession_AudioOut -v`
Expected: FAIL.

**Step 3: Run all Twilio tests**

```bash
go test ./internal/voice/... -run TestTwilio -v
```
Expected: All PASS.

**Step 4: Commit**

```bash
git add internal/voice/adapter_twilio_test.go
git commit -m "test(voice): cover Twilio session audio in/out paths"
```

---

### Task 7: Verify final coverage

**Step 1: Run full voice suite with coverage**

```bash
go test ./internal/voice/... -cover -coverprofile=/tmp/voice.out && go tool cover -func=/tmp/voice.out
```

Expected: Total coverage ≥ 90%. All previously-0% functions now covered.

**Step 2: Run full suite to confirm no regressions**

```bash
go test ./... 2>&1 | grep -E "FAIL|ok"
```

Expected: All packages `ok`.
