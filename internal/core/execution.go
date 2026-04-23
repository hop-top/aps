package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hop.top/aps/internal/core/bundle"
)

// buildEnvVars assembles the full environment slice (KEY=VALUE) for a profile.
// It is the canonical source of env injection; both InjectEnvironment and
// ProcessHandler.BuildEnv delegate here.
func buildEnvVars(profile *Profile) ([]string, error) {
	profileDir, err := GetProfileDir(profile.ID)
	if err != nil {
		return nil, err
	}
	profileYaml, err := GetProfilePath(profile.ID)
	if err != nil {
		return nil, err
	}
	secretsPath := filepath.Join(profileDir, "secrets.env")
	agentsDir, err := GetAgentsDir()
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to load secrets: %w", err)
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
		internalKeyPath := filepath.Join(profileDir, "ssh.key")
		if _, err := os.Stat(internalKeyPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -F /dev/null", internalKeyPath))
		}
	}

	// 6. Inject capability env vars (only for this profile's capabilities)
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
		return nil, fmt.Errorf("bundle resolution failed: %w", err)
	}
	for _, rb := range resolved {
		for k, v := range rb.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return env, nil
}

// InjectEnvironment prepares environment variables for a command.
// Kept for backward compatibility; delegates to buildEnvVars.
func InjectEnvironment(cmd *exec.Cmd, profile *Profile) error {
	env, err := buildEnvVars(profile)
	if err != nil {
		return err
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
		Email:       profile.Email,
		ConfigDir:   configDir,
		DataDir:     profileDir,
		Runtime:     "",                   // auto-detected by resolver
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

// actionBinaryArgs resolves the interpreter binary and argv for an action type.
func actionBinaryArgs(action *Action) (binary string, argv []string) {
	switch action.Type {
	case "sh":
		return "sh", []string{action.Path}
	case "py":
		return "python3", []string{action.Path}
	case "js":
		return "node", []string{action.Path}
	default:
		return action.Path, nil
	}
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
		return runCommandWithProcessIsolation(profile, command, args)
	case IsolationPlatform:
		return fmt.Errorf("platform isolation not yet implemented")
	case IsolationContainer:
		return fmt.Errorf("container isolation not yet implemented")
	default:
		return fmt.Errorf("unsupported isolation level: %s", requestedLevel)
	}
}

// runCommandWithProcessIsolation executes a command using process-level isolation.
// Builds env via buildEnvVars (fails fast on error); uses exec.Command for interactive stdio.
func runCommandWithProcessIsolation(profile *Profile, command string, args []string) error {
	env, err := buildEnvVars(profile)
	if err != nil {
		return fmt.Errorf("failed to setup environment: %w", err)
	}

	cmd := exec.CommandContext(context.Background(), command, args...) //nolint:gosec // command from profile config
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

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
		return runActionWithProcessIsolation(profile, action, payload)
	case IsolationPlatform:
		return fmt.Errorf("platform isolation not yet implemented")
	case IsolationContainer:
		return fmt.Errorf("container isolation not yet implemented")
	default:
		return fmt.Errorf("unsupported isolation level: %s", requestedLevel)
	}
}

// runActionWithProcessIsolation executes an action using process-level isolation.
// Builds env via buildEnvVars (fails fast on error); uses exec.Command for interactive stdio.
func runActionWithProcessIsolation(profile *Profile, action *Action, payload []byte) error {
	env, err := buildEnvVars(profile)
	if err != nil {
		return fmt.Errorf("failed to setup environment: %w", err)
	}

	binary, argv := actionBinaryArgs(action)

	cmd := exec.CommandContext(context.Background(), binary, argv...) //nolint:gosec // binary from action config
	cmd.Env = env

	// Stdin handling
	if len(payload) > 0 {
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}
		go func() {
			defer pipe.Close()
			_, _ = pipe.Write(payload)
		}()
	} else {
		cmd.Stdin = os.Stdin
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
