// Package listen contains cross-process e2e tests for `aps listen`.
//
// Each test:
//  1. starts a kit/bus hub in the test process (httptest WS server),
//  2. spawns `aps listen --profile X --topics Y --exit-after-events Z`
//     as a child process pointing at the hub,
//  3. publishes events on the hub bus,
//  4. asserts the child's stdout contains the expected JSONL records
//     and the child exits cleanly within a deadline.
//
// Build tag listen_e2e — excluded from default `go test`. Run with:
//
//	go test -tags listen_e2e -count=1 ./tests/e2e/listen/...
//
// Closes T-0097 (story 051).
//
//go:build listen_e2e

package listen

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	kitbus "hop.top/kit/go/runtime/bus"
)

var apsBinary string

const propagationDeadline = 5 * time.Second

func TestMain(m *testing.M) {
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "compile aps binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.Remove(apsBinary)
	os.Exit(code)
}

func compileBinary() error {
	binName := "aps-listen-e2e"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	apsBinary = filepath.Join(os.TempDir(), binName)

	rootDir, err := filepath.Abs("../../..")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// syncBuf is a thread-safe bytes.Buffer wrapper. exec.Cmd writes to
// Stdout/Stderr from its internal copy goroutine while tests read; an
// unsynchronised bytes.Buffer would race.
type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *syncBuf) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, s.buf.Len())
	copy(out, s.buf.Bytes())
	return out
}

// busHub mirrors tests/e2e/bus/helpers_test.go: an in-process kit/bus
// instance with NetworkAdapter exposed over an httptest WS endpoint.
type busHub struct {
	bus     kitbus.Bus
	adapter *kitbus.NetworkAdapter
	server  *httptest.Server
	addr    string
	token   string
}

func setupBusHub(t *testing.T) *busHub {
	t.Helper()
	token := randToken()
	b := kitbus.New()
	adapter := kitbus.NewNetworkAdapter(
		b,
		kitbus.WithOriginID("test-hub"),
		kitbus.WithAuth(&kitbus.StaticTokenAuth{Token_: token}),
	)
	srv := httptest.NewServer(adapter.Handler())
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	h := &busHub{bus: b, adapter: adapter, server: srv, addr: wsURL, token: token}
	t.Cleanup(func() {
		_ = adapter.Close()
		srv.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = b.Close(ctx)
	})
	return h
}

// publish wraps hub.bus.Publish with a bounded ctx so flaky tests fail
// loudly instead of hanging. Source is hard-coded ("test") because
// every caller is a test process — exposing it would just invite
// inconsistency.
func (h *busHub) publish(t *testing.T, topic string, payload any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := h.bus.Publish(ctx, kitbus.NewEvent(kitbus.Topic(topic), "test", payload)); err != nil {
		t.Fatalf("hub publish %s: %v", topic, err)
	}
}

// listenChild is a running `aps listen` child wired to a busHub. Use
// startListen to construct.
type listenChild struct {
	cmd       *exec.Cmd
	stdoutBuf *syncBuf
	stderrBuf *syncBuf
	doneCh    chan error
	t         *testing.T
}

// listenLine mirrors the JSONL shape emitted by `aps listen`. Kept
// local so the test does not import internal/cli.
type listenLine struct {
	Topic     string          `json:"topic"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
	Profile   string          `json:"profile"`
}

// startListen creates a profile in a fresh HOME and spawns the listener
// pointed at it. Returns once the child has emitted its "aps listen:"
// startup banner on stderr OR the deadline elapses.
func startListen(t *testing.T, hub *busHub, profileID string, args ...string) *listenChild {
	t.Helper()
	home := t.TempDir()

	// Create profile in child HOME (listen requires LoadProfile success).
	createCmd := exec.Command(apsBinary, "profile", "create", profileID,
		"--display-name", profileID, "--email", profileID+"@test")
	createCmd.Env = childEnv(home, hub)
	if out, err := createCmd.CombinedOutput(); err != nil {
		t.Fatalf("profile create %s failed: %v\n%s", profileID, err, out)
	}

	full := append([]string{"listen", "--profile", profileID}, args...)
	cmd := exec.Command(apsBinary, full...)
	cmd.Env = childEnv(home, hub)

	// Use synchronized buffers for stdout/stderr instead of StdoutPipe.
	// StdoutPipe + a reader goroutine adds a race: cmd.Wait blocks until
	// all stdout copies finish, so when the child exits via signal we
	// can hang waiting for the scanner goroutine to drain. A shared
	// buffer avoids the wait/reader cycle entirely.
	stdoutBuf := &syncBuf{}
	stderrBuf := &syncBuf{}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("start aps listen: %v", err)
	}

	c := &listenChild{
		cmd:       cmd,
		stdoutBuf: stdoutBuf,
		stderrBuf: stderrBuf,
		doneCh:    make(chan error, 1),
		t:         t,
	}
	go func() { c.doneCh <- cmd.Wait() }()
	t.Cleanup(func() { c.Stop() })

	// Wait for startup banner so the WS connect+subscribe is wired
	// before the test publishes.
	deadline := time.Now().Add(propagationDeadline)
	for time.Now().Before(deadline) {
		if strings.Contains(stderrBuf.String(), "aps listen: profile=") {
			// Add a short settle so the inbound peer is fully registered
			// on the hub side (the hub's Handler goroutine spawns
			// readLoop after Accept; without this the first publish
			// can race the WS upgrade).
			time.Sleep(150 * time.Millisecond)
			return c
		}
		select {
		case err := <-c.doneCh:
			t.Fatalf("aps listen exited before banner: %v\nstderr:\n%s", err, stderrBuf.String())
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
	t.Fatalf("aps listen did not start within %s\nstderr:\n%s", propagationDeadline, stderrBuf.String())
	return nil
}

// Lines parses the child's stdout buffer into JSONL listenLine records,
// silently dropping non-JSON noise.
func (c *listenChild) Lines() []listenLine {
	var out []listenLine
	scanner := bufio.NewScanner(bytes.NewReader(c.stdoutBuf.Bytes()))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var line listenLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		out = append(out, line)
	}
	return out
}

// stderr returns the current stderr contents.
func (c *listenChild) stderr() string { return c.stderrBuf.String() }

// WaitForLines polls until len(Lines) >= n or deadline elapses.
func (c *listenChild) WaitForLines(n int, deadline time.Duration) []listenLine {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		ls := c.Lines()
		if len(ls) >= n {
			return ls
		}
		time.Sleep(50 * time.Millisecond)
	}
	return c.Lines()
}

// WaitExit blocks until the child exits or the deadline elapses. Returns
// (true, err) on exit and (false, nil) on timeout. Bool comes first per
// revive's error-return rule.
func (c *listenChild) WaitExit(deadline time.Duration) (bool, error) {
	select {
	case err := <-c.doneCh:
		return true, err
	case <-time.After(deadline):
		return false, nil
	}
}

// Stop sends SIGINT and reaps. Safe to call multiple times.
func (c *listenChild) Stop() {
	if c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Signal(os.Interrupt)
	select {
	case <-c.doneCh:
	case <-time.After(3 * time.Second):
		_ = c.cmd.Process.Kill()
		select {
		case <-c.doneCh:
		case <-time.After(2 * time.Second):
		}
	}
}

func childEnv(home string, hub *busHub) []string {
	overridden := map[string]bool{
		"HOME": true, "USERPROFILE": true,
		"XDG_DATA_HOME": true, "APS_DATA_PATH": true,
		"APS_BUS_ADDR": true, "APS_BUS_TOKEN": true, "BUS_TOKEN": true,
	}
	env := []string{
		"HOME=" + home,
		"USERPROFILE=" + home,
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"APS_BUS_ADDR=" + hub.addr,
		"APS_BUS_TOKEN=" + hub.token,
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if overridden[key] {
			continue
		}
		env = append(env, e)
	}
	return env
}

func randToken() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return hex.EncodeToString(b)
}

func hasTopic(lines []listenLine, topic string) bool {
	for _, l := range lines {
		if l.Topic == topic {
			return true
		}
	}
	return false
}
