package adapter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"hop.top/cxr"
)

// Manager owns adapter runtimes and routes start requests via cxr.Router.
type Manager struct {
	mu       sync.RWMutex
	runtimes map[string]*AdapterRuntime
	stopCh   chan struct{}
}

func NewManager() *Manager {
	return &Manager{
		runtimes: make(map[string]*AdapterRuntime),
		stopCh:   make(chan struct{}),
	}
}

func (m *Manager) CreateAdapter(name string, deviceType AdapterType, strategy LoadingStrategy, scope AdapterScope, profileID string) (*Adapter, error) {
	if !IsAdapterTypeValid(deviceType) {
		return nil, ErrAdapterTypeInvalid(string(deviceType))
	}

	if !IsAdapterTypeImplemented(deviceType) {
		return nil, ErrAdapterTypeNotImplemented(deviceType)
	}

	exists, err := AdapterExists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAdapterAlreadyExists(name)
	}

	if strategy == "" {
		strategy = DefaultStrategyForType(deviceType)
	}

	device := &Adapter{
		Name:      name,
		Type:      deviceType,
		Scope:     scope,
		ProfileID: profileID,
		Strategy:  strategy,
		Config:    make(map[string]any),
	}

	if err := SaveAdapter(device); err != nil {
		return nil, err
	}

	return device, nil
}

// StartAdapter routes to the appropriate handler via cxr.Router.
func (m *Manager) StartAdapter(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, err := LoadAdapter(name)
	if err != nil {
		return err
	}

	if runtime, exists := m.runtimes[name]; exists {
		if runtime.State == StateRunning {
			return ErrAdapterAlreadyRunning(name)
		}
	}

	router, err := m.buildRouter(device)
	if err != nil {
		return err
	}

	req := cxr.Request{
		Operation: string(device.Strategy),
		Config:    map[string]any{"handler": string(device.Strategy)},
	}

	handler, err := router.Route(req)
	if err != nil {
		return ErrStrategyInvalid(string(device.Strategy))
	}

	_, err = handler.Exec(ctx, req)
	if err != nil {
		return fmt.Errorf("adapter exec failed: %w", err)
	}
	return nil
}

// buildRouter constructs a cxr.Router with strategy-specific handlers for device.
func (m *Manager) buildRouter(device *Adapter) (*cxr.Router, error) {
	switch device.Strategy {
	case StrategySubprocess:
		h := m.newSubprocessHandler(device)
		return cxr.NewRouter([]cxr.Handler{h}), nil
	case StrategyBuiltin:
		h := m.newBuiltinHandler(device)
		return cxr.NewRouter([]cxr.Handler{h}), nil
	case StrategyScript:
		return nil, fmt.Errorf("script strategy does not support persistent start")
	default:
		return nil, ErrStrategyInvalid(string(device.Strategy))
	}
}

// ── subprocess handler ────────────────────────────────────────────────────────

// subprocessHandler wraps subprocess start logic as a cxr.Handler.
// Uses cxr.ProcessHandler for env building; overrides Exec for daemon semantics
// (log-file stdio, Setpgid, background wait goroutine).
type subprocessHandler struct {
	m      *Manager
	device *Adapter
	ph     *cxr.ProcessHandler // provides BuildEnv; Exec not used directly
}

func (m *Manager) newSubprocessHandler(device *Adapter) *subprocessHandler {
	ph := cxr.NewProcessHandler(device.Name, "", nil, nil)
	ph.BuildEnv = func(_ cxr.Request) []string {
		return m.buildAdapterEnv(device)
	}
	return &subprocessHandler{m: m, device: device, ph: ph}
}

func (h *subprocessHandler) ID() string { return string(StrategySubprocess) }

func (h *subprocessHandler) CanHandle(req cxr.Request) bool {
	return req.Operation == string(StrategySubprocess)
}

func (h *subprocessHandler) Exec(ctx context.Context, _ cxr.Request) (cxr.Result, error) {
	if err := h.m.startSubprocess(ctx, h.device, h.ph); err != nil {
		return cxr.Result{}, err
	}
	return cxr.Result{}, nil
}

func (h *subprocessHandler) Probe(_ context.Context) (*cxr.Capabilities, error) {
	return &cxr.Capabilities{}, nil
}

// ── builtin handler ───────────────────────────────────────────────────────────

type builtinHandler struct {
	m      *Manager
	device *Adapter
}

func (m *Manager) newBuiltinHandler(device *Adapter) *builtinHandler {
	return &builtinHandler{m: m, device: device}
}

func (h *builtinHandler) ID() string { return string(StrategyBuiltin) }

func (h *builtinHandler) CanHandle(req cxr.Request) bool {
	return req.Operation == string(StrategyBuiltin)
}

func (h *builtinHandler) Exec(ctx context.Context, _ cxr.Request) (cxr.Result, error) {
	h.m.startBuiltin(ctx, h.device)
	return cxr.Result{}, nil
}

func (h *builtinHandler) Probe(_ context.Context) (*cxr.Capabilities, error) {
	return &cxr.Capabilities{}, nil
}

// ── start helpers ─────────────────────────────────────────────────────────────

// startSubprocess launches the adapter as a background process.
// ph provides the BuildEnv hook so env construction is unified with cxr.
func (m *Manager) startSubprocess(ctx context.Context, device *Adapter, ph *cxr.ProcessHandler) error {
	executablePath, err := m.findAdapterExecutable(device)
	if err != nil {
		return ErrStartFailed(device.Name, err)
	}

	cmd := exec.CommandContext(ctx, executablePath)
	cmd.Dir = device.Path

	// Env: system env + adapter-specific env from ProcessHandler.BuildEnv hook.
	extraEnv := ph.BuildEnv(cxr.Request{Operation: string(device.Strategy)})
	cmd.Env = append(os.Environ(), extraEnv...)

	cmd.SysProcAttr = sysProcAttr()

	stdoutPath := filepath.Join(device.Path, "stdout.log")
	stderrPath := filepath.Join(device.Path, "stderr.log")

	stdout, err := os.OpenFile(stdoutPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return ErrStartFailed(device.Name, err)
	}
	stderr, err := os.OpenFile(stderrPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		stdout.Close()
		return ErrStartFailed(device.Name, err)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return ErrStartFailed(device.Name, err)
	}

	now := time.Now()
	m.runtimes[device.Name] = &AdapterRuntime{
		Name:      device.Name,
		State:     StateRunning,
		Health:    HealthUnknown,
		PID:       cmd.Process.Pid,
		StartedAt: &now,
	}

	go func() {
		cmd.Wait()
		stdout.Close()
		stderr.Close()

		m.mu.Lock()
		if rt, exists := m.runtimes[device.Name]; exists {
			if rt.State == StateRunning {
				rt.State = StateStopped
			}
		}
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) startBuiltin(_ context.Context, device *Adapter) {
	now := time.Now()
	m.runtimes[device.Name] = &AdapterRuntime{
		Name:      device.Name,
		State:     StateRunning,
		Health:    HealthHealthy,
		StartedAt: &now,
	}
}

// ── stop ──────────────────────────────────────────────────────────────────────

func (m *Manager) StopAdapter(ctx context.Context, name string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, err := LoadAdapter(name)
	if err != nil {
		return err
	}

	runtime, exists := m.runtimes[name]
	if !exists || runtime.State == StateStopped {
		return ErrAdapterAlreadyStopped(name)
	}

	switch device.Strategy {
	case StrategySubprocess:
		return m.stopSubprocess(ctx, device, runtime, force)
	case StrategyBuiltin:
		return m.stopBuiltin(ctx, device, runtime)
	case StrategyScript:
		return nil
	default:
		return ErrStrategyInvalid(string(device.Strategy))
	}
}

func (m *Manager) stopSubprocess(ctx context.Context, device *Adapter, runtime *AdapterRuntime, force bool) error {
	if runtime.PID == 0 {
		return nil
	}

	process, err := os.FindProcess(runtime.PID)
	if err != nil {
		return ErrStopFailed(device.Name, err)
	}

	sig := termSignal()
	if force {
		sig = killSignal()
	}

	if err := process.Signal(sig); err != nil {
		if err == os.ErrProcessDone {
			runtime.State = StateStopped
			return nil
		}
		return ErrStopFailed(device.Name, err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	timeout := 5 * time.Second
	if force {
		timeout = 1 * time.Second
	}

	select {
	case <-done:
		runtime.State = StateStopped
		return nil
	case <-time.After(timeout):
		if !force {
			return m.stopSubprocess(ctx, device, runtime, true)
		}
		return ErrStopFailed(device.Name, fmt.Errorf("process did not terminate after SIGKILL"))
	}
}

func (m *Manager) stopBuiltin(ctx context.Context, device *Adapter, runtime *AdapterRuntime) error {
	runtime.State = StateStopped
	return nil
}

// ── query ─────────────────────────────────────────────────────────────────────

func (m *Manager) GetRuntime(name string) (*AdapterRuntime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if runtime, exists := m.runtimes[name]; exists {
		return runtime, nil
	}

	return &AdapterRuntime{
		Name:  name,
		State: StateStopped,
	}, nil
}

func (m *Manager) HealthCheck(ctx context.Context, name string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runtime, exists := m.runtimes[name]
	if !exists {
		return ErrHealthCheckFailed(name, fmt.Errorf("adapter not running"))
	}

	if runtime.State != StateRunning {
		return ErrHealthCheckFailed(name, fmt.Errorf("adapter not running (state: %s)", runtime.State))
	}

	now := time.Now()
	runtime.LastCheck = &now
	runtime.Health = HealthHealthy

	return nil
}

func (m *Manager) LinkAdapter(deviceName, profileID string) error {
	device, err := LoadAdapter(deviceName)
	if err != nil {
		return err
	}

	for _, p := range device.LinkedTo {
		if p == profileID {
			return nil
		}
	}

	device.LinkedTo = append(device.LinkedTo, profileID)
	return SaveAdapter(device)
}

func (m *Manager) UnlinkAdapter(deviceName, profileID string) error {
	device, err := LoadAdapter(deviceName)
	if err != nil {
		return err
	}

	found := false
	linked := make([]string, 0, len(device.LinkedTo))
	for _, p := range device.LinkedTo {
		if p == profileID {
			found = true
			continue
		}
		linked = append(linked, p)
	}

	if !found {
		return fmt.Errorf("adapter '%s' is not linked to profile '%s'", deviceName, profileID)
	}

	device.LinkedTo = linked
	return SaveAdapter(device)
}

func (m *Manager) GetAdapterLogs(name string, tail int, follow bool) ([]string, error) {
	device, err := LoadAdapter(name)
	if err != nil {
		return nil, err
	}

	logPath := filepath.Join(device.Path, "stdout.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	lines := splitLines(string(data))
	if tail > 0 && len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}

	return lines, nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, runtime := range m.runtimes {
		if runtime.State == StateRunning {
			if err := m.StopAdapter(ctx, name, false); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}
	return nil
}

// ── env / exec helpers ────────────────────────────────────────────────────────

func (m *Manager) findAdapterExecutable(device *Adapter) (string, error) {
	executableName := fmt.Sprintf("aps-adapter-%s", device.Name)
	if path, err := exec.LookPath(executableName); err == nil {
		return path, nil
	}

	localPath := filepath.Join(device.Path, "run")
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	return "", fmt.Errorf("adapter executable not found for %s", device.Name)
}

func (m *Manager) buildAdapterEnv(device *Adapter) []string {
	env := []string{
		fmt.Sprintf("APS_ADAPTER_NAME=%s", device.Name),
		fmt.Sprintf("APS_ADAPTER_TYPE=%s", device.Type),
		fmt.Sprintf("APS_ADAPTER_PATH=%s", device.Path),
	}
	for k, v := range device.Config {
		env = append(env, fmt.Sprintf("APS_ADAPTER_CONFIG_%s=%v", k, v))
	}
	return env
}

func (m *Manager) ListRunningAdapters() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var running []string
	for name, runtime := range m.runtimes {
		if runtime.State == StateRunning {
			running = append(running, name)
		}
	}
	return running
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
