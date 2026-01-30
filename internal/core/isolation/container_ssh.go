package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"oss-aps-cli/internal/core/session"
)

func ConfigureContainerSSH(engine ContainerEngine, containerID, profileID string) error {
	adminKeysDir := filepath.Join(os.Getenv("HOME"), ".aps/keys")
	adminPubKeyPath := filepath.Join(adminKeysDir, "admin_pub")

	adminPubKey, err := os.ReadFile(adminPubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read admin public key: %w", err)
	}

	createSSHDirCmd := exec.Command("docker", "exec", containerID, "mkdir", "-p", "/home/appuser/.ssh")
	if err := createSSHDirCmd.Run(); err != nil {
		return fmt.Errorf("failed to create .ssh directory in container: %w", err)
	}

	authorizedKeysPath := "/home/appuser/.ssh/authorized_keys"
	writeKeyCmd := exec.Command("docker", "exec", "-i", containerID, "sh", "-c",
		fmt.Sprintf("echo '%s' > %s", string(adminPubKey), authorizedKeysPath))
	if err := writeKeyCmd.Run(); err != nil {
		return fmt.Errorf("failed to write authorized_keys: %w", err)
	}

	chmodCmd := exec.Command("docker", "exec", containerID, "chmod", "700", "/home/appuser/.ssh")
	if err := chmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to set .ssh permissions: %w", err)
	}

	authorizedKeysChmodCmd := exec.Command("docker", "exec", containerID, "chmod", "600", authorizedKeysPath)
	if err := authorizedKeysChmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to set authorized_keys permissions: %w", err)
	}

	chownCmd := exec.Command("docker", "exec", containerID, "chown", "-R", "appuser:appuser", "/home/appuser/.ssh")
	if err := chownCmd.Run(); err != nil {
		return fmt.Errorf("failed to set .ssh ownership: %w", err)
	}

	return nil
}

func AttachToContainer(engine ContainerEngine, session *session.SessionInfo, mode string) error {
	keyPath := filepath.Join(os.Getenv("HOME"), ".aps/keys/admin_key")

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("admin private key not found at %s", keyPath)
	}

	containerIP, err := engine.GetContainerIP(session.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %w", err)
	}

	if containerIP == "" {
		port, err := engine.GetContainerPortMapping(session.ContainerID, "22")
		if err != nil || port == "" {
			return fmt.Errorf("container not accessible: no IP or port mapping found")
		}

		if session.Environment != nil {
			if sshPort, ok := session.Environment["ssh_port"]; ok {
				port = sshPort
			}
		}

		containerIP = "localhost"
	}

	var tmuxCmd string
	if mode == "view" {
		tmuxCmd = fmt.Sprintf("tmux attach -t %s -r", session.TmuxSession)
	} else {
		tmuxCmd = fmt.Sprintf("tmux attach -t %s", session.TmuxSession)
	}

	sshArgs := []string{
		"-i", keyPath,
		"-p", "22",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("appuser@%s", containerIP),
		tmuxCmd,
	}

	sshCmd := exec.Command("ssh", sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func GetContainerSSHConfig(containerID, username string) (string, int, error) {
	cmd := exec.Command("docker", "port", containerID, "22")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get SSH port mapping: %w\nOutput: %s", err, string(output))
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid port mapping format: %s", string(output))
	}

	host := parts[0]
	portStr := parts[1]

	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse port: %w", err)
	}

	return host, port, nil
}

func VerifySSHConnection(containerID, username string) error {
	keyPath := filepath.Join(os.Getenv("HOME"), ".aps/keys/admin_key")

	host, port, err := GetContainerSSHConfig(containerID, username)
	if err != nil {
		return err
	}

	sshArgs := []string{
		"-i", keyPath,
		"-p", fmt.Sprintf("%d", port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=5",
		"-o", "BatchMode=yes",
		fmt.Sprintf("%s@%s", username, host),
		"echo", "connected",
	}

	cmd := exec.Command("ssh", sshArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w\nOutput: %s", err, string(output))
	}

	if strings.TrimSpace(string(output)) != "connected" {
		return fmt.Errorf("unexpected SSH output: %s", string(output))
	}

	return nil
}
