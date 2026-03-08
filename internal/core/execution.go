package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hop.top/aps/internal/core/bundle"
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

	// 6. Inject capability env vars (only for this profile's capabilities)
	// Uses builtinCaps set to skip built-ins; resolves external cap paths
	// from ~/.aps/capabilities/ and configured sources.
	builtinCaps := map[string]bool{
		"a2a": true, "agent-protocol": true, "webhooks": true,
	}
	capRoots := []string{}
	if dataDir, e := GetDataDir(); e == nil {
		capRoots = append(capRoots, filepath.Join(dataDir, "capabilities"))
	}
	for _, src := range config.CapabilitySources {
		if src != "" {
			capRoots = append(capRoots, src)
		}
	}
	for _, capName := range profile.Capabilities {
		if builtinCaps[capName] {
			continue
		}
		for _, root := range capRoots {
			p := filepath.Join(root, capName)
			if info, e := os.Stat(p); e == nil && info.IsDir() {
				safeName := strings.ToUpper(strings.ReplaceAll(capName, "-", "_"))
				env = append(env, fmt.Sprintf("APS_%s_PATH=%s", safeName, p))
				break
			}
		}
	}

	// 7. Resolve bundles and inject bundle env vars (T-0052)
	resolved, err := ResolveBundlesForProfile(profile)
	if err != nil {
		return fmt.Errorf("bundle resolution failed: %w", err)
	}
	for _, rb := range resolved {
		for k, v := range rb.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmd.Env = env
	return nil
}

// ResolveBundlesForProfile resolves all bundles declared in the profile's capabilities list.
// It returns the resolved bundles, logs warnings, and returns an error if any bundle has resolution errors.
func ResolveBundlesForProfile(profile *Profile) ([]*bundle.ResolvedBundle, error) {
	bundleNames, _ := ExtractBundleNames(profile.Capabilities)
	if len(bundleNames) == 0 {
		return nil, nil
	}

	reg, err := bundle.NewRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to load bundle registry: %w", err)
	}

	profileDir, _ := GetProfileDir(profile.ID)
	configDir, _ := os.UserConfigDir()
	if configDir == "" {
		configDir = profileDir
	}

	ctx := bundle.ProfileContext{
		ID:          profile.ID,
		DisplayName: profile.DisplayName,
		Email:       "", // Profile has no Email field
		ConfigDir:   configDir,
		DataDir:     profileDir,
		Runtime:     "",            // auto-detected by resolver
		Scope:       bundle.BundleScope{}, // populated below if profile has scope
	}

	// Seed ctx.Scope from the profile's own ScopeConfig (T-0053 bridge).
	if profile.Scope != nil {
		ctx.Scope = bundle.BundleScope{
			Operations:   profile.Scope.Operations,
			FilePatterns: profile.Scope.FilePatterns,
			Networks:     profile.Scope.Networks,
		}
	}

	var resolved []*bundle.ResolvedBundle
	for _, name := range bundleNames {
		b, err := reg.Get(name)
		if err != nil {
			return nil, fmt.Errorf("bundle %q not found: %w", name, err)
		}
		rb, err := bundle.Resolve(*b, reg, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve bundle %q: %w", name, err)
		}
		// Surface errors collected during resolution.
		if len(rb.Errors) > 0 {
			var msgs []string
			for _, e := range rb.Errors {
				msgs = append(msgs, e.Error())
			}
			return nil, fmt.Errorf("bundle %q: %s", name, strings.Join(msgs, "; "))
		}
		// Log warnings.
		for _, w := range rb.Warnings {
			log.Printf("bundle %q warning: %s", name, w)
		}
		// Log always-services (actual startup is out of scope for now).
		for _, svc := range rb.AlwaysServices {
			log.Printf("bundle %q: would start always-service %q (adapter=%s)", name, svc.Name, svc.Adapter)
		}
		resolved = append(resolved, rb)
	}
	return resolved, nil
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
