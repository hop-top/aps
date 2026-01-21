package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"oss-aps-cli/internal/core"
)

type ProcessIsolation struct {
	context *ExecutionContext
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

	profile, err := core.LoadProfile(p.context.ProfileID)
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
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
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
