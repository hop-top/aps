package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// InjectEnvironment prepares environment variables for a command
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
	config, _ := LoadConfig()
	prefix := config.Prefix

	apsEnv := map[string]string{
		fmt.Sprintf("%s_PROFILE_ID", prefix):       profile.ID,
		fmt.Sprintf("%s_PROFILE_DIR", prefix):      profileDir,
		fmt.Sprintf("%s_PROFILE_YAML", prefix):     profileYaml,
		fmt.Sprintf("%s_PROFILE_SECRETS", prefix):  secretsPath,
		fmt.Sprintf("%s_PROFILE_DOCS_DIR", prefix): docsDir,
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
		// Actually, spec says "If ssh.key exists in profile directory... APS may inject"
		// The spec example shows: GIT_SSH_COMMAND=ssh -i <profile-dir>/ssh.key -F /dev/null
		// But T008 goal is generic environment. Let's stick to spec section 9 and 8.
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

// RunCommand executes a command within a profile's context using configured isolation
func RunCommand(profileID string, command string, args []string) error {
	profile, err := LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	// Check if requested isolation level is supported
	requestedLevel := profile.Isolation.Level
	if requestedLevel == "" {
		requestedLevel = IsolationProcess
	}

	switch requestedLevel {
	case IsolationProcess:
		// Process isolation is always available
		return runCommandWithProcessIsolation(profile, command, args)
	case IsolationPlatform:
		return fmt.Errorf("platform isolation not yet implemented")
	case IsolationContainer:
		return fmt.Errorf("container isolation not yet implemented")
	default:
		return fmt.Errorf("unsupported isolation level: %s", requestedLevel)
	}
}

// runCommandWithProcessIsolation executes a command using process-level isolation
func runCommandWithProcessIsolation(profile *Profile, command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := InjectEnvironment(cmd, profile); err != nil {
		return fmt.Errorf("failed to setup environment: %w", err)
	}

	return cmd.Run()
}

// RunAction executes a defined action using configured isolation
func RunAction(profileID string, actionID string, payload []byte) error {
	profile, err := LoadProfile(profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile %s: %w", profileID, err)
	}

	action, err := GetAction(profileID, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action %s: %w", actionID, err)
	}

	// Check if requested isolation level is supported
	requestedLevel := profile.Isolation.Level
	if requestedLevel == "" {
		requestedLevel = IsolationProcess
	}

	switch requestedLevel {
	case IsolationProcess:
		// Process isolation is always available
		return runActionWithProcessIsolation(profile, action, payload)
	case IsolationPlatform:
		return fmt.Errorf("platform isolation not yet implemented")
	case IsolationContainer:
		return fmt.Errorf("container isolation not yet implemented")
	default:
		return fmt.Errorf("unsupported isolation level: %s", requestedLevel)
	}
}

// runActionWithProcessIsolation executes an action using process-level isolation
func runActionWithProcessIsolation(profile *Profile, action *Action, payload []byte) error {
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
			return fmt.Errorf("failed to create stdin pipe: %w", err)
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

	if err := InjectEnvironment(cmd, profile); err != nil {
		return fmt.Errorf("failed to setup environment: %w", err)
	}

	return cmd.Run()
}
