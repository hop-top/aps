package voice

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"charm.land/log/v2"
)

// BackendBinConfig holds the path and args for a managed backend binary.
type BackendBinConfig struct {
	Bin  string   `yaml:"bin"`
	Args []string `yaml:"args,omitempty"`
}

// GlobalBackendConfig is the voice section of ~/.config/aps/config.yaml.
type GlobalBackendConfig struct {
	DefaultBackend string                      `yaml:"default_backend,omitempty"`
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

// ResolveType returns the effective backend type.
// For "auto", walks the platform-preferred order and picks the first configured binary.
// Falls back to "compatible" if nothing is configured.
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
// If cfg.URL is set, it is used directly. Otherwise defaults to the locally managed instance.
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
