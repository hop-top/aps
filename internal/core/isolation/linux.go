//go:build linux

package isolation

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"oss-aps-cli/internal/core"
	"oss-aps-cli/internal/core/session"
)

const (
	sandboxUserPrefix = "aps-sandbox-"
	sharedWorkspace   = "/tmp/aps-shared"
)

type LinuxSandbox struct {
	context        *ExecutionContext
	username       string
	password       string
	groupname      string
	homeDir        string
	sharedDir      string
	namespaceID    string
	chrootPath     string
	cgroupPath     string
	tmuxSocket     string
	tmuxSession    string
	useTmux        bool
	sessionPID     int
	configured     bool
	adminPublicKey []byte
}

func NewLinuxSandbox() *LinuxSandbox {
	return &LinuxSandbox{}
}

func (l *LinuxSandbox) PrepareContext(profileID string) (*ExecutionContext, error) {
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

	l.username = sandboxUserPrefix + profileID
	l.groupname = sandboxUserPrefix + profileID
	l.homeDir = filepath.Join("/home", l.username)
	l.sharedDir = filepath.Join(sharedWorkspace, os.Getenv("USER"))
	l.namespaceID = fmt.Sprintf("aps-%s", profileID)
	l.chrootPath = filepath.Join(os.TempDir(), fmt.Sprintf("aps-chroot-%s", profileID))
	l.cgroupPath = filepath.Join("/sys/fs/cgroup", fmt.Sprintf("aps-%s", profileID))

	context := &ExecutionContext{
		ProfileID:   profileID,
		ProfileDir:  profileDir,
		ProfileYaml: profileYaml,
		SecretsPath: secretsPath,
		DocsDir:     docsDir,
		Environment: make(map[string]string),
		WorkingDir:  l.sharedDir,
	}

	l.context = context
	return context, nil
}

func (l *LinuxSandbox) SetupEnvironment(cmd interface{}) error {
	execCmd, ok := cmd.(*exec.Cmd)
	if !ok {
		return fmt.Errorf("cmd must be *exec.Cmd")
	}

	if l.context == nil {
		return fmt.Errorf("context not prepared")
	}

	if !l.configured {
		if err := l.configure(); err != nil {
			return fmt.Errorf("failed to configure sandbox: %w", err)
		}
		l.configured = true
	}

	env := os.Environ()

	config, _ := core.LoadConfig()
	prefix := config.Prefix

	apsEnv := map[string]string{
		fmt.Sprintf("%s_PROFILE_ID", prefix):       l.context.ProfileID,
		fmt.Sprintf("%s_PROFILE_DIR", prefix):      l.context.ProfileDir,
		fmt.Sprintf("%s_PROFILE_YAML", prefix):     l.context.ProfileYaml,
		fmt.Sprintf("%s_PROFILE_SECRETS", prefix):  l.context.SecretsPath,
		fmt.Sprintf("%s_PROFILE_DOCS_DIR", prefix): l.context.DocsDir,
	}

	for k, v := range apsEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	secrets, err := core.LoadSecrets(l.context.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	profile, err := core.LoadProfile(l.context.ProfileID)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	if profile.Git.Enabled {
		gitConfigPath := filepath.Join(l.context.ProfileDir, "gitconfig")
		if _, err := os.Stat(gitConfigPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_CONFIG_GLOBAL=%s", gitConfigPath))
		}
	}

	if profile.SSH.Enabled && profile.SSH.KeyPath != "" {
		internalKeyPath := filepath.Join(l.context.ProfileDir, "ssh.key")
		if _, err := os.Stat(internalKeyPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -F /dev/null", internalKeyPath))
		}
	}

	execCmd.Env = env
	execCmd.Dir = l.context.WorkingDir

	return nil
}

func (l *LinuxSandbox) Execute(command string, args []string) error {
	if !l.configured {
		if err := l.configure(); err != nil {
			return fmt.Errorf("failed to configure sandbox: %w", err)
		}
		l.configured = true
	}

	l.tmuxSocket = filepath.Join(os.TempDir(), fmt.Sprintf("aps-tmux-%s-socket", l.context.ProfileID))
	l.tmuxSession = fmt.Sprintf("aps-%s-%d", l.context.ProfileID, time.Now().Unix())
	l.useTmux = true

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := l.SetupEnvironment(cmd); err != nil {
		return err
	}

	wrappedCommand := strings.Join(append([]string{command}, args...), " ")
	sudoCmd := exec.Command("sudo", "-u", l.username, "-H", "bash", "-l", "-c", wrappedCommand)
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr

	if err := sudoCmd.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	l.sessionPID = sudoCmd.Process.Pid

	if err := l.registerSession(command, args); err != nil {
		sudoCmd.Process.Kill()
		return fmt.Errorf("failed to register session: %w", err)
	}

	if err := sudoCmd.Wait(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (l *LinuxSandbox) registerSession(command string, args []string) error {
	fullCommand := strings.Join(append([]string{command}, args...), " ")

	registry := session.GetRegistry()
	sess := &session.SessionInfo{
		ID:         l.tmuxSession,
		ProfileID:  l.context.ProfileID,
		ProfileDir: l.context.ProfileDir,
		Command:    fmt.Sprintf("sudo -u %s %s", l.username, fullCommand),
		PID:        l.sessionPID,
		Status:     session.SessionActive,
		Tier:       session.TierStandard,
		TmuxSocket: l.tmuxSocket,
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		Environment: map[string]string{
			"sandbox_user":  l.username,
			"sandbox_group": l.groupname,
			"sandbox_home":  l.homeDir,
			"shared_dir":    l.sharedDir,
			"namespace_id":  l.namespaceID,
			"chroot_path":   l.chrootPath,
			"cgroup_path":   l.cgroupPath,
			"tmux_socket":   l.tmuxSocket,
			"tmux_session":  l.tmuxSession,
			"platform_type": "linux",
		},
	}

	return registry.Register(sess)
}

func (l *LinuxSandbox) ExecuteAction(actionID string, payload []byte) error {
	action, err := core.GetAction(l.context.ProfileID, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}

	if !l.configured {
		if err := l.configure(); err != nil {
			return fmt.Errorf("failed to configure sandbox: %w", err)
		}
		l.configured = true
	}

	var cmd *exec.Cmd
	switch action.Type {
	case "sh":
		cmd = exec.Command("bash", action.Path)
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

	if err := l.SetupEnvironment(cmd); err != nil {
		return err
	}

	sudoArgs := append([]string{"-u", l.username, cmd.Path}, cmd.Args[1:]...)
	sudoCmd := exec.Command("sudo", sudoArgs...)
	sudoCmd.Stdin = cmd.Stdin
	sudoCmd.Stdout = cmd.Stdout
	sudoCmd.Stderr = cmd.Stderr

	if err := sudoCmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (l *LinuxSandbox) Cleanup() error {
	if l.useTmux {
		l.cleanupTmux()
	}

	registry := session.GetRegistry()
	_ = registry.Unregister(l.tmuxSession)

	l.context = nil
	return nil
}

func (l *LinuxSandbox) cleanupTmux() {
	if l.tmuxSocket == "" || l.tmuxSession == "" {
		return
	}

	cmd := exec.Command("tmux", "-S", l.tmuxSocket, "kill-session", "-t", l.tmuxSession)
	_ = cmd.Run()
}

func (l *LinuxSandbox) Validate() error {
	if l.context == nil {
		return fmt.Errorf("context not prepared")
	}

	if _, err := os.Stat(l.context.ProfileDir); os.IsNotExist(err) {
		return fmt.Errorf("profile directory does not exist: %s", l.context.ProfileDir)
	}

	if _, err := os.Stat(l.context.ProfileYaml); os.IsNotExist(err) {
		return fmt.Errorf("profile.yaml does not exist: %s", l.context.ProfileYaml)
	}

	if _, err := exec.LookPath("unshare"); err != nil {
		return fmt.Errorf("unshare not available: %w", err)
	}

	if _, err := exec.LookPath("setfacl"); err != nil {
		return fmt.Errorf("setfacl not available: %w", err)
	}

	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("sudo not available: %w", err)
	}

	return nil
}

func (l *LinuxSandbox) IsAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	if _, err := exec.LookPath("unshare"); err != nil {
		return false
	}

	if _, err := exec.LookPath("setfacl"); err != nil {
		return false
	}

	if _, err := exec.LookPath("sudo"); err != nil {
		return false
	}

	return true
}

func (l *LinuxSandbox) configure() error {
	if err := l.createSandboxUser(); err != nil {
		return fmt.Errorf("failed to create sandbox user: %w", err)
	}

	if err := l.configureSharedWorkspace(); err != nil {
		return fmt.Errorf("failed to configure shared workspace: %w", err)
	}

	if err := l.configurePasswordlessSudo(); err != nil {
		return fmt.Errorf("failed to configure passwordless sudo: %w", err)
	}

	if err := l.distributeSSHKeys(); err != nil {
		return fmt.Errorf("failed to distribute SSH keys: %w", err)
	}

	return nil
}

func (l *LinuxSandbox) createSandboxUser() error {
	if l.userExists(l.username) {
		return nil
	}

	password, err := l.generateRandomPassword()
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}
	l.password = password

	useraddCmd := exec.Command("sudo", "useradd", "-m", "-s", "/bin/bash", l.username)
	if output, err := useraddCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create user: %w\nOutput: %s", err, string(output))
	}

	passwdCmd := exec.Command("sudo", "chpasswd")
	passwdCmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", l.username, l.password))
	if output, err := passwdCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set password: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (l *LinuxSandbox) userExists(username string) bool {
	cmd := exec.Command("id", username)
	err := cmd.Run()
	return err == nil
}

func (l *LinuxSandbox) generateRandomPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (l *LinuxSandbox) configureSharedWorkspace() error {
	if err := os.MkdirAll(l.sharedDir, 0770); err != nil {
		return fmt.Errorf("failed to create shared workspace: %w", err)
	}

	currentUser := os.Getenv("USER")
	chownCmd := exec.Command("sudo", "chown", "-R", fmt.Sprintf("%s:%s", currentUser, l.username), l.sharedDir)
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set workspace ownership: %w\nOutput: %s", err, string(output))
	}

	chmodCmd := exec.Command("sudo", "chmod", "-R", "0770", l.sharedDir)
	if output, err := chmodCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set workspace permissions: %w\nOutput: %s", err, string(output))
	}

	aclRule := fmt.Sprintf("u:%s:rwX", l.username)
	setfaclCmd := exec.Command("sudo", "setfacl", "-R", "-m", aclRule, l.sharedDir)
	if output, err := setfaclCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set ACL: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (l *LinuxSandbox) configurePasswordlessSudo() error {
	currentUser := os.Getenv("USER")
	sudoersContent := fmt.Sprintf(`# Allow %s to sudo to %s without password
%s ALL=(%s) NOPASSWD: ALL
`, currentUser, l.username, currentUser, l.username)

	sudoersFile := fmt.Sprintf("/etc/sudoers.d/50-nopasswd-for-%s", l.username)

	if _, err := os.Stat(sudoersFile); err == nil {
		return nil
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("sudoers-%s", l.username))
	if err := os.WriteFile(tmpFile, []byte(sudoersContent), 0440); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}
	defer os.Remove(tmpFile)

	cpCmd := exec.Command("sudo", "cp", tmpFile, sudoersFile)
	if output, err := cpCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install sudoers file: %w\nOutput: %s", err, string(output))
	}

	chmodCmd := exec.Command("sudo", "chmod", "0440", sudoersFile)
	if output, err := chmodCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %w\nOutput: %s", err, string(output))
	}

	validateCmd := exec.Command("sudo", "visudo", "-c", sudoersFile)
	if output, err := validateCmd.CombinedOutput(); err != nil {
		exec.Command("sudo", "rm", "-f", sudoersFile).Run()
		return fmt.Errorf("sudoers validation failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (l *LinuxSandbox) distributeSSHKeys() error {
	homeDir := l.homeDir
	sshDir := filepath.Join(homeDir, ".ssh")

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	chownCmd := exec.Command("sudo", "chown", "-R", l.username, sshDir)
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set .ssh ownership: %w\nOutput: %s", err, string(output))
	}

	adminKeysDir := filepath.Join(os.Getenv("HOME"), ".aps/keys")
	adminPubKeyPath := filepath.Join(adminKeysDir, "admin_pub")

	adminPubKey, err := os.ReadFile(adminPubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read admin public key: %w", err)
	}

	authorizedKeysPath := filepath.Join(sshDir, "authorized_keys")

	if _, err := os.Stat(authorizedKeysPath); os.IsNotExist(err) {
		touchCmd := exec.Command("sudo", "touch", authorizedKeysPath)
		if err := touchCmd.Run(); err != nil {
			return fmt.Errorf("failed to create authorized_keys: %w", err)
		}
	}

	authorizedKeys, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	if !bytes.Contains(authorizedKeys, adminPubKey) {
		appendCmd := exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo >> %s", authorizedKeysPath))
		pipe, err := appendCmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create pipe: %w", err)
		}

		if _, err := pipe.Write(adminPubKey); err != nil {
			pipe.Close()
			return fmt.Errorf("failed to write admin key: %w", err)
		}
		pipe.Close()

		if output, err := appendCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to append admin key to authorized_keys: %w\nOutput: %s", err, string(output))
		}
	}

	chownCmd2 := exec.Command("sudo", "chown", l.username, authorizedKeysPath)
	if output, err := chownCmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set authorized_keys ownership: %w\nOutput: %s", err, string(output))
	}

	chmodCmd2 := exec.Command("sudo", "chmod", "0600", authorizedKeysPath)
	if output, err := chmodCmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set authorized_keys permissions: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (l *LinuxSandbox) setupUserNamespace() error {
	if _, err := exec.LookPath("unshare"); err != nil {
		return fmt.Errorf("unshare not available: %w", err)
	}

	return nil
}

func (l *LinuxSandbox) createChrootEnv() error {
	if _, err := os.Stat(l.chrootPath); os.IsNotExist(err) {
		dirs := []string{"bin", "lib", "etc", "home", "tmp", "dev", "proc", "sys"}
		for _, dir := range dirs {
			if err := os.MkdirAll(filepath.Join(l.chrootPath, dir), 0755); err != nil {
				return fmt.Errorf("failed to create chroot directory %s: %w", dir, err)
			}
		}

		mountCommands := []struct {
			source string
			target string
			opts   []string
		}{
			{"/bin", filepath.Join(l.chrootPath, "bin"), []string{"--bind"}},
			{"/lib", filepath.Join(l.chrootPath, "lib"), []string{"--bind"}},
			{"/lib64", filepath.Join(l.chrootPath, "lib64"), []string{"--bind"}},
			{"/etc", filepath.Join(l.chrootPath, "etc"), []string{"--bind", "--read-only"}},
			{"/dev", filepath.Join(l.chrootPath, "dev"), []string{"--bind"}},
			{"proc", filepath.Join(l.chrootPath, "proc"), []string{"-t", "proc"}},
			{"sys", filepath.Join(l.chrootPath, "sys"), []string{"-t", "sysfs"}},
		}

		for _, mc := range mountCommands {
			mountArgs := append(mc.opts, mc.source, mc.target)
			mountCmd := exec.Command("sudo", append([]string{"mount"}, mountArgs...)...)
			if output, err := mountCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to mount %s to %s: %w\nOutput: %s", mc.source, mc.target, err, string(output))
			}
		}
	}

	return nil
}

func (l *LinuxSandbox) setupCgroups() error {
	if _, err := os.Stat("/sys/fs/cgroup"); os.IsNotExist(err) {
		return nil
	}

	if err := os.MkdirAll(l.cgroupPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	cgroupTasks := filepath.Join(l.cgroupPath, "cgroup.procs")
	if _, err := os.Stat(cgroupTasks); err == nil {
		return nil
	}

	return nil
}

func (l *LinuxSandbox) configureACLs() error {
	if err := l.configureSharedWorkspace(); err != nil {
		return err
	}

	return nil
}
