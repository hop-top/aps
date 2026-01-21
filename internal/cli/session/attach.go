package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"oss-aps-cli/internal/core/session"
)

func NewAttachCmd() *cobra.Command {
	var mode string
	var latest bool

	cmd := &cobra.Command{
		Use:   "attach <session-id>",
		Short: "Attach to a running session",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry := session.GetRegistry()

			var sessionID string
			if latest {
				sessionID = getLatestSession(registry)
				if sessionID == "" {
					return fmt.Errorf("no active sessions found")
				}
			} else if len(args) == 0 {
				return fmt.Errorf("session ID required or use --latest flag")
			} else {
				sessionID = args[0]
			}

			sess, err := registry.Get(sessionID)
			if err != nil {
				return fmt.Errorf("session not found: %w", err)
			}

			if sess.Status != session.SessionActive {
				return fmt.Errorf("session %s is not active (status: %s)", sessionID, sess.Status)
			}

			return attachToSession(sess, mode)
		},
	}

	cmd.Flags().StringVarP(&mode, "mode", "m", "control", "Attachment mode (view|control)")
	cmd.Flags().BoolVarP(&latest, "latest", "l", false, "Attach to the most recent session")

	return cmd
}

func attachToSession(sess *session.SessionInfo, mode string) error {
	if sess.TmuxSocket == "" {
		return fmt.Errorf("session does not have a tmux socket")
	}

	platformType := sess.Environment["platform_type"]

	if platformType == "macos-darwin" || platformType == "linux-namespace" {
		return attachToPlatformSandbox(sess, mode, platformType)
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	fmt.Printf("Attaching to session %s\n", sess.ID)
	fmt.Printf("Press Ctrl+B then D to detach\n")

	var args []string
	args = append(args, "-S", sess.TmuxSocket)

	switch mode {
	case "view":
		args = append(args, "attach", "-r", "-t", sess.ID)
	case "control":
		args = append(args, "attach", "-t", sess.ID)
	default:
		return fmt.Errorf("invalid mode: %s (must be 'view' or 'control')", mode)
	}

	attachCmd := exec.Command(tmuxPath, args...)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr

	return attachCmd.Run()
}

func attachToPlatformSandbox(sess *session.SessionInfo, mode string, platformType string) error {
	sandboxUser, ok := sess.Environment["sandbox_user"]
	if !ok {
		return fmt.Errorf("sandbox user not found in session environment")
	}

	apsDir := os.Getenv("HOME")
	if apsDir == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

	keyPath := filepath.Join(apsDir, ".aps/keys/admin_priv")
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("admin private key not found at %s: %w", keyPath, err)
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	fmt.Printf("Attaching to platform sandbox session %s\n", sess.ID)
	fmt.Printf("Platform: %s, User: %s\n", platformType, sandboxUser)
	fmt.Printf("Press Ctrl+B then D to detach\n")

	var tmuxCmd string
	if mode == "view" {
		tmuxCmd = fmt.Sprintf("%s -S %s attach -r -t %s", tmuxPath, sess.TmuxSocket, sess.ID)
	} else {
		tmuxCmd = fmt.Sprintf("%s -S %s attach -t %s", tmuxPath, sess.TmuxSocket, sess.ID)
	}

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found: %w", err)
	}

	sshCmd := exec.Command(sshPath, "-i", keyPath, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@localhost", sandboxUser), tmuxCmd)

	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func getLatestSession(registry *session.SessionRegistry) string {
	sessions := registry.List()
	if len(sessions) == 0 {
		return ""
	}

	var activeSessions []*session.SessionInfo
	for _, s := range sessions {
		if s.Status == session.SessionActive {
			activeSessions = append(activeSessions, s)
		}
	}

	if len(activeSessions) == 0 {
		return ""
	}

	sort.Slice(activeSessions, func(i, j int) bool {
		return activeSessions[i].CreatedAt.After(activeSessions[j].CreatedAt)
	})

	return activeSessions[0].ID
}
