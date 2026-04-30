package isolation

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hop.top/aps/internal/core"
	"hop.top/aps/internal/core/session"
)

type ProcessIsolation struct {
	context     *ExecutionContext
	tmuxSocket  string
	tmuxSession string
	useTmux     bool
}

func NewProcessIsolation() *ProcessIsolation {
	return &ProcessIsolation{}
}

func (p *ProcessIsolation) PrepareContext(profileID string) (*ExecutionContext, error) {
	_, err := core.LoadProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidProfile, err)
	}

	profileDir, err := core.GetProfileDir(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile dir: %w", err)
	}

	profileYaml, err := core.GetProfilePath(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile path: %w", err)
	}

	secretsPath := filepath.Join(profileDir, "secrets.env")
	agentsDir, err := core.GetAgentsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get agents dir: %w", err)
	}
	docsDir := filepath.Join(agentsDir, "docs")

	context := &ExecutionContext{
		ProfileID:   profileID,
		ProfileDir:  profileDir,
		ProfileYaml: profileYaml,
		SecretsPath: secretsPath,
		DocsDir:     docsDir,
		Environment: make(map[string]string),
		WorkingDir:  profileDir,
	}

	p.context = context
	return context, nil
}

func (p *ProcessIsolation) SetupEnvironment(cmd interface{}) error {
	execCmd, ok := cmd.(*exec.Cmd)
	if !ok {
		return fmt.Errorf("cmd must be *exec.Cmd")
	}

	if p.context == nil {
		return fmt.Errorf("context not prepared")
	}

	env := os.Environ()

	config, _ := core.LoadConfig()
	prefix := config.Prefix

	apsEnv := map[string]string{
		fmt.Sprintf("%s_PROFILE_ID", prefix):       p.context.ProfileID,
		fmt.Sprintf("%s_PROFILE_DIR", prefix):      p.context.ProfileDir,
		fmt.Sprintf("%s_PROFILE_YAML", prefix):     p.context.ProfileYaml,
		fmt.Sprintf("%s_PROFILE_SECRETS", prefix):  p.context.SecretsPath,
		fmt.Sprintf("%s_PROFILE_DOCS_DIR", prefix): p.context.DocsDir,
	}

	for k, v := range apsEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	secrets, err := core.LoadSecrets(p.context.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	profile, err := core.LoadProfileFromPath(p.context.ProfileID, p.context.ProfileYaml)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	if profile.Git.Enabled {
		gitConfigPath := filepath.Join(p.context.ProfileDir, "gitconfig")
		if _, err := os.Stat(gitConfigPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_CONFIG_GLOBAL=%s", gitConfigPath))
		}
	}

	if profile.SSH.Enabled && profile.SSH.KeyPath != "" {
		internalKeyPath := filepath.Join(p.context.ProfileDir, "ssh.key")
		if _, err := os.Stat(internalKeyPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -F /dev/null", internalKeyPath))
		}
	}

	execCmd.Env = env
	execCmd.Dir = p.context.WorkingDir

	return nil
}

func (p *ProcessIsolation) Execute(command string, args []string) error {
	p.tmuxSocket = filepath.Join(os.TempDir(), fmt.Sprintf("aps-tmux-%s-socket", p.context.ProfileID))
	p.tmuxSession = fmt.Sprintf("aps-%s-%d", p.context.ProfileID, time.Now().Unix())
	p.useTmux = true

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := p.SetupEnvironment(cmd); err != nil {
		return err
	}

	if err := p.setupTmuxSession(command, args); err != nil {
		return err
	}

	return nil
}

func (p *ProcessIsolation) setupTmuxSession(command string, args []string) error {
	fullCommand := strings.Join(append([]string{command}, args...), " ")

	tmuxNewCmd := exec.Command("tmux", "-S", p.tmuxSocket, "new-session", "-d", "-s", p.tmuxSession, "-n", "aps", fullCommand)
	tmuxNewCmd.Stdout = os.Stdout
	tmuxNewCmd.Stderr = os.Stderr

	if err := tmuxNewCmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	if err := p.registerSession(); err != nil {
		p.cleanupTmux()
		return fmt.Errorf("failed to register session: %w", err)
	}

	fmt.Printf("Session started: %s\n", p.tmuxSession)
	fmt.Printf("Tmux socket: %s\n", p.tmuxSocket)
	fmt.Printf("Attach with: aps session attach %s\n", p.tmuxSession)

	return nil
}

func (p *ProcessIsolation) registerSession() error {
	cmd := exec.Command("tmux", "-S", p.tmuxSocket, "display-message", "-p", "'#{pid}'")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get tmux session PID: %w", err)
	}

	pidStr := strings.Trim(strings.TrimSpace(string(output)), "'")
	var pid int
	_, err = fmt.Sscanf(pidStr, "%d", &pid)
	if err != nil {
		return fmt.Errorf("failed to parse PID: %w", err)
	}

	registry := session.GetRegistry()
	sess := &session.SessionInfo{
		ID:          p.tmuxSession,
		ProfileID:   p.context.ProfileID,
		ProfileDir:  p.context.ProfileDir,
		Command:     strings.Join(append([]string{"tmux"}, "-S", p.tmuxSocket, "new-session"), " "),
		PID:         pid,
		Status:      session.SessionActive,
		Tier:        session.TierStandard,
		TmuxSocket:  p.tmuxSocket,
		TmuxSession: p.tmuxSession,
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
		Environment: map[string]string{
			"tmux_socket":  p.tmuxSocket,
			"tmux_session": p.tmuxSession,
		},
	}

	return registry.Register(sess)
}

// cleanupTmux tears down the tmux session for this isolation. On a
// clean shutdown (success or benign already-dead) the session is
// removed from the registry. On a real teardown failure, the session
// is marked SessionErrored and LEFT in the registry so operators can
// observe it via `aps session list` and remove it explicitly with
// `aps session delete`.
func (p *ProcessIsolation) cleanupTmux() {
	if p.tmuxSocket == "" || p.tmuxSession == "" {
		return
	}

	cmd := exec.Command("tmux", "-S", p.tmuxSocket, "kill-session", "-t", p.tmuxSession)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	killErr := cmd.Run()

	registry := session.GetRegistry()

	if killErr != nil && !session.IsBenignTmuxError(stderr.String()) {
		// Real teardown failure — mark errored, leave registered.
		fmt.Fprintf(os.Stderr, "isolation: tmux teardown failed for session %s: %v: %s\n",
			p.tmuxSession, killErr, strings.TrimSpace(stderr.String()))
		_ = registry.UpdateStatus(p.tmuxSession, session.SessionErrored)
		return
	}

	_ = registry.Unregister(p.tmuxSession)
}

func (p *ProcessIsolation) ExecuteAction(actionID string, payload []byte) error {
	action, err := core.GetAction(p.context.ProfileID, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}

	var cmd *exec.Cmd
	switch action.Type {
	case "sh":
		cmd = exec.Command("sh", action.Path)
	case "py":
		cmd = exec.Command("python3", action.Path)
	case "js":
		cmd = exec.Command("node", action.Path)
	default:
		cmd = exec.Command(action.Path)
	}

	if len(payload) > 0 {
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		go func() {
			defer pipe.Close()
			pipe.Write(payload)
		}()
	} else {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := p.SetupEnvironment(cmd); err != nil {
		return err
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (p *ProcessIsolation) Cleanup() error {
	if p.useTmux {
		p.cleanupTmux()
	}
	p.context = nil
	return nil
}

func (p *ProcessIsolation) Validate() error {
	if p.context == nil {
		return fmt.Errorf("context not prepared")
	}

	if _, err := os.Stat(p.context.ProfileDir); os.IsNotExist(err) {
		return fmt.Errorf("profile directory does not exist: %s", p.context.ProfileDir)
	}

	if _, err := os.Stat(p.context.ProfileYaml); os.IsNotExist(err) {
		return fmt.Errorf("profile.yaml does not exist: %s", p.context.ProfileYaml)
	}

	return nil
}

func (p *ProcessIsolation) IsAvailable() bool {
	return true
}
