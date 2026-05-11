package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

var apsBinary string

func TestMain(m *testing.M) {
	if err := compileBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile aps binary: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.Remove(apsBinary)
	os.Exit(code)
}

func compileBinary() error {
	binName := "aps-chat-test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	apsBinary = filepath.Join(os.TempDir(), binName)
	rootDir, err := filepath.Abs("../../..")
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", apsBinary, "./cmd/aps")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func prepareAPS(t *testing.T, homeDir string, extraEnv map[string]string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(apsBinary, args...)
	overridden := map[string]bool{
		"HOME":          true,
		"USERPROFILE":   true,
		"XDG_DATA_HOME": true,
		"APS_DATA_PATH": true,
	}
	env := []string{
		"HOME=" + homeDir,
		"USERPROFILE=" + homeDir,
		"XDG_DATA_HOME=" + filepath.Join(homeDir, ".local", "share"),
	}
	for k, v := range extraEnv {
		overridden[k] = true
		env = append(env, k+"="+v)
	}
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if !overridden[key] {
			env = append(env, e)
		}
	}
	cmd.Env = env
	return cmd
}

func runAPS(t *testing.T, homeDir string, extraEnv map[string]string, args ...string) (string, string, error) {
	t.Helper()
	cmd := prepareAPS(t, homeDir, extraEnv, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func createProfile(t *testing.T, home, id string) {
	t.Helper()
	stdout, stderr, err := runAPS(t, home, nil, "profile", "create", id)
	if err != nil {
		t.Fatalf("profile create failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func fakeEnv(response string) map[string]string {
	return map[string]string{
		"APS_CHAT_FAKE_RESPONSE": response,
		"APS_CHAT_TUI_TEST":      "1",
	}
}

func TestChatOnce_PersistsSession(t *testing.T) {
	home := t.TempDir()
	createProfile(t, home, "chat-once")

	stdout, stderr, err := runAPS(t, home, fakeEnv("reply {{prompt}} {{model}}"), "chat", "chat-once", "--once", "hello", "--model", "stub-model")
	if err != nil {
		t.Fatalf("chat once failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "reply hello stub-model") {
		t.Fatalf("expected fake response, got stdout=%q stderr=%q", stdout, stderr)
	}

	sessions := readRegistry(t, home)
	var found bool
	for _, sess := range sessions {
		if sess.ProfileID == "chat-once" && sess.Type == "chat" && sess.Environment["chat_turn_count"] == "1" {
			found = true
			if !strings.Contains(sess.Environment["chat_transcript_json"], "hello") {
				t.Fatalf("transcript missing prompt: %+v", sess.Environment)
			}
		}
	}
	if !found {
		t.Fatalf("chat session not visible in registry: %+v", sessions)
	}
}

func TestChatSessionList_ShowsChatSession(t *testing.T) {
	home := t.TempDir()
	createProfile(t, home, "chat-list")
	if _, stderr, err := runAPS(t, home, fakeEnv("listed"), "chat", "chat-list", "--once", "hello"); err != nil {
		t.Fatalf("chat once failed: %v\nstderr=%s", err, stderr)
	}

	stdout, stderr, err := runAPS(t, home, nil, "session", "list")
	if err != nil {
		t.Fatalf("session list failed: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "chat-list") || !strings.Contains(stdout, "chat") {
		t.Fatalf("expected chat session in list, stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestChatRepl_Opens(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty is unix-only")
	}
	home := t.TempDir()
	createProfile(t, home, "chat-repl")

	out := runPTY(t, home, fakeEnv("repl response"), []string{"chat", "chat-repl"}, func(f *os.File) {
		_, _ = f.Write([]byte(" "))
		waitForOutput(t, "aps chat chat-repl")
		_, _ = f.Write([]byte{0x1b})
	})
	if !strings.Contains(out, "session chat-") {
		t.Fatalf("expected REPL status line, got %q", out)
	}
}

func TestChatAttach_ReplaysPriorTurns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty is unix-only")
	}
	home := t.TempDir()
	createProfile(t, home, "chat-attach")
	if _, stderr, err := runAPS(t, home, fakeEnv("first reply"), "chat", "chat-attach", "--once", "remember this"); err != nil {
		t.Fatalf("chat once failed: %v\nstderr=%s", err, stderr)
	}
	sessionID := onlyChatSessionID(t, home)

	out := runPTY(t, home, fakeEnv("second reply"), []string{"chat", "chat-attach", "--attach", sessionID}, func(f *os.File) {
		_, _ = f.Write([]byte(" "))
		waitForOutput(t, "remember this")
		_, _ = f.Write([]byte{0x1b})
	})
	if !strings.Contains(out, "first reply") {
		t.Fatalf("expected replayed assistant turn, got %q", out)
	}
}

type registrySession struct {
	ProfileID   string            `json:"profile_id"`
	Type        string            `json:"type"`
	Environment map[string]string `json:"environment"`
}

func readRegistry(t *testing.T, home string) map[string]registrySession {
	t.Helper()
	path := filepath.Join(home, ".local", "share", "aps", "sessions", "registry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var sessions map[string]registrySession
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("decode registry: %v\n%s", err, data)
	}
	return sessions
}

func onlyChatSessionID(t *testing.T, home string) string {
	t.Helper()
	sessions := readRegistry(t, home)
	for id, sess := range sessions {
		if sess.Type == "chat" {
			return id
		}
	}
	t.Fatalf("no chat session found: %+v", sessions)
	return ""
}

func runPTY(t *testing.T, home string, env map[string]string, args []string, interact func(*os.File)) string {
	t.Helper()
	cmd := prepareAPS(t, home, env, args...)
	f, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty.Start: %v", err)
	}
	defer f.Close()

	ptyOutput := &safeBuffer{}
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(ptyOutput, f)
		done <- err
	}()
	currentPTYOutput = ptyOutput
	interact(f)
	currentPTYOutput = nil

	waitErr := cmd.Wait()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	if waitErr != nil {
		t.Fatalf("aps pty failed: %v\noutput=%s", waitErr, ptyOutput.String())
	}
	return ptyOutput.String()
}

var currentPTYOutput *safeBuffer

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitForOutput(t *testing.T, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		seen := ""
		if currentPTYOutput != nil {
			seen = currentPTYOutput.String()
		}
		if strings.Contains(seen, want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	seen := ""
	if currentPTYOutput != nil {
		seen = currentPTYOutput.String()
	}
	t.Fatalf("timed out waiting for %q\nseen=%s", want, seen)
}
