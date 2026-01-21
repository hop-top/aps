//go:build darwin

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
	sandboxUserPrefix = "aps-"
	sharedWorkspace   = "/Users/Shared/aps"
)

type DarwinSandbox struct {
	context        *ExecutionContext
	username       string
	password       string
	homeDir        string
	sharedDir      string
	tmuxSocket     string
	tmuxSession    string
	useTmux        bool
	sessionPID     int
	configured     bool
	adminPublicKey []byte
}

func NewDarwinSandbox() *DarwinSandbox {
	return &DarwinSandbox{}
}

func (d *DarwinSandbox) PrepareContext(profileID string) (*ExecutionContext, error) {
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

	d.username = sandboxUserPrefix + profileID
	d.homeDir = "/Users/" + d.username
	d.sharedDir = filepath.Join(sharedWorkspace, os.Getenv("USER"))

	context := &ExecutionContext{
		ProfileID:   profileID,
		ProfileDir:  profileDir,
		ProfileYaml: profileYaml,
		SecretsPath: secretsPath,
		DocsDir:     docsDir,
		Environment: make(map[string]string),
		WorkingDir:  d.sharedDir,
	}

	d.context = context
	return context, nil
}

func (d *DarwinSandbox) SetupEnvironment(cmd interface{}) error {
	execCmd, ok := cmd.(*exec.Cmd)
	if !ok {
		return fmt.Errorf("cmd must be *exec.Cmd")
	}

	if d.context == nil {
		return fmt.Errorf("context not prepared")
	}

	if !d.configured {
		if err := d.configure(); err != nil {
			return fmt.Errorf("failed to configure sandbox: %w", err)
		}
		d.configured = true
	}

	env := os.Environ()

	config, _ := core.LoadConfig()
	prefix := config.Prefix

	apsEnv := map[string]string{
		fmt.Sprintf("%s_PROFILE_ID", prefix):       d.context.ProfileID,
		fmt.Sprintf("%s_PROFILE_DIR", prefix):      d.context.ProfileDir,
		fmt.Sprintf("%s_PROFILE_YAML", prefix):     d.context.ProfileYaml,
		fmt.Sprintf("%s_PROFILE_SECRETS", prefix):  d.context.SecretsPath,
		fmt.Sprintf("%s_PROFILE_DOCS_DIR", prefix): d.context.DocsDir,
	}

	for k, v := range apsEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	secrets, err := core.LoadSecrets(d.context.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	profile, err := core.LoadProfile(d.context.ProfileID)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}

	if profile.Git.Enabled {
		gitConfigPath := filepath.Join(d.context.ProfileDir, "gitconfig")
		if _, err := os.Stat(gitConfigPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_CONFIG_GLOBAL=%s", gitConfigPath))
		}
	}

	if profile.SSH.Enabled && profile.SSH.KeyPath != "" {
		internalKeyPath := filepath.Join(d.context.ProfileDir, "ssh.key")
		if _, err := os.Stat(internalKeyPath); err == nil {
			env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -F /dev/null", internalKeyPath))
		}
	}

	execCmd.Env = env
	execCmd.Dir = d.context.WorkingDir

	return nil
}

func (d *DarwinSandbox) Execute(command string, args []string) error {
	if !d.configured {
		if err := d.configure(); err != nil {
			return fmt.Errorf("failed to configure sandbox: %w", err)
		}
		d.configured = true
	}

	d.tmuxSocket = filepath.Join(os.TempDir(), fmt.Sprintf("aps-tmux-%s-socket", d.context.ProfileID))
	d.tmuxSession = fmt.Sprintf("aps-%s-%d", d.context.ProfileID, time.Now().Unix())
	d.useTmux = true

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := d.SetupEnvironment(cmd); err != nil {
		return err
	}

	wrappedCommand := strings.Join(append([]string{command}, args...), " ")
	sudoCmd := exec.Command("sudo", "-u", d.username, "-H", "zsh", "-l", "-c", wrappedCommand)
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr

	if err := sudoCmd.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	d.sessionPID = sudoCmd.Process.Pid

	if err := d.registerSession(command, args); err != nil {
		sudoCmd.Process.Kill()
		return fmt.Errorf("failed to register session: %w", err)
	}

	if err := sudoCmd.Wait(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (d *DarwinSandbox) registerSession(command string, args []string) error {
	fullCommand := strings.Join(append([]string{command}, args...), " ")

	registry := session.GetRegistry()
	sess := &session.SessionInfo{
		ID:         d.tmuxSession,
		ProfileID:  d.context.ProfileID,
		ProfileDir: d.context.ProfileDir,
		Command:    fmt.Sprintf("sudo -u %s %s", d.username, fullCommand),
		PID:        d.sessionPID,
		Status:     session.SessionActive,
		Tier:       session.TierStandard,
		TmuxSocket: d.tmuxSocket,
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
		Environment: map[string]string{
			"sandbox_user":  d.username,
			"sandbox_home":  d.homeDir,
			"shared_dir":    d.sharedDir,
			"tmux_socket":   d.tmuxSocket,
			"tmux_session":  d.tmuxSession,
			"platform_type": "macos-darwin",
		},
	}

	return registry.Register(sess)
}

func (d *DarwinSandbox) ExecuteAction(actionID string, payload []byte) error {
	action, err := core.GetAction(d.context.ProfileID, actionID)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}

	if !d.configured {
		if err := d.configure(); err != nil {
			return fmt.Errorf("failed to configure sandbox: %w", err)
		}
		d.configured = true
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

	if err := d.SetupEnvironment(cmd); err != nil {
		return err
	}

	sudoArgs := append([]string{"-u", d.username, cmd.Path}, cmd.Args[1:]...)
	sudoCmd := exec.Command("sudo", sudoArgs...)
	sudoCmd.Stdin = cmd.Stdin
	sudoCmd.Stdout = cmd.Stdout
	sudoCmd.Stderr = cmd.Stderr

	if err := sudoCmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutionFailed, err)
	}

	return nil
}

func (d *DarwinSandbox) Cleanup() error {
	if d.useTmux {
		d.cleanupTmux()
	}

	registry := session.GetRegistry()
	_ = registry.Unregister(d.tmuxSession)

	d.context = nil
	return nil
}

func (d *DarwinSandbox) cleanupTmux() {
	if d.tmuxSocket == "" || d.tmuxSession == "" {
		return
	}

	cmd := exec.Command("tmux", "-S", d.tmuxSocket, "kill-session", "-t", d.tmuxSession)
	_ = cmd.Run()
}

func (d *DarwinSandbox) Validate() error {
	if d.context == nil {
		return fmt.Errorf("context not prepared")
	}

	if _, err := os.Stat(d.context.ProfileDir); os.IsNotExist(err) {
		return fmt.Errorf("profile directory does not exist: %s", d.context.ProfileDir)
	}

	if _, err := os.Stat(d.context.ProfileYaml); os.IsNotExist(err) {
		return fmt.Errorf("profile.yaml does not exist: %s", d.context.ProfileYaml)
	}

	if _, err := exec.LookPath("dscl"); err != nil {
		return fmt.Errorf("dscl not available: %w", err)
	}

	if _, err := exec.LookPath("launchctl"); err != nil {
		return fmt.Errorf("launchctl not available: %w", err)
	}

	return nil
}

func (d *DarwinSandbox) IsAvailable() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	if _, err := exec.LookPath("dscl"); err != nil {
		return false
	}

	if _, err := exec.LookPath("launchctl"); err != nil {
		return false
	}

	return true
}

func (d *DarwinSandbox) configure() error {
	if err := d.createSandboxUser(); err != nil {
		return fmt.Errorf("failed to create sandbox user: %w", err)
	}

	if err := d.configureSharedWorkspace(); err != nil {
		return fmt.Errorf("failed to configure shared workspace: %w", err)
	}

	if err := d.configurePasswordlessSudo(); err != nil {
		return fmt.Errorf("failed to configure passwordless sudo: %w", err)
	}

	if err := d.distributeSSHKeys(); err != nil {
		return fmt.Errorf("failed to distribute SSH keys: %w", err)
	}

	return nil
}

func (d *DarwinSandbox) createSandboxUser() error {
	if d.userExists(d.username) {
		return nil
	}

	nextUID, err := d.findNextAvailableUID()
	if err != nil {
		return fmt.Errorf("failed to find available UID: %w", err)
	}

	d.password, err = d.generateRandomPassword()
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	userPath := "/Users/" + d.username

	cmds := []struct {
		name string
		args []string
	}{
		{"dscl", []string{".", "-create", userPath}},
		{"dscl", []string{".", "-create", userPath, "UniqueID", fmt.Sprintf("%d", nextUID)}},
		{"dscl", []string{".", "-create", userPath, "PrimaryGroupID", fmt.Sprintf("%d", nextUID)}},
		{"dscl", []string{".", "-create", userPath, "RealName", fmt.Sprintf("APS Sandbox %s", d.context.ProfileID)}},
		{"dscl", []string{".", "-create", userPath, "UserShell", "/bin/zsh"}},
		{"dscl", []string{".", "-create", userPath, "NFSHomeDirectory", userPath}},
	}

	for _, cmd := range cmds {
		sudoCmd := exec.Command("sudo", append([]string{cmd.name}, cmd.args...)...)
		if output, err := sudoCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to run %s %v: %w\nOutput: %s", cmd.name, cmd.args, err, string(output))
		}
	}

	passwdCmd := exec.Command("sudo", "dscl", ".", "-passwd", userPath, d.password)
	if output, err := passwdCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set password: %w\nOutput: %s", err, string(output))
	}

	hideCmd := exec.Command("sudo", "dscl", ".", "-create", userPath, "IsHidden", "1")
	if output, err := hideCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to hide user: %w\nOutput: %s", err, string(output))
	}

	if err := os.MkdirAll(userPath, 0755); err != nil {
		return fmt.Errorf("failed to create home directory: %w", err)
	}

	chownCmd := exec.Command("sudo", "chown", "-R", fmt.Sprintf("%s:%s", d.username, d.username), userPath)
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set home directory ownership: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (d *DarwinSandbox) userExists(username string) bool {
	cmd := exec.Command("dscl", ".", "-read", "/Users/"+username)
	err := cmd.Run()
	return err == nil
}

func (d *DarwinSandbox) findNextAvailableUID() (int, error) {
	cmd := exec.Command("dscl", ".", "-list", "/Users", "UniqueID")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	maxUID := 500
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			var uid int
			if _, err := fmt.Sscanf(parts[1], "%d", &uid); err == nil {
				if uid > maxUID {
					maxUID = uid
				}
			}
		}
	}

	return maxUID + 1, nil
}

func (d *DarwinSandbox) generateRandomPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (d *DarwinSandbox) configureSharedWorkspace() error {
	if err := os.MkdirAll(d.sharedDir, 0770); err != nil {
		return fmt.Errorf("failed to create shared workspace: %w", err)
	}

	currentUser := os.Getenv("USER")
	chownCmd := exec.Command("sudo", "chown", "-R", fmt.Sprintf("%s:%s", currentUser, d.username), d.sharedDir)
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set workspace ownership: %w\nOutput: %s", err, string(output))
	}

	chmodCmd := exec.Command("sudo", "chmod", "0770", d.sharedDir)
	if output, err := chmodCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set workspace permissions: %w\nOutput: %s", err, string(output))
	}

	aclRule := fmt.Sprintf("group:%s allow read,write,append,delete,delete_child,readattr,writeattr,readextattr,writeextattr,readsecurity,writesecurity,chown,file_inherit,directory_inherit", d.username)
	aclCmd := exec.Command("sudo", "chmod", "-h", "+a", aclRule, d.sharedDir)
	if output, err := aclCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set ACL: %w\nOutput: %s", err, string(output))
	}

	findCmd := exec.Command("find", d.sharedDir, "-print0")
	xargsCmd := exec.Command("xargs", "-0", "sudo", "chmod", "-h", "+a", aclRule)

	findOutput, err := findCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	xargsCmd.Stdin = findOutput

	if err := findCmd.Start(); err != nil {
		return fmt.Errorf("failed to start find: %w", err)
	}

	if output, err := xargsCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to apply ACLs to files: %w\nOutput: %s", err, string(output))
	}

	if err := findCmd.Wait(); err != nil {
		return fmt.Errorf("find command failed: %w", err)
	}

	return nil
}

func (d *DarwinSandbox) configurePasswordlessSudo() error {
	currentUser := os.Getenv("USER")
	sudoersContent := fmt.Sprintf(`# Allow %s to sudo to %s without password
%s ALL=(%s) NOPASSWD: ALL
`, currentUser, d.username, currentUser, d.username)

	sudoersFile := fmt.Sprintf("/etc/sudoers.d/50-nopasswd-for-%s", d.username)

	if _, err := os.Stat(sudoersFile); err == nil {
		return nil
	}

	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("sudoers-%s", d.username))
	if err := os.WriteFile(tmpFile, []byte(sudoersContent), 0440); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}
	defer os.Remove(tmpFile)

	visudoCmd := exec.Command("sudo", "cp", tmpFile, sudoersFile)
	if output, err := visudoCmd.CombinedOutput(); err != nil {
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

func (d *DarwinSandbox) distributeSSHKeys() error {
	homeDir := "/Users/" + d.username
	sshDir := filepath.Join(homeDir, ".ssh")

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	chownCmd := exec.Command("sudo", "chown", "-R", d.username, sshDir)
	if output, err := chownCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set .ssh ownership: %w\nOutput: %s", err, string(output))
	}

	adminKeysDir := filepath.Join(os.Getenv("HOME"), ".aps", "keys")
	adminPubKeyPath := filepath.Join(adminKeysDir, "admin_pub")

	adminPubKey, err := os.ReadFile(adminPubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read admin public key: %w", err)
	}

	authorizedKeysPath := filepath.Join(sshDir, "authorized_keys")

	if _, err := os.Stat(authorizedKeysPath); os.IsNotExist(err) {
		chmodCmd := exec.Command("sudo", "touch", authorizedKeysPath)
		if err := chmodCmd.Run(); err != nil {
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

	chownCmd2 := exec.Command("sudo", "chown", d.username, authorizedKeysPath)
	if output, err := chownCmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set authorized_keys ownership: %w\nOutput: %s", err, string(output))
	}

	chmodCmd2 := exec.Command("sudo", "chmod", "0600", authorizedKeysPath)
	if output, err := chmodCmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set authorized_keys permissions: %w\nOutput: %s", err, string(output))
	}

	return nil
}
