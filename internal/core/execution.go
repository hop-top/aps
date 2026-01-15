package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InjectEnvironment prepares the environment variables for a command
func InjectEnvironment(cmd *exec.Cmd, profile *Profile) error {
	profileDir, err := GetProfileDir(profile.ID)
	if err != nil {
		return err
	}
	profileYaml, err := GetProfilePath(profile.ID)
	if err != nil {
		return err
	}
	secretsPath := filepath.Join(profileDir, "secrets.env")
	agentsDir, err := GetAgentsDir()
	if err != nil {
		return err
	}
	docsDir := filepath.Join(agentsDir, "docs")

	// 1. Start with parent environment
	env := os.Environ()

	// 2. Inject APS specific variables
	apsEnv := map[string]string{
		"AGENT_PROFILE_ID":        profile.ID,
		"AGENT_PROFILE_DIR":       profileDir,
		"AGENT_PROFILE_YAML":      profileYaml,
		"AGENT_PROFILE_SECRETS":   secretsPath,
		"AGENT_PROFILE_DOCS_DIR":  docsDir,
	}

	for k, v := range apsEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 3. Inject Secrets
	secrets, err := LoadSecrets(secretsPath)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 4. Inject Git Config (Module)
	if profile.Git.Enabled {
		gitConfigPath := filepath.Join(profileDir, "gitconfig")
		if _, err := os.Stat(gitConfigPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_CONFIG_GLOBAL=%s", gitConfigPath))
		}
	}

	// 5. Inject SSH Config (Module)
	if profile.SSH.Enabled && profile.SSH.KeyPath != "" {
		// Resolve relative paths (like ~) if necessary, simplified here to use raw or absolute
		// Ideally we expand ~, but for now we assume valid path or let ssh handle it if absolute
		// Actually the spec says "If ssh.key exists in the profile directory... APS may inject"
		// The spec example shows: GIT_SSH_COMMAND=ssh -i <profile-dir>/ssh.key -F /dev/null
		// But T008 goal is generic environment. Let's stick to the spec's module logic if we were implementing modules fully.
		// For T008 scope "InjectEnvironment", we'll stick to spec section 9 and 8.
		// Spec 8.2 says: "If ssh.key exists... and SSH is enabled... inject GIT_SSH_COMMAND"
		// Let's check for ssh.key in profile dir as per spec 8.2 logic
		internalKeyPath := filepath.Join(profileDir, "ssh.key")
		if _, err := os.Stat(internalKeyPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -F /dev/null", internalKeyPath))
		}
	}

	cmd.Env = env
	return nil
}

// RunCommand executes a command within a profile's context
func RunCommand(profileID string, command string, args []string) error {
	profile, err := LoadProfile(profileID)
	if err != nil {
		return err
	}

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := InjectEnvironment(cmd, profile); err != nil {
		return err
	}

	return cmd.Run()
}

// RunAction executes a defined action
func RunAction(profileID string, actionID string, payload []byte) error {
	action, err := GetAction(profileID, actionID)
	if err != nil {
		return err
	}

	// Prepare command based on type
	var cmd *exec.Cmd
	switch action.Type {
	case "sh":
		cmd = exec.Command("sh", action.Path)
	case "py":
		cmd = exec.Command("python3", action.Path)
	case "js":
		cmd = exec.Command("node", action.Path)
	default:
		// Try executing directly
		cmd = exec.Command(action.Path)
	}

	// Stdin handling
	if len(payload) > 0 {
		// If payload provided, pipe it
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		go func() {
			defer pipe.Close()
			pipe.Write(payload)
		}()
	} else {
		// Interactive if no payload? Spec 10.3: "attach stdio for interactive scripts unless explicitly disabled"
		// If payload provided, we used stdin for it. If not, inherit?
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	profile, err := LoadProfile(profileID)
	if err != nil {
		return err
	}

	if err := InjectEnvironment(cmd, profile); err != nil {
		return err
	}

	return cmd.Run()
}
