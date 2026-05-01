// Package bus contains cross-process e2e tests for aps bus events.
//
// These tests verify that aps lifecycle events (aps.profile.*, etc.)
// round-trip across process boundaries: the test process hosts a real
// kit/bus hub (via httptest.NewServer + NetworkAdapter.Handler), the
// `aps` binary runs as a child process and connects to that hub via
// APS_BUS_ADDR + APS_BUS_TOKEN, and the test asserts the event lands
// on a subscriber in the test process within a deadline.
//
// Build tag bus_e2e — excluded from default `go test`. Run with:
//
//	go test -tags bus_e2e -count=1 ./tests/e2e/bus/...
//
// Closes T-0162 (tools-showcase-scenarios).
//
//go:build bus_e2e

package bus

import (
	"context"
	"encoding/hex"
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

// apsBinary is the absolute path to the compiled aps binary used by all
// child-process publishers in this package. Built once in TestMain.
var apsBinary string

// TestMain compiles the aps binary once for all tests in this package
// and stashes the path in apsBinary. Mirrors tests/e2e/main_test.go.
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
	binName := "aps-bus-e2e"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	apsBinary = filepath.Join(os.TempDir(), binName)

	// tests/e2e/bus → ../../.. is the module root.
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

// busHub is a running kit/bus hub backed by an in-process Bus, exposed
// over a local httptest WebSocket endpoint with static-token auth. It
// is the receiving side of the cross-process round-trip.
type busHub struct {
	bus     kitbus.Bus
	adapter *kitbus.NetworkAdapter
	server  *httptest.Server
	addr    string // ws://… URL the child process connects to
	token   string // shared secret for StaticTokenAuth
}

// setupBusHub starts an in-process kit/bus hub with token auth. The
// returned addr/token can be passed to a child process via
// APS_BUS_ADDR + APS_BUS_TOKEN. Cleanup is registered on t.
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

	// httptest serves http://; kit/bus.Connect needs ws://.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	h := &busHub{
		bus:     b,
		adapter: adapter,
		server:  srv,
		addr:    wsURL,
		token:   token,
	}

	t.Cleanup(func() {
		_ = adapter.Close()
		srv.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = b.Close(ctx)
	})

	return h
}

// recordedEvent is a snapshot of an event delivered to the test
// subscriber. Payload is the raw decoded value the network adapter
// produced — typically map[string]any after JSON round-trip.
type recordedEvent struct {
	Topic   kitbus.Topic
	Source  string
	Payload any
}

// subscribe attaches a subscriber to the hub's bus on `pattern` and
// returns a thread-safe accessor for the events received. Cleanup
// (unsubscribe) is registered on t.
func (h *busHub) subscribe(t *testing.T, pattern string) (waitFor func(deadline time.Duration, want int) []recordedEvent) {
	t.Helper()

	var (
		mu     sync.Mutex
		events []recordedEvent
	)

	unsub := h.bus.Subscribe(pattern, func(_ context.Context, e kitbus.Event) error {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, recordedEvent{
			Topic:   e.Topic,
			Source:  e.Source,
			Payload: e.Payload,
		})
		return nil
	})
	t.Cleanup(unsub)

	return func(deadline time.Duration, want int) []recordedEvent {
		t.Helper()
		end := time.Now().Add(deadline)
		for {
			mu.Lock()
			n := len(events)
			mu.Unlock()
			if n >= want {
				break
			}
			if time.Now().After(end) {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		mu.Lock()
		defer mu.Unlock()
		out := make([]recordedEvent, len(events))
		copy(out, events)
		return out
	}
}

// runAPSChild executes the aps binary as a child process with HOME
// pointing to a fresh temp dir and APS_BUS_ADDR/APS_BUS_TOKEN set
// to the hub's coordinates. It returns combined output for diagnosis.
//
// We pass a fresh HOME so each test's profile state is isolated — and
// so it does not pollute the developer's real ~/.local/share/aps.
func runAPSChild(t *testing.T, home string, hub *busHub, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(apsBinary, args...)

	overridden := map[string]bool{
		"HOME":          true,
		"USERPROFILE":   true,
		"XDG_DATA_HOME": true,
		"APS_DATA_PATH": true,
		"APS_BUS_ADDR":  true,
		"APS_BUS_TOKEN": true,
		"BUS_TOKEN":     true,
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
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// payloadString extracts a string field from the JSON-decoded payload
// the network adapter delivers (which is map[string]any after the wire
// hop). Empty string when missing or wrong type — caller asserts.
func payloadString(p any, field string) string {
	m, ok := p.(map[string]any)
	if !ok {
		return ""
	}
	v, ok := m[field]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// payloadStringSlice extracts a []string field (decoded as []any) from
// the JSON payload.
func payloadStringSlice(p any, field string) []string {
	m, ok := p.(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := m[field].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// randToken returns a short hex-encoded random token suitable for
// StaticTokenAuth. We do not need crypto-strength here.
func randToken() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return hex.EncodeToString(b)
}

// findEvent returns the first recorded event matching topic, or zero
// value + false. Network forwarding can interleave; callers should
// search rather than index.
func findEvent(events []recordedEvent, topic kitbus.Topic) (recordedEvent, bool) {
	for _, e := range events {
		if e.Topic == topic {
			return e, true
		}
	}
	return recordedEvent{}, false
}
