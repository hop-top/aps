package messenger

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const hashPhone = "+14155550123"

// useKeyFile points the installation key at dir and drops cached keys,
// as a fresh process would see it.
func useKeyFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, idKeyFile)
	old := idKeyPath
	idKeyPath = func() (string, error) { return path, nil }
	resetIDKeyCache()
	t.Cleanup(func() { idKeyPath = old; resetIDKeyCache() })
	return path
}

func resetIDKeyCache() {
	idKeyMu.Lock()
	idKeyCache = map[string][]byte{}
	idKeyMu.Unlock()
}

func TestIDHMAC_StableAcrossWritesAndRestarts(t *testing.T) {
	useKeyFile(t, t.TempDir())
	h1 := idHMAC(hashPhone)
	if len(h1) != 2*idHMACSize {
		t.Fatalf("hash %q: want %d hex chars", h1, 2*idHMACSize)
	}
	resetIDKeyCache() // simulated restart: key re-read from disk
	if h2 := idHMAC(hashPhone); h2 != h1 {
		t.Fatalf("hash changed after restart: %s vs %s", h1, h2)
	}
	if idHMAC("+14155550124") == h1 {
		t.Fatal("different numbers share a hash")
	}
}

func TestIDHMAC_DiffersPerInstallation(t *testing.T) {
	useKeyFile(t, t.TempDir())
	a := idHMAC(hashPhone)
	useKeyFile(t, t.TempDir())
	if b := idHMAC(hashPhone); a == "" || a == b {
		t.Fatalf("installations share hash: %q %q", a, b)
	}
}

func TestIDKey_FileMode0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode bits")
	}
	path := useKeyFile(t, t.TempDir())
	if idHMAC(hashPhone) == "" {
		t.Fatal("no hash")
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, want 0600", st.Mode().Perm())
	}
	ents, _ := os.ReadDir(filepath.Dir(path))
	if len(ents) != 1 {
		t.Fatalf("temp files left behind: %v", ents)
	}
}

func TestIDKey_ConcurrentFirstUseCreatesOneKey(t *testing.T) {
	path := useKeyFile(t, t.TempDir())
	const n = 32
	keys := make([][]byte, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			// bypass the in-process cache so each caller races on the file
			k, err := readOrCreateForTest(path)
			if err != nil {
				t.Error(err)
			}
			keys[i] = k
		}(i)
	}
	close(start)
	wg.Wait()
	onDisk, _ := os.ReadFile(path)
	for i, k := range keys {
		if !bytes.Equal(k, onDisk) {
			t.Fatalf("caller %d got a different key than the file holds", i)
		}
	}
}

// readOrCreateForTest is loadIDKey without the process-wide lock/cache,
// so goroutines exercise the same create-or-adopt path as separate
// processes.
func readOrCreateForTest(path string) ([]byte, error) {
	k, err := readIDKey(path)
	if errors.Is(err, os.ErrNotExist) {
		return createIDKey(path)
	}
	return k, err
}

func TestIDKey_ConcurrentProcessesAgree(t *testing.T) {
	if os.Getenv("APS_IDHASH_CHILD") == "1" {
		return
	}
	dir := t.TempDir()
	const n = 8
	outs := make([]*bytes.Buffer, n)
	cmds := make([]*exec.Cmd, n)
	for i := range cmds {
		outs[i] = &bytes.Buffer{}
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestIDHMAC_ChildHelper$")
		cmds[i].Env = append(os.Environ(), "APS_IDHASH_CHILD=1", "APS_IDHASH_DIR="+dir)
		cmds[i].Stdout = outs[i]
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	var first string
	for i, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Fatalf("child %d: %v", i, err)
		}
		got := extractHash(outs[i].String())
		if got == "" {
			t.Fatalf("child %d no hash: %q", i, outs[i].String())
		}
		if first == "" {
			first = got
		} else if got != first {
			t.Fatalf("processes disagree: %s vs %s", first, got)
		}
	}
	// and a fresh in-process reader sees the same key
	useKeyFile(t, dir)
	if h := idHMAC(hashPhone); h != first {
		t.Fatalf("parent hash %s != children %s", h, first)
	}
}

func extractHash(s string) string {
	const p = "HASH="
	i := strings.Index(s, p)
	if i < 0 {
		return ""
	}
	rest := s[i+len(p):]
	if j := strings.IndexByte(rest, '\n'); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}

// TestIDHMAC_ChildHelper runs only inside the child processes spawned
// above.
func TestIDHMAC_ChildHelper(t *testing.T) {
	if os.Getenv("APS_IDHASH_CHILD") != "1" {
		t.Skip("helper")
	}
	useKeyFile(t, os.Getenv("APS_IDHASH_DIR"))
	_, _ = os.Stdout.WriteString("HASH=" + idHMAC(hashPhone) + "\n")
}

func TestPhonePlatforms_RedactedWithStableHash(t *testing.T) {
	useKeyFile(t, t.TempDir())
	want := idHMAC(hashPhone)
	for _, p := range []MessengerPlatform{PlatformWhatsApp, PlatformSMS, PlatformEmail} {
		t.Run(string(p), func(t *testing.T) {
			dir := t.TempDir()
			a := NewAuditLogger(dir).WithPlatform(p)
			for i := 0; i < 2; i++ { // stable across writes
				if err := a.LogMappingChanged("m", "prof", hashPhone, "", "agent:prof:run"); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.LogLinkCreated("m", "prof", map[string]string{hashPhone: "agent:prof:run"}); err != nil {
				t.Fatal(err)
			}
			raw := readFileT(t, filepath.Join(dir, "logs", "audit", "capability-changes.jsonl"))
			if strings.Contains(raw, "4155550123") {
				t.Fatalf("phone persisted: %s", raw)
			}
			es := auditEntries(t, dir)
			for _, e := range es[:2] {
				if e["channel_id_hmac"] != want {
					t.Fatalf("channel_id_hmac = %v, want %s", e["channel_id_hmac"], want)
				}
			}
			m := es[2]["mappings"].([]any)[0].(map[string]any)
			if m["channel_id_hmac"] != want {
				t.Fatalf("mapping hmac = %v", m["channel_id_hmac"])
			}

			l := NewWorkspaceMessageLogger(dir, "m").WithPlatform(p)
			if err := l.LogMessageSent(hashPhone, hashPhone, "ok"); err != nil {
				t.Fatal(err)
			}
			le := readJSONLEntries(t, filepath.Join(l.mustLogDir(t), "_all.jsonl"))[0]
			if le["message_id_hmac"] != want || le["channel_id_hmac"] != want {
				t.Fatalf("log hmacs missing: %v", le)
			}
			if strings.Contains(readFileT(t, filepath.Join(l.mustLogDir(t), "_all.jsonl")), "4155550123") {
				t.Fatal("phone persisted in log")
			}
		})
	}
}

func TestVerbatimPlatforms_NoHash(t *testing.T) {
	useKeyFile(t, t.TempDir())
	dir := t.TempDir()
	a := NewAuditLogger(dir).WithPlatform(PlatformSlack)
	if err := a.LogMappingChanged("m", "prof", "C01ABC2DEF", "", "x"); err != nil {
		t.Fatal(err)
	}
	if _, ok := auditEntries(t, dir)[0]["channel_id_hmac"]; ok {
		t.Fatal("verbatim platform got a hash")
	}
}

func TestUnreadableKey_NoHashAndStillRedacted(t *testing.T) {
	path := useKeyFile(t, t.TempDir())
	if err := os.WriteFile(path, []byte("short"), 0o600); err != nil { // corrupt key
		t.Fatal(err)
	}
	dir := t.TempDir()
	a := NewAuditLogger(dir).WithPlatform(PlatformWhatsApp)
	if err := a.LogMappingChanged("m", "prof", hashPhone, "", "x"); err != nil {
		t.Fatal(err)
	}
	raw := readFileT(t, filepath.Join(dir, "logs", "audit", "capability-changes.jsonl"))
	if strings.Contains(raw, "4155550123") || strings.Contains(raw, "_hmac") {
		t.Fatalf("want redacted and no hash: %s", raw)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "short" {
		t.Fatal("unreadable key was overwritten")
	}
}

func TestHMACFieldsSurviveRedaction(t *testing.T) {
	useKeyFile(t, t.TempDir())
	dir := t.TempDir()
	a := NewAuditLogger(dir).WithPlatform(PlatformSMS)
	for i := 0; i < 300; i++ {
		num := "+1415555" + strings.Repeat("0", 4-len(itoa(i%10000))) + itoa(i%10000)
		if err := a.LogMappingChanged("m", "prof", num, "", "x"); err != nil {
			t.Fatal(err)
		}
		if got := auditEntries(t, dir)[i]["channel_id_hmac"]; got != idHMAC(num) {
			t.Fatalf("hash for %s mangled: %v", num, got)
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for ; i > 0; i /= 10 {
		b = append([]byte{byte('0' + i%10)}, b...)
	}
	return string(b)
}
