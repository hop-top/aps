package voice

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"charm.land/log/v2"
	"hop.top/kit/go/console/ps"
	"hop.top/kit/go/core/xdg"
)

// pidFileName is the canonical PID file under $XDG_RUNTIME_DIR/aps/.
const pidFileName = "voice.pid"

// stopGracePeriod is how long Stop waits between SIGTERM and SIGKILL.
const stopGracePeriod = 2 * time.Second

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
//
// Lifecycle state lives on disk as a PID file at the path returned by
// PIDFilePath() — by default $XDG_RUNTIME_DIR/aps/voice.pid. The PID file
// is read via kit/console/ps so liveness checks share semantics with
// every other hop.top tool. Tests can override the path by constructing
// a manager with NewBackendManagerWithPIDPath.
type BackendManager struct {
	cfg     GlobalBackendConfig
	pidPath string
	mu      sync.Mutex
}

// NewBackendManager returns a manager that writes its PID file to the
// canonical XDG runtime location ($XDG_RUNTIME_DIR/aps/voice.pid, with
// fallback to $TMPDIR/aps/voice.pid when XDG_RUNTIME_DIR is unset).
func NewBackendManager(cfg GlobalBackendConfig) *BackendManager {
	return &BackendManager{cfg: cfg}
}

// NewBackendManagerWithPIDPath is like NewBackendManager but uses an
// explicit PID-file path. Intended for tests; production callers should
// use NewBackendManager so every aps invocation agrees on the location.
func NewBackendManagerWithPIDPath(cfg GlobalBackendConfig, pidPath string) *BackendManager {
	return &BackendManager{cfg: cfg, pidPath: pidPath}
}

// PIDFilePath returns the absolute path to the PID file this manager
// reads/writes. Resolution order:
//  1. explicit path passed via NewBackendManagerWithPIDPath
//  2. $XDG_RUNTIME_DIR/aps/voice.pid via kit/core/xdg
//  3. $TMPDIR/aps/voice.pid (if XDG resolution fails)
func (m *BackendManager) PIDFilePath() (string, error) {
	if m.pidPath != "" {
		return m.pidPath, nil
	}
	path, err := xdg.RuntimeFile("aps", pidFileName)
	if err != nil {
		// Last-ditch fallback so users on bare environments still get
		// a deterministic location. This mirrors xdg's own fallback
		// for runtime files when XDG_RUNTIME_DIR is unavailable.
		return filepath.Join(os.TempDir(), "aps", pidFileName), nil //nolint:nilerr // intentional
	}
	return path, nil
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

// Start launches the managed backend process as a detached child.
//
// Behaviour:
//   - If profileCfg.URL is set, Start is a no-op (external instance).
//   - If a PID file exists and refers to a live process, returns an
//     error — concurrent Start is refused.
//   - If a PID file exists but the process is gone (zombie pid),
//     it is silently removed before starting fresh.
//   - On success, writes the new child's PID to the PID file.
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

	pidPath, err := m.PIDFilePath()
	if err != nil {
		return fmt.Errorf("resolve voice pid path: %w", err)
	}

	// Probe existing pid file: refuse to start if alive, sweep if dead.
	if entry, perr := ps.EntryFromPIDFile(pidPath); perr == nil {
		if entry.Status == ps.StatusRunning {
			return fmt.Errorf("voice backend already running (pid %s)", entry.ID)
		}
		// stopped → stale pid, remove and continue
		_ = os.Remove(pidPath)
	}

	if err := os.MkdirAll(filepath.Dir(pidPath), 0o700); err != nil {
		return fmt.Errorf("create voice runtime dir: %w", err)
	}

	cmd := exec.Command(binCfg.Bin, binCfg.Args...) //nolint:gosec
	cmd.SysProcAttr = detachSysProcAttr()
	// Detach stdio so the child outlives the CLI process; nil leaves
	// stdio at the parent's defaults, which is fine for tests and for
	// users running interactively. A future enhancement may redirect
	// to a log file under XDG_STATE_DIR/aps/voice.log.
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start voice backend %q: %w", t, err)
	}
	pid := cmd.Process.Pid

	if err := writePIDFile(pidPath, pid); err != nil {
		// Try to clean up the orphaned child rather than leaking it.
		_ = cmd.Process.Kill()
		return fmt.Errorf("write voice pid file: %w", err)
	}

	// Release goroutine so the kernel reaps the child when it exits;
	// we are explicitly *not* keeping cmd around — lifecycle is
	// driven by the PID file going forward.
	go func() { _ = cmd.Wait() }()

	log.Info("voice backend started", "type", t, "pid", pid, "pidfile", pidPath)
	return nil
}

// Stop terminates the managed backend process if running.
//
// No-op if the PID file is absent or refers to a dead process. Sends
// SIGTERM, waits up to stopGracePeriod, then SIGKILL. Removes the PID
// file on success.
func (m *BackendManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pidPath, err := m.PIDFilePath()
	if err != nil {
		return fmt.Errorf("resolve voice pid path: %w", err)
	}

	entry, err := ps.EntryFromPIDFile(pidPath)
	if err != nil {
		// No pid file — nothing to stop.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		// Unparseable pid file: blow it away and report success.
		_ = os.Remove(pidPath)
		return nil
	}

	if entry.Status != ps.StatusRunning {
		_ = os.Remove(pidPath)
		return nil
	}

	pid, err := strconv.Atoi(entry.ID)
	if err != nil {
		return fmt.Errorf("parse pid: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find voice backend process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("signal voice backend: %w", err)
	}

	deadline := time.Now().Add(stopGracePeriod)
	for time.Now().Before(deadline) {
		if !looksAlive(pid) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if looksAlive(pid) {
		_ = proc.Kill()
	}

	_ = os.Remove(pidPath)
	return nil
}

// IsRunning reports whether the managed process is alive, as reported
// by kit/console/ps.EntryFromPIDFile.
func (m *BackendManager) IsRunning() bool {
	pidPath, err := m.PIDFilePath()
	if err != nil {
		return false
	}
	entry, err := ps.EntryFromPIDFile(pidPath)
	if err != nil {
		return false
	}
	return entry.Status == ps.StatusRunning
}

// Status returns a structured snapshot of the backend service. PID is 0
// when no PID file is on disk.
type Status struct {
	Running bool
	PID     int
	PIDFile string
}

// Status returns the current lifecycle state.
func (m *BackendManager) Status() Status {
	pidPath, err := m.PIDFilePath()
	if err != nil {
		return Status{}
	}
	entry, err := ps.EntryFromPIDFile(pidPath)
	if err != nil {
		return Status{PIDFile: pidPath}
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(entry.ID))
	return Status{
		Running: entry.Status == ps.StatusRunning,
		PID:     pid,
		PIDFile: pidPath,
	}
}

// writePIDFile writes pid to path atomically (write-then-rename).
func writePIDFile(path string, pid int) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("create temp pid file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := fmt.Fprintf(tmp, "%d\n", pid); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write pid: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp pid file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename pid file into place: %w", err)
	}
	return nil
}
