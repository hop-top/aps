# Voice Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `voice` subsystem to APS that manages speech-to-speech backend services (PersonaPlex, Moshi), routes voice sessions across web/TUI/messenger/telephony channels, and maps APS profile personas to backend voice+prompt configs.

**Architecture:** A new `internal/voice/` package owns backend lifecycle, session management, and the transcript→action pipeline. Channel adapters (web, TUI socket, messenger, Twilio) implement a single `ChannelAdapter` interface so the orchestrator stays channel-agnostic. The `Profile` struct gains a `Voice` field; APS auto-generates the backend text prompt from existing `Persona` fields when no template is set.

**Tech Stack:** Go 1.25, Cobra (CLI), `gorilla/websocket` (backend WebSocket), `gopkg.in/yaml.v3` (config), `github.com/stretchr/testify` (tests). Module: `hop.top/aps`.

**Design doc:** `docs/plans/2026-03-16-voice-integration-design.md`

---

## Task 1: Voice config types & Profile extension

**Files:**
- Create: `internal/voice/config.go`
- Modify: `internal/core/profile.go`
- Test: `internal/voice/config_test.go`

**Step 1: Write the failing test**

```go
// internal/voice/config_test.go
package voice_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestBackendConfig_Defaults(t *testing.T) {
	cfg := voice.BackendConfig{}
	assert.Equal(t, "", cfg.URL)
	assert.Equal(t, "", cfg.Type)
}

func TestVoiceConfig_IsEnabled(t *testing.T) {
	cfg := voice.Config{Enabled: true}
	assert.True(t, cfg.Enabled)
}

func TestChannelsConfig_Fields(t *testing.T) {
	cfg := voice.ChannelsConfig{
		Web: true,
		TUI: true,
	}
	assert.True(t, cfg.Web)
	assert.True(t, cfg.TUI)
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/jadb/.w/ideacrafterslabs/aps/hops/main
go test ./internal/voice/... 2>&1
```

Expected: compile error — package does not exist yet.

**Step 3: Implement config types**

```go
// internal/voice/config.go
package voice

// BackendConfig selects and locates the speech-to-speech backend.
// Type values: auto | personaplex-cuda | personaplex-mlx | moshi | moshi-mlx | compatible
// URL empty means APS manages the backend process; set to delegate to an external instance.
type BackendConfig struct {
	URL  string `yaml:"url,omitempty"`
	Type string `yaml:"type,omitempty"` // default: "auto"
}

// TelegramChannelConfig holds Telegram bot credentials for a profile's voice channel.
type TelegramChannelConfig struct {
	Enabled        bool   `yaml:"enabled,omitempty"`
	BotTokenSecret string `yaml:"bot_token_secret,omitempty"`
}

// TwilioChannelConfig holds Twilio credentials and phone number for inbound call routing.
type TwilioChannelConfig struct {
	Enabled          bool   `yaml:"enabled,omitempty"`
	PhoneNumber      string `yaml:"phone_number,omitempty"`
	AccountSIDSecret string `yaml:"account_sid_secret,omitempty"`
	AuthTokenSecret  string `yaml:"auth_token_secret,omitempty"`
}

// ChannelsConfig declares which channels this profile's voice is active on.
type ChannelsConfig struct {
	Web      bool                   `yaml:"web,omitempty"`
	TUI      bool                   `yaml:"tui,omitempty"`
	Telegram *TelegramChannelConfig `yaml:"telegram,omitempty"`
	Twilio   *TwilioChannelConfig   `yaml:"twilio,omitempty"`
}

// Config is the voice block inside a Profile.
// All fields are optional; APS provides sensible defaults.
type Config struct {
	Enabled        bool           `yaml:"enabled,omitempty"`
	Backend        BackendConfig  `yaml:"backend,omitempty"`
	VoiceID        string         `yaml:"voice_id,omitempty"` // e.g. "NATF0"
	PromptTemplate string         `yaml:"prompt_template,omitempty"`
	Channels       ChannelsConfig `yaml:"channels,omitempty"`
}
```

**Step 4: Add `Voice` field to `Profile`**

In `internal/core/profile.go`, add the field after `Trust`:

```go
Voice *voice.Config `yaml:"voice,omitempty"`
```

Add import at top of file:
```go
"hop.top/aps/internal/voice"
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/voice/... ./internal/core/... -v 2>&1 | tail -20
```

Expected: all PASS, no compile errors.

**Step 6: Commit**

```bash
git add internal/voice/config.go internal/voice/config_test.go internal/core/profile.go
git commit -m "feat(voice): add voice config types and Profile.Voice field"
```

---

## Task 2: Persona prompt auto-generation

**Files:**
- Create: `internal/voice/prompt.go`
- Test: `internal/voice/prompt_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/prompt_test.go
package voice_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/core"
	"hop.top/aps/internal/voice"
)

func TestPromptGenerator_UsesTemplateWhenSet(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{
		DisplayName: "Alice",
		Voice:       &voice.Config{PromptTemplate: "You are a pirate."},
	}
	assert.Equal(t, "You are a pirate.", g.Generate(p))
}

func TestPromptGenerator_AutoGeneratesFromPersona(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{
		DisplayName: "Support Bot",
		Persona:     core.Persona{Tone: "friendly", Style: "concise", Risk: "low"},
		Voice:       &voice.Config{},
	}
	result := g.Generate(p)
	assert.Contains(t, result, "Support Bot")
	assert.Contains(t, result, "warm and approachable")
	assert.Contains(t, result, "brief and to the point")
	assert.Contains(t, result, "Never speculate")
}

func TestPromptGenerator_NilVoiceConfig(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{DisplayName: "Bot", Persona: core.Persona{Tone: "casual"}}
	result := g.Generate(p)
	assert.Contains(t, result, "Bot")
	assert.Contains(t, result, "relaxed and conversational")
}

func TestPromptGenerator_UnknownToneFallsBack(t *testing.T) {
	g := voice.NewPromptGenerator()
	p := &core.Profile{DisplayName: "Bot", Persona: core.Persona{Tone: "weird"}}
	result := g.Generate(p)
	assert.Contains(t, result, "Bot")
	// should not panic, unknown tone produces empty string gracefully
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestPromptGenerator -v 2>&1
```

Expected: compile error — `NewPromptGenerator` not defined.

**Step 3: Implement prompt generator**

```go
// internal/voice/prompt.go
package voice

import (
	"fmt"
	"hop.top/aps/internal/core"
)

var toneMap = map[string]string{
	"friendly":     "warm and approachable",
	"professional": "formal and precise",
	"casual":       "relaxed and conversational",
}

var styleMap = map[string]string{
	"concise":  "brief and to the point",
	"detailed": "thorough and comprehensive",
	"casual":   "conversational and informal",
}

var riskMap = map[string]string{
	"low":    "Never speculate. If unsure, say so.",
	"medium": "Use best judgement, flag uncertainty.",
	"high":   "Act decisively with available information.",
}

// PromptGenerator builds a PersonaPlex/Moshi text prompt from an APS Profile.
type PromptGenerator struct{}

func NewPromptGenerator() *PromptGenerator { return &PromptGenerator{} }

// Generate returns the text prompt to inject into the voice backend.
// If profile.Voice.PromptTemplate is set, it takes precedence over auto-generation.
func (g *PromptGenerator) Generate(p *core.Profile) string {
	if p.Voice != nil && p.Voice.PromptTemplate != "" {
		return p.Voice.PromptTemplate
	}
	tone := toneMap[p.Persona.Tone]
	style := styleMap[p.Persona.Style]
	risk := riskMap[p.Persona.Risk]
	return fmt.Sprintf("You are %s. Your communication style is %s and %s. %s",
		p.DisplayName, tone, style, risk)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestPromptGenerator -v 2>&1
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/voice/prompt.go internal/voice/prompt_test.go
git commit -m "feat(voice): persona prompt auto-generation from Profile fields"
```

---

## Task 3: Backend manager (process lifecycle)

**Files:**
- Create: `internal/voice/backend.go`
- Test: `internal/voice/backend_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/backend_test.go
package voice_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestBackendManager_ResolveType_Auto_DefaultsToCompatible(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "auto",
		Backends:       map[string]voice.BackendBinConfig{},
	}
	m := voice.NewBackendManager(cfg)
	resolved := m.ResolveType()
	assert.Equal(t, "compatible", resolved)
}

func TestBackendManager_ResolveType_Explicit(t *testing.T) {
	cfg := voice.GlobalBackendConfig{
		DefaultBackend: "moshi-mlx",
		Backends: map[string]voice.BackendBinConfig{
			"moshi-mlx": {Bin: "/usr/local/bin/moshi-mlx", Args: []string{"--port", "8998"}},
		},
	}
	m := voice.NewBackendManager(cfg)
	assert.Equal(t, "moshi-mlx", m.ResolveType())
}

func TestBackendManager_URL_ExplicitOverride(t *testing.T) {
	cfg := voice.GlobalBackendConfig{}
	m := voice.NewBackendManager(cfg)
	url := m.ResolveURL(&voice.BackendConfig{URL: "ws://remote:8998"})
	assert.Equal(t, "ws://remote:8998", url)
}

func TestBackendManager_URL_ManagedDefault(t *testing.T) {
	cfg := voice.GlobalBackendConfig{}
	m := voice.NewBackendManager(cfg)
	url := m.ResolveURL(&voice.BackendConfig{})
	assert.Equal(t, "ws://localhost:8998", url)
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestBackendManager -v 2>&1
```

Expected: compile error.

**Step 3: Implement backend manager**

```go
// internal/voice/backend.go
package voice

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"github.com/charmbracelet/log"
)

// BackendBinConfig holds the path and args for a managed backend binary.
type BackendBinConfig struct {
	Bin  string   `yaml:"bin"`
	Args []string `yaml:"args,omitempty"`
}

// GlobalBackendConfig is the voice section of ~/.config/aps/config.yaml.
type GlobalBackendConfig struct {
	DefaultBackend string                      `yaml:"default_backend,omitempty"` // default: "auto"
	Backends       map[string]BackendBinConfig `yaml:"backends,omitempty"`
}

// BackendManager manages the voice backend process lifecycle.
type BackendManager struct {
	cfg  GlobalBackendConfig
	mu   sync.Mutex
	proc *exec.Cmd
}

func NewBackendManager(cfg GlobalBackendConfig) *BackendManager {
	return &BackendManager{cfg: cfg}
}

// autoOrder returns the preferred backend types for auto-detection.
func autoOrder() []string {
	if runtime.GOOS == "darwin" {
		return []string{"personaplex-mlx", "moshi-mlx", "personaplex-cuda", "moshi"}
	}
	return []string{"personaplex-cuda", "moshi", "personaplex-mlx", "moshi-mlx"}
}

// ResolveType returns the effective backend type to use.
// For "auto", walks the platform-preferred order and picks the first configured binary.
// Falls back to "compatible" if nothing is configured (caller supplies URL).
func (m *BackendManager) ResolveType() string {
	t := m.cfg.DefaultBackend
	if t == "" || t == "auto" {
		for _, candidate := range autoOrder() {
			if _, ok := m.cfg.Backends[candidate]; ok {
				return candidate
			}
		}
		return "compatible"
	}
	return t
}

// ResolveURL returns the WebSocket URL for the backend.
// If cfg.URL is set, it is used directly (external instance).
// Otherwise defaults to the locally managed instance.
func (m *BackendManager) ResolveURL(cfg *BackendConfig) string {
	if cfg != nil && cfg.URL != "" {
		return cfg.URL
	}
	return "ws://localhost:8998"
}

// Start launches the managed backend process.
// No-op if URL is set on the profile (external instance).
func (m *BackendManager) Start(profileCfg *BackendConfig) error {
	if profileCfg != nil && profileCfg.URL != "" {
		log.Info("voice backend: using external instance", "url", profileCfg.URL)
		return nil
	}
	t := m.cfg.DefaultBackend
	if profileCfg != nil && profileCfg.Type != "" {
		t = profileCfg.Type
	}
	if t == "" || t == "auto" {
		t = m.ResolveType()
	}
	if t == "compatible" {
		return fmt.Errorf("no voice backend binary configured; set voice.backends in config.yaml or provide backend.url in profile")
	}
	binCfg, ok := m.cfg.Backends[t]
	if !ok {
		return fmt.Errorf("no binary configured for backend type %q", t)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.proc = exec.Command(binCfg.Bin, binCfg.Args...) //nolint:gosec
	if err := m.proc.Start(); err != nil {
		return fmt.Errorf("start voice backend %q: %w", t, err)
	}
	log.Info("voice backend started", "type", t, "pid", m.proc.Process.Pid)
	return nil
}

// Stop terminates the managed backend process if running.
func (m *BackendManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.proc == nil || m.proc.Process == nil {
		return nil
	}
	if err := m.proc.Process.Kill(); err != nil {
		return fmt.Errorf("stop voice backend: %w", err)
	}
	m.proc = nil
	return nil
}

// IsRunning reports whether the managed process is alive.
func (m *BackendManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.proc != nil && m.proc.Process != nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestBackendManager -v 2>&1
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/voice/backend.go internal/voice/backend_test.go
git commit -m "feat(voice): backend manager with process lifecycle and auto-detection"
```

---

## Task 4: Channel adapter interfaces & session types

**Files:**
- Create: `internal/voice/channel.go`
- Test: `internal/voice/channel_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/channel_test.go
package voice_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestSessionMeta_Fields(t *testing.T) {
	meta := voice.SessionMeta{
		ProfileID:   "my-profile",
		ChannelType: "web",
		CallerID:    "user-123",
	}
	assert.Equal(t, "my-profile", meta.ProfileID)
	assert.Equal(t, "web", meta.ChannelType)
	assert.Equal(t, "user-123", meta.CallerID)
}

// mockChannelSession implements ChannelSession for testing.
type mockChannelSession struct {
	audioIn  chan []byte
	audioOut chan []byte
	textOut  chan string
	meta     voice.SessionMeta
	closed   bool
}

func (m *mockChannelSession) AudioIn() <-chan []byte  { return m.audioIn }
func (m *mockChannelSession) AudioOut() chan<- []byte { return m.audioOut }
func (m *mockChannelSession) TextOut() chan<- string  { return m.textOut }
func (m *mockChannelSession) Meta() voice.SessionMeta { return m.meta }
func (m *mockChannelSession) Close() error           { m.closed = true; return nil }

func TestMockChannelSession_ImplementsInterface(t *testing.T) {
	var _ voice.ChannelSession = &mockChannelSession{}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestSessionMeta -v 2>&1
```

Expected: compile error.

**Step 3: Implement channel interfaces**

```go
// internal/voice/channel.go
package voice

// SessionMeta carries metadata about an incoming channel connection.
type SessionMeta struct {
	ProfileID   string // hint from channel (may be empty for intent-routed sessions)
	ChannelType string // "web" | "tui" | "telegram" | "twilio"
	CallerID    string // platform user ID, phone number, etc.
}

// ChannelSession is the uniform interface all channel adapters present to the orchestrator.
// AudioIn delivers raw PCM frames from the caller.
// AudioOut accepts raw PCM frames to send to the caller.
// TextOut accepts text responses for text-only channels (messenger).
type ChannelSession interface {
	AudioIn() <-chan []byte
	AudioOut() chan<- []byte
	TextOut() chan<- string
	Meta() SessionMeta
	Close() error
}

// ChannelAdapter listens on a channel and emits ChannelSessions.
type ChannelAdapter interface {
	Accept() (<-chan ChannelSession, error)
	Close() error
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestSessionMeta -v 2>&1
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/voice/channel.go internal/voice/channel_test.go
git commit -m "feat(voice): ChannelAdapter and ChannelSession interfaces"
```

---

## Task 5: Session manager

**Files:**
- Create: `internal/voice/session.go`
- Test: `internal/voice/session_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/session_test.go
package voice_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestSessionManager_CreateAndGet(t *testing.T) {
	sm := voice.NewSessionManager()
	sess := sm.Create("profile-1", "web")
	assert.NotEmpty(t, sess.ID)
	assert.Equal(t, "profile-1", sess.ProfileID)
	assert.Equal(t, "web", sess.ChannelType)
	assert.Equal(t, voice.SessionStateActive, sess.State)

	got, err := sm.Get(sess.ID)
	assert.NoError(t, err)
	assert.Equal(t, sess.ID, got.ID)
}

func TestSessionManager_List(t *testing.T) {
	sm := voice.NewSessionManager()
	sm.Create("p1", "web")
	sm.Create("p2", "tui")
	sessions := sm.List()
	assert.Len(t, sessions, 2)
}

func TestSessionManager_Close(t *testing.T) {
	sm := voice.NewSessionManager()
	sess := sm.Create("p1", "web")
	err := sm.Close(sess.ID)
	assert.NoError(t, err)
	got, err := sm.Get(sess.ID)
	assert.NoError(t, err)
	assert.Equal(t, voice.SessionStateClosed, got.State)
}

func TestSessionManager_GetUnknown(t *testing.T) {
	sm := voice.NewSessionManager()
	_, err := sm.Get("does-not-exist")
	assert.Error(t, err)
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestSessionManager -v 2>&1
```

Expected: compile error.

**Step 3: Implement session manager**

```go
// internal/voice/session.go
package voice

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionState string

const (
	SessionStateActive SessionState = "active"
	SessionStateClosed SessionState = "closed"
)

// Session tracks one active voice session.
type Session struct {
	ID          string
	ProfileID   string
	ChannelType string
	State       SessionState
	CreatedAt   time.Time
}

// SessionManager tracks all active voice sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session)}
}

// Create registers a new active session and returns it.
func (sm *SessionManager) Create(profileID, channelType string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s := &Session{
		ID:          uuid.New().String(),
		ProfileID:   profileID,
		ChannelType: channelType,
		State:       SessionStateActive,
		CreatedAt:   time.Now(),
	}
	sm.sessions[s.ID] = s
	return s
}

// Get returns a session by ID.
func (sm *SessionManager) Get(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("voice session %q not found", id)
	}
	return s, nil
}

// List returns all sessions.
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		out = append(out, s)
	}
	return out
}

// Close marks a session as closed.
func (sm *SessionManager) Close(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("voice session %q not found", id)
	}
	s.State = SessionStateClosed
	return nil
}

// SwitchProfile updates the profile for an active session (mid-session switch).
func (sm *SessionManager) SwitchProfile(id, newProfileID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.sessions[id]
	if !ok {
		return fmt.Errorf("voice session %q not found", id)
	}
	s.ProfileID = newProfileID
	return nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestSessionManager -v 2>&1
```

Expected: all PASS.

**Step 5: Commit**

```bash
git add internal/voice/session.go internal/voice/session_test.go
git commit -m "feat(voice): session manager with create/list/close/switch"
```

---

## Task 6: Web UI channel adapter

**Files:**
- Create: `internal/voice/adapter_web.go`
- Test: `internal/voice/adapter_web_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/adapter_web_test.go
package voice_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestWebAdapter_ServeHTTP_Upgrade(t *testing.T) {
	adapter := voice.NewWebAdapter(":0", "profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)

	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	select {
	case sess := <-sessions:
		assert.Equal(t, "web", sess.Meta().ChannelType)
		sess.Close()
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestWebAdapter -v 2>&1
```

Expected: compile error.

**Step 3: Implement web adapter**

```go
// internal/voice/adapter_web.go
package voice

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebAdapter serves the PersonaPlex React client and accepts WebSocket voice sessions.
type WebAdapter struct {
	addr      string
	profileID string
	sessions  chan ChannelSession
	done      chan struct{}
}

func NewWebAdapter(addr, profileID string) *WebAdapter {
	return &WebAdapter{
		addr:      addr,
		profileID: profileID,
		sessions:  make(chan ChannelSession, 8),
		done:      make(chan struct{}),
	}
}

// Accept returns the channel of incoming sessions. Call before serving.
func (a *WebAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

// ServeHTTP handles WebSocket upgrade at /ws; all other paths return 404.
// Attach this as the HTTP handler to serve voice sessions.
func (a *WebAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ws" {
		http.NotFound(w, r)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("web adapter: websocket upgrade failed", "err", err)
		return
	}
	sess := newWebSession(conn, a.profileID)
	select {
	case a.sessions <- sess:
	case <-a.done:
		conn.Close()
	}
}

// Close shuts down the adapter.
func (a *WebAdapter) Close() error {
	close(a.done)
	return nil
}

// webSession wraps a WebSocket connection as a ChannelSession.
type webSession struct {
	conn      *websocket.Conn
	profileID string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func newWebSession(conn *websocket.Conn, profileID string) *webSession {
	s := &webSession{
		conn:      conn,
		profileID: profileID,
		audioIn:   make(chan []byte, 32),
		audioOut:  make(chan []byte, 32),
		textOut:   make(chan string, 8),
	}
	go s.readLoop()
	go s.writeLoop()
	return s
}

func (s *webSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *webSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *webSession) TextOut() chan<- string  { return s.textOut }
func (s *webSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: "web"}
}
func (s *webSession) Close() error { return s.conn.Close() }

func (s *webSession) readLoop() {
	defer close(s.audioIn)
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		s.audioIn <- msg
	}
}

func (s *webSession) writeLoop() {
	for {
		select {
		case frame, ok := <-s.audioOut:
			if !ok {
				return
			}
			if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				return
			}
		case text, ok := <-s.textOut:
			if !ok {
				return
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
				return
			}
		}
	}
}
```

Check gorilla/websocket is in go.mod; if not:

```bash
go get github.com/gorilla/websocket
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestWebAdapter -v 2>&1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/voice/adapter_web.go internal/voice/adapter_web_test.go go.mod go.sum
git commit -m "feat(voice): web channel adapter with WebSocket upgrade"
```

---

## Task 7: TUI socket adapter (APS side)

The Hex TUI is a separate binary. APS exposes a Unix domain socket; the TUI connects to it.

**Files:**
- Create: `internal/voice/adapter_tui.go`
- Test: `internal/voice/adapter_tui_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/adapter_tui_test.go
package voice_test

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestTUIAdapter_AcceptConnection(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "voice.sock")
	adapter, err := voice.NewTUIAdapter(socketPath, "profile-1")
	assert.NoError(t, err)

	sessions, err := adapter.Accept()
	assert.NoError(t, err)
	defer adapter.Close()

	// simulate TUI connecting
	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions
	assert.Equal(t, "tui", sess.Meta().ChannelType)
	sess.Close()
	_ = os.Remove(socketPath)
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestTUIAdapter -v 2>&1
```

Expected: compile error.

**Step 3: Implement TUI adapter**

```go
// internal/voice/adapter_tui.go
package voice

import (
	"fmt"
	"net"
	"os"

	"github.com/charmbracelet/log"
)

// TUIAdapter listens on a Unix domain socket for Hex TUI connections.
type TUIAdapter struct {
	socketPath string
	profileID  string
	listener   net.Listener
	sessions   chan ChannelSession
}

func NewTUIAdapter(socketPath, profileID string) (*TUIAdapter, error) {
	_ = os.Remove(socketPath) // clean up stale socket
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("tui adapter listen %s: %w", socketPath, err)
	}
	a := &TUIAdapter{
		socketPath: socketPath,
		profileID:  profileID,
		listener:   l,
		sessions:   make(chan ChannelSession, 8),
	}
	go a.acceptLoop()
	return a, nil
}

func (a *TUIAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

func (a *TUIAdapter) Close() error {
	return a.listener.Close()
}

func (a *TUIAdapter) acceptLoop() {
	for {
		conn, err := a.listener.Accept()
		if err != nil {
			return // listener closed
		}
		log.Info("tui adapter: new connection")
		a.sessions <- newTUISession(conn, a.profileID)
	}
}

// tuiSession wraps a Unix socket connection as a ChannelSession.
// Protocol: length-prefixed binary frames (4-byte big-endian length + payload).
type tuiSession struct {
	conn      net.Conn
	profileID string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func newTUISession(conn net.Conn, profileID string) *tuiSession {
	s := &tuiSession{
		conn:      conn,
		profileID: profileID,
		audioIn:   make(chan []byte, 32),
		audioOut:  make(chan []byte, 32),
		textOut:   make(chan string, 8),
	}
	go s.readLoop()
	go s.writeLoop()
	return s
}

func (s *tuiSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *tuiSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *tuiSession) TextOut() chan<- string  { return s.textOut }
func (s *tuiSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: "tui"}
}
func (s *tuiSession) Close() error { return s.conn.Close() }

func (s *tuiSession) readLoop() {
	defer close(s.audioIn)
	buf := make([]byte, 4096)
	for {
		n, err := s.conn.Read(buf)
		if err != nil {
			return
		}
		frame := make([]byte, n)
		copy(frame, buf[:n])
		s.audioIn <- frame
	}
}

func (s *tuiSession) writeLoop() {
	for frame := range s.audioOut {
		if _, err := s.conn.Write(frame); err != nil {
			return
		}
	}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestTUIAdapter -v 2>&1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/voice/adapter_tui.go internal/voice/adapter_tui_test.go
git commit -m "feat(voice): TUI Unix socket channel adapter"
```

---

## Task 8: Messenger channel adapter (Telegram/WhatsApp)

Extends APS's existing messenger layer. Voice messages arrive as audio `Attachment`s; responses go back as audio or text.

**Files:**
- Create: `internal/voice/adapter_messenger.go`
- Test: `internal/voice/adapter_messenger_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/adapter_messenger_test.go
package voice_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/core/messenger"
	"hop.top/aps/internal/voice"
)

func TestMessengerAdapter_RouteVoiceMessage(t *testing.T) {
	adapter := voice.NewMessengerAdapter("telegram", "profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)
	defer adapter.Close()

	msg := &messenger.NormalizedMessage{
		ID:       "msg-1",
		Platform: "telegram",
		Sender:   messenger.Sender{ID: "user-1", Name: "Alice"},
		Channel:  messenger.Channel{ID: "chan-1"},
		Attachments: []messenger.Attachment{
			{Type: "audio", URL: "https://example.com/voice.ogg", MimeType: "audio/ogg"},
		},
	}

	err = adapter.Deliver(msg)
	assert.NoError(t, err)

	sess := <-sessions
	assert.Equal(t, "telegram", sess.Meta().ChannelType)
	assert.Equal(t, "user-1", sess.Meta().CallerID)
	sess.Close()
}

func TestMessengerAdapter_IgnoresNonAudioMessages(t *testing.T) {
	adapter := voice.NewMessengerAdapter("telegram", "profile-1")
	_, err := adapter.Accept()
	assert.NoError(t, err)
	defer adapter.Close()

	msg := &messenger.NormalizedMessage{
		ID: "msg-2", Platform: "telegram",
		Sender: messenger.Sender{ID: "u1"}, Channel: messenger.Channel{ID: "c1"},
		Text: "just text, no audio",
	}
	err = adapter.Deliver(msg)
	assert.NoError(t, err) // should not error, just ignored
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestMessengerAdapter -v 2>&1
```

Expected: compile error.

**Step 3: Implement messenger adapter**

```go
// internal/voice/adapter_messenger.go
package voice

import (
	"github.com/charmbracelet/log"
	"hop.top/aps/internal/core/messenger"
)

// MessengerAdapter bridges APS's messenger layer into ChannelSessions.
// Voice messages (audio attachments) open a new session; text-only messages are ignored.
type MessengerAdapter struct {
	platform  string
	profileID string
	sessions  chan ChannelSession
	done      chan struct{}
}

func NewMessengerAdapter(platform, profileID string) *MessengerAdapter {
	return &MessengerAdapter{
		platform:  platform,
		profileID: profileID,
		sessions:  make(chan ChannelSession, 8),
		done:      make(chan struct{}),
	}
}

func (a *MessengerAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

func (a *MessengerAdapter) Close() error {
	close(a.done)
	return nil
}

// Deliver routes an incoming messenger message into the voice pipeline if it contains audio.
// Call this from APS's messenger event handler.
func (a *MessengerAdapter) Deliver(msg *messenger.NormalizedMessage) error {
	audioURL := ""
	for _, att := range msg.Attachments {
		if att.Type == "audio" {
			audioURL = att.URL
			break
		}
	}
	if audioURL == "" {
		return nil // not a voice message
	}
	log.Info("messenger adapter: voice message received", "platform", a.platform, "caller", msg.Sender.ID)
	sess := newMessengerSession(a.platform, msg.Sender.ID, a.profileID, audioURL)
	select {
	case a.sessions <- sess:
	case <-a.done:
	}
	return nil
}

// messengerSession handles a single voice exchange from a messenger platform.
type messengerSession struct {
	platform  string
	callerID  string
	profileID string
	audioURL  string
	audioIn   chan []byte
	audioOut  chan []byte
	textOut   chan string
}

func newMessengerSession(platform, callerID, profileID, audioURL string) *messengerSession {
	s := &messengerSession{
		platform:  platform,
		callerID:  callerID,
		profileID: profileID,
		audioURL:  audioURL,
		audioIn:   make(chan []byte, 32),
		audioOut:  make(chan []byte, 32),
		textOut:   make(chan string, 8),
	}
	// TODO: fetch audio from audioURL, decode to PCM, push to audioIn
	// This is wired up in the orchestrator once audio decoding is available.
	return s
}

func (s *messengerSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *messengerSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *messengerSession) TextOut() chan<- string  { return s.textOut }
func (s *messengerSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: s.platform, CallerID: s.callerID}
}
func (s *messengerSession) Close() error {
	close(s.audioOut)
	close(s.textOut)
	return nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestMessengerAdapter -v 2>&1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/voice/adapter_messenger.go internal/voice/adapter_messenger_test.go
git commit -m "feat(voice): messenger channel adapter for Telegram/WhatsApp voice messages"
```

---

## Task 9: Twilio channel adapter

**Files:**
- Create: `internal/voice/adapter_twilio.go`
- Test: `internal/voice/adapter_twilio_test.go`

**Step 1: Write the failing tests**

```go
// internal/voice/adapter_twilio_test.go
package voice_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"hop.top/aps/internal/voice"
)

func TestTwilioAdapter_AcceptsMediaStream(t *testing.T) {
	adapter := voice.NewTwilioAdapter("+15551234567", "profile-1")
	sessions, err := adapter.Accept()
	assert.NoError(t, err)

	srv := httptest.NewServer(adapter)
	defer srv.Close()
	defer adapter.Close()

	// Twilio connects via WebSocket to /twilio/media-stream
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/twilio/media-stream"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	defer conn.Close()

	sess := <-sessions
	assert.Equal(t, "twilio", sess.Meta().ChannelType)
	assert.Equal(t, "+15551234567", sess.Meta().CallerID)
	sess.Close()
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/voice/... -run TestTwilioAdapter -v 2>&1
```

Expected: compile error.

**Step 3: Implement Twilio adapter**

```go
// internal/voice/adapter_twilio.go
package voice

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

// TwilioAdapter accepts Twilio Media Streams WebSocket connections.
// Twilio sends mulaw/8kHz audio; the orchestrator is responsible for resampling to PCM/24kHz.
type TwilioAdapter struct {
	phoneNumber string
	profileID   string
	sessions    chan ChannelSession
	done        chan struct{}
}

func NewTwilioAdapter(phoneNumber, profileID string) *TwilioAdapter {
	return &TwilioAdapter{
		phoneNumber: phoneNumber,
		profileID:   profileID,
		sessions:    make(chan ChannelSession, 8),
		done:        make(chan struct{}),
	}
}

func (a *TwilioAdapter) Accept() (<-chan ChannelSession, error) {
	return a.sessions, nil
}

func (a *TwilioAdapter) Close() error {
	close(a.done)
	return nil
}

// ServeHTTP handles Twilio Media Stream WebSocket at /twilio/media-stream.
func (a *TwilioAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/twilio/media-stream" {
		http.NotFound(w, r)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("twilio adapter: websocket upgrade failed", "err", err)
		return
	}
	log.Info("twilio adapter: call connected", "phone", a.phoneNumber)
	sess := newTwilioSession(conn, a.phoneNumber, a.profileID)
	select {
	case a.sessions <- sess:
	case <-a.done:
		conn.Close()
	}
}

type twilioSession struct {
	conn        *websocket.Conn
	phoneNumber string
	profileID   string
	audioIn     chan []byte
	audioOut    chan []byte
	textOut     chan string
}

func newTwilioSession(conn *websocket.Conn, phoneNumber, profileID string) *twilioSession {
	s := &twilioSession{
		conn:        conn,
		phoneNumber: phoneNumber,
		profileID:   profileID,
		audioIn:     make(chan []byte, 32),
		audioOut:    make(chan []byte, 32),
		textOut:     make(chan string, 8),
	}
	go s.readLoop()
	go s.writeLoop()
	return s
}

func (s *twilioSession) AudioIn() <-chan []byte  { return s.audioIn }
func (s *twilioSession) AudioOut() chan<- []byte { return s.audioOut }
func (s *twilioSession) TextOut() chan<- string  { return s.textOut }
func (s *twilioSession) Meta() SessionMeta {
	return SessionMeta{ProfileID: s.profileID, ChannelType: "twilio", CallerID: s.phoneNumber}
}
func (s *twilioSession) Close() error { return s.conn.Close() }

func (s *twilioSession) readLoop() {
	defer close(s.audioIn)
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		// TODO: decode Twilio mulaw/8kHz JSON payload to raw PCM bytes
		s.audioIn <- msg
	}
}

func (s *twilioSession) writeLoop() {
	for frame := range s.audioOut {
		if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			return
		}
	}
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/voice/... -run TestTwilioAdapter -v 2>&1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add internal/voice/adapter_twilio.go internal/voice/adapter_twilio_test.go
git commit -m "feat(voice): Twilio Media Streams channel adapter"
```

---

## Task 10: CLI commands

**Files:**
- Create: `internal/cli/voice.go`

**Step 1: Check how init() registers commands**

Read `internal/cli/action.go` to confirm the `init()` + `rootCmd.AddCommand()` pattern, then implement:

```go
// internal/cli/voice.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hop.top/aps/internal/voice"
)

var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "Manage voice sessions and the voice backend service",
}

var voiceServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "Control the voice backend service",
}

var voiceServiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the voice backend service",
	Run: func(cmd *cobra.Command, args []string) {
		// Load global config, start backend
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if err := mgr.Start(nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Voice backend started.")
	},
}

var voiceServiceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the voice backend service",
	Run: func(cmd *cobra.Command, args []string) {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if err := mgr.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Voice backend stopped.")
	},
}

var voiceServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show voice backend service status",
	Run: func(cmd *cobra.Command, args []string) {
		mgr := voice.NewBackendManager(voice.GlobalBackendConfig{})
		if mgr.IsRunning() {
			fmt.Println("running")
		} else {
			fmt.Println("stopped")
		}
	},
}

var voiceSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage active voice sessions",
}

var voiceSessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active voice sessions",
	Run: func(cmd *cobra.Command, args []string) {
		sm := voice.NewSessionManager()
		sessions := sm.List()
		if len(sessions) == 0 {
			fmt.Println("No active voice sessions.")
			return
		}
		for _, s := range sessions {
			fmt.Printf("%s  profile=%-20s  channel=%-10s  state=%s\n",
				s.ID, s.ProfileID, s.ChannelType, s.State)
		}
	},
}

var voiceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a voice session",
	Run: func(cmd *cobra.Command, args []string) {
		profileID, _ := cmd.Flags().GetString("profile")
		channel, _ := cmd.Flags().GetString("channel")
		fmt.Printf("Starting voice session: profile=%s channel=%s\n", profileID, channel)
		// TODO: wire up orchestrator once implemented
	},
}

func init() {
	rootCmd.AddCommand(voiceCmd)
	voiceCmd.AddCommand(voiceServiceCmd)
	voiceServiceCmd.AddCommand(voiceServiceStartCmd)
	voiceServiceCmd.AddCommand(voiceServiceStopCmd)
	voiceServiceCmd.AddCommand(voiceServiceStatusCmd)
	voiceCmd.AddCommand(voiceSessionCmd)
	voiceSessionCmd.AddCommand(voiceSessionListCmd)
	voiceCmd.AddCommand(voiceStartCmd)
	voiceStartCmd.Flags().String("profile", "", "Profile ID to use for this voice session")
	voiceStartCmd.Flags().String("channel", "web", "Channel: web | tui | telegram | twilio")
}
```

**Step 2: Build to verify no compile errors**

```bash
go build ./... 2>&1
```

Expected: no errors.

**Step 3: Smoke test CLI**

```bash
go run ./cmd/aps voice --help 2>&1
go run ./cmd/aps voice service --help 2>&1
go run ./cmd/aps voice session list 2>&1
```

Expected: help text and "No active voice sessions."

**Step 4: Commit**

```bash
git add internal/cli/voice.go
git commit -m "feat(voice): CLI commands for voice service and session management"
```

---

## Task 11: Run full test suite & fix any issues

**Step 1: Run all voice tests**

```bash
go test ./internal/voice/... -v 2>&1
```

Expected: all PASS.

**Step 2: Run full suite**

```bash
go test ./... 2>&1 | tail -30
```

Expected: no new failures.

**Step 3: Build**

```bash
go build ./... 2>&1
```

Expected: clean build.

**Step 4: Commit if any fixes were needed**

```bash
git add -p
git commit -m "fix(voice): resolve issues from full test suite run"
```

---

## What's deferred (not in this plan)

- **Orchestrator pipeline** (audio↔backend WebSocket bridge): Task 12+ once channels are wired
- **Audio resampling** (Twilio mulaw→PCM, messenger OGG→PCM): requires audio library decision
- **Intent classifier** for ambient routing: separate design
- **Global config loading** (`~/.config/aps/config.yaml` voice section): wire into existing config loader
- **Hex TUI binary**: separate Haskell project
- **PersonaPlex React client static serving**: add file serving to WebAdapter once build pipeline is decided
