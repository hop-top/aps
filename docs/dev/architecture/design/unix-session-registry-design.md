# Unix Session Registry Design

## Overview

This document proposes enhancements to the session registry design to better support Unix platforms (macOS and Linux), including Unix-specific session metadata (UID, GID), SSH key distribution, and tmux considerations across Unix platforms.

## Current Session Registry Schema

### Existing `SessionInfo` Structure

```go
type SessionInfo struct {
    ID          string            `json:"id"`
    ProfileID   string            `json:"profile_id"`
    ProfileDir  string            `json:"profile_dir,omitempty"`
    Command     string            `json:"command"`
    PID         int               `json:"pid"`
    Status      SessionStatus     `json:"status"`
    Tier        SessionTier       `json:"tier,omitempty"`
    TmuxSocket  string            `json:"tmux_socket,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    LastSeenAt  time.Time         `json:"last_seen_at"`
    Environment map[string]string `json:"environment,omitempty"`
}
```

## Proposed Unix-Specific Session Metadata

### 1. Extended `SessionInfo` Structure

Add Unix-specific fields:

```go
type SessionInfo struct {
    ID          string            `json:"id"`
    ProfileID   string            `json:"profile_id"`
    ProfileDir  string            `json:"profile_dir,omitempty"`
    Command     string            `json:"command"`
    PID         int               `json:"pid"`
    Status      SessionStatus     `json:"status"`
    Tier        SessionTier       `json:"tier,omitempty"`
    TmuxSocket  string            `json:"tmux_socket,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    LastSeenAt  time.Time         `json:"last_seen_at"`
    Environment map[string]string `json:"environment,omitempty"`

    // Unix-specific fields
    UID         int               `json:"uid,omitempty"`
    GID         int               `json:"gid,omitempty"`
    Username    string            `json:"username,omitempty"`
    Groupname   string            `json:"groupname,omitempty"`
    HomeDir     string            `json:"home_dir,omitempty"`
    Shell       string            `json:"shell,omitempty"`
    NamespaceID string            `json:"namespace_id,omitempty"`
    CgroupPath  string            `json:"cgroup_path,omitempty"`
    TTY         string            `json:"tty,omitempty"`
    PPID        int               `json:"ppid,omitempty"`
    PGID        int               `json:"pgid,omitempty"`
    Platform    string            `json:"platform,omitempty"`
}
```

**Field Descriptions:**

| Field | Description | Required |
|-------|-------------|----------|
| `UID` | User ID of session process | Yes (Unix) |
| `GID` | Group ID of session process | Yes (Unix) |
| `Username` | Username of session owner | Yes (Unix) |
| `Groupname` | Primary group name | Optional |
| `HomeDir` | Home directory of session user | Optional |
| `Shell` | Shell used by session | Optional |
| `NamespaceID` | Namespace identifier (Linux) | Optional |
| `CgroupPath` | cgroup path (Linux) | Optional |
| `TTY` | TTY device for session | Optional |
| `PPID` | Parent process ID | Optional |
| `PGID` | Process group ID | Optional |
| `Platform` | Platform identifier (macos/linux) | Yes (Unix) |

### 2. New Session Context Types

```go
type UnixSessionContext struct {
    User      *UnixUserInfo
    Namespace *LinuxNamespaceInfo
    Cgroup    *LinuxCgroupInfo
    Tmux      *TmuxSessionInfo
    Process   *UnixProcessInfo
}

type UnixUserInfo struct {
    UID      int
    GID      int
    Username string
    Group    string
    Groups   []GroupInfo
    HomeDir  string
    Shell    string
}

type GroupInfo struct {
    GID  int
    Name string
}

type LinuxNamespaceInfo struct {
    UserNamespaceID    string
    PIDNamespaceID     string
    MountNamespaceID   string
    NetworkNamespaceID string
    UTSNamespaceID     string
    IPCNamespaceID     string
    CgroupVersion      int
}

type LinuxCgroupInfo struct {
    Path      string
    Version   int
    CPUShares int64
    MemoryMB  int64
    PIDsLimit int64
}

type TmuxSessionInfo struct {
    SocketPath   string
    SessionName  string
    ServerPID    int
    WindowCount  int
    PaneCount    int
}

type UnixProcessInfo struct {
    PID      int
    PPID     int
    PGID     int
    TTY      string
    State    string
    StartTime time.Time
}
```

## SSH Key Distribution for Unix Platforms

### 1. SSH Key Management Structure

```go
type SSHKeyManager struct {
    profileDir  string
    sshDir      string
    keys        map[string]*SSHKeyInfo
}

type SSHKeyInfo struct {
    Type         string // "ed25519", "rsa"
    PublicKey    string
    PrivateKey   string
    Fingerprint  string
    CreatedAt    time.Time
    ExpiresAt    time.Time
    Permissions os.FileMode
}
```

### 2. SSH Key Distribution Architecture

```
APS CLI Profile                SSH Server
   |                               |
   | 1. Generate SSH Key           |
   |------------------------------->|
   |                               |
   | 2. Store Public Key            |
   |                               | 3. Add to authorized_keys
   |                               |<-------------------------------|
   |                               |
   | 4. Connect via SSH            |
   |<------------------------------|
```

### 3. SSH Key Generation

```go
func (m *SSHKeyManager) GenerateKeyPair(keyType string) (*SSHKeyInfo, error) {
    var key *SSHKeyInfo
    var err error

    switch keyType {
    case "ed25519":
        key, err = m.generateED25519Key()
    case "rsa":
        key, err = m.generateRSAKey()
    default:
        return nil, fmt.Errorf("unsupported key type: %s", keyType)
    }

    if err != nil {
        return nil, err
    }

    key.CreatedAt = time.Now()
    key.ExpiresAt = time.Now().Add(90 * 24 * time.Hour) // 90 days

    if err := m.saveKeys(key); err != nil {
        return nil, err
    }

    return key, nil
}

func (m *SSHKeyManager) generateED25519Key() (*SSHKeyInfo, error) {
    // Generate ED25519 key using golang.org/x/crypto/ssh
    privateKey, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, fmt.Errorf("failed to generate key: %w", err)
    }

    privateKeyPEM, err := ssh.MarshalPrivateKey(privateKey, "")
    if err != nil {
        return nil, fmt.Errorf("failed to marshal private key: %w", err)
    }

    publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to create public key: %w", err)
    }

    key := &SSHKeyInfo{
        Type:       "ed25519",
        PrivateKey: string(privateKeyPEM),
        PublicKey:  string(ssh.MarshalAuthorizedKey(publicKey)),
        Fingerprint: ssh.FingerprintSHA256(publicKey),
        Permissions: 0600,
    }

    return key, nil
}
```

### 4. SSH Key Storage

```go
func (m *SSHKeyManager) saveKeys(key *SSHKeyInfo) error {
    // Ensure SSH directory exists
    sshDir := filepath.Join(m.profileDir, ".ssh")
    if err := os.MkdirAll(sshDir, 0700); err != nil {
        return fmt.Errorf("failed to create SSH directory: %w", err)
    }

    // Save private key
    privateKeyPath := filepath.Join(sshDir, "id_ed25519")
    if err := os.WriteFile(privateKeyPath, []byte(key.PrivateKey), 0600); err != nil {
        return fmt.Errorf("failed to save private key: %w", err)
    }

    // Save public key
    publicKeyPath := filepath.Join(sshDir, "id_ed25519.pub")
    if err := os.WriteFile(publicKeyPath, []byte(key.PublicKey), 0644); err != nil {
        return fmt.Errorf("failed to save public key: %w", err)
    }

    return nil
}
```

### 5. SSH Key Distribution to Servers

#### macOS SSH Server

```go
func (m *SSHKeyManager) ConfigureMacOSSSH(username string, publicKey string) error {
    // Get user's SSH directory
    homeDir, err := m.getUserHomeDir(username)
    if err != nil {
        return err
    }

    sshDir := filepath.Join(homeDir, ".ssh")
    if err := os.MkdirAll(sshDir, 0700); err != nil {
        return fmt.Errorf("failed to create .ssh directory: %w", err)
    }

    // Add public key to authorized_keys
    authKeysPath := filepath.Join(sshDir, "authorized_keys")
    if err := m.appendAuthorizedKey(authKeysPath, publicKey); err != nil {
        return err
    }

    // Fix permissions
    if err := os.Chmod(authKeysPath, 0600); err != nil {
        return fmt.Errorf("failed to set permissions on authorized_keys: %w", err)
    }

    // Configure SSH server to run as user
    if err := m.configureSSHdUser(username); err != nil {
        return err
    }

    return nil
}

func (m *SSHKeyManager) configureSSHdUser(username string) error {
    // Add to sshd_config.d/aps.conf
    configPath := "/etc/ssh/sshd_config.d/aps.conf"
    config := fmt.Sprintf(`
# APS SSH Server Configuration
AllowUsers %s
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
`, username)

    return os.WriteFile(configPath, []byte(config), 0644)
}
```

#### Linux SSH Server

```go
func (m *SSHKeyManager) ConfigureLinuxSSH(username string, publicKey string) error {
    // Get user's SSH directory
    homeDir, err := m.getUserHomeDir(username)
    if err != nil {
        return err
    }

    sshDir := filepath.Join(homeDir, ".ssh")
    if err := os.MkdirAll(sshDir, 0700); err != nil {
        return fmt.Errorf("failed to create .ssh directory: %w", err)
    }

    // Add public key to authorized_keys
    authKeysPath := filepath.Join(sshDir, "authorized_keys")
    if err := m.appendAuthorizedKey(authKeysPath, publicKey); err != nil {
        return err
    }

    // Fix permissions
    if err := os.Chmod(authKeysPath, 0600); err != nil {
        return fmt.Errorf("failed to set permissions on authorized_keys: %w", err)
    }

    // Change ownership to user
    uid, gid, err := m.getUserIDs(username)
    if err != nil {
        return err
    }

    if err := os.Chown(sshDir, uid, gid); err != nil {
        return fmt.Errorf("failed to chown .ssh directory: %w", err)
    }

    if err := os.Chown(authKeysPath, uid, gid); err != nil {
        return fmt.Errorf("failed to chown authorized_keys: %w", err)
    }

    // Configure SSH server
    return m.configureLinuxSSHD()
}

func (m *SSHKeyManager) configureLinuxSSHD() error {
    // Add to sshd_config.d/aps.conf
    configPath := "/etc/ssh/sshd_config.d/aps.conf"
    config := `
# APS SSH Server Configuration
PasswordAuthentication no
PubkeyAuthentication yes
PermitRootLogin no
X11Forwarding no
AllowTcpForwarding yes
`

    return os.WriteFile(configPath, []byte(config), 0644)
}
```

### 6. SSH Key Rotation

```go
func (m *SSHKeyManager) RotateKeys(oldKey, newKey *SSHKeyInfo) error {
    // Remove old key from authorized_keys
    if err := m.removeAuthorizedKey(oldKey.PublicKey); err != nil {
        return err
    }

    // Add new key to authorized_keys
    if err := m.appendAuthorizedKey("", newKey.PublicKey); err != nil {
        return err
    }

    // Save new keys
    if err := m.saveKeys(newKey); err != nil {
        return err
    }

    // Remove old key files
    if err := m.removeOldKeyFiles(oldKey); err != nil {
        return err
    }

    return nil
}
```

## Tmux Considerations Across Unix Platforms

### 1. Tmux Socket Location Differences

| Platform | Default Socket Location | APS Socket Location |
|----------|------------------------|---------------------|
| macOS    | `/tmp/tmux-{uid}/default` | `/tmp/aps-tmux-{profile-id}-socket` |
| Linux    | `/tmp/tmux-{uid}/default` | `/tmp/aps-tmux-{profile-id}-socket` |

### 2. Tmux Version Compatibility

```go
type TmuxVersion struct {
    Major int
    Minor int
    Patch int
}

func DetectTmuxVersion() (*TmuxVersion, error) {
    cmd := exec.Command("tmux", "-V")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("tmux not found: %w", err)
    }

    // Parse version from output like "tmux 3.2a"
    return parseTmuxVersion(string(output))
}

func parseTmuxVersion(output string) (*TmuxVersion, error) {
    // Implementation details for parsing version string
    // ...
}
```

### 3. Tmux Feature Detection

```go
type TmuxFeatures struct {
    Version     *TmuxVersion
    NewPane     bool
    HasFlags    bool
    HasUnicode  bool
    Supports256 bool
}

func DetectTmuxFeatures() (*TmuxFeatures, error) {
    version, err := DetectTmuxVersion()
    if err != nil {
        return nil, err
    }

    features := &TmuxFeatures{
        Version: version,
    }

    // Detect features based on version
    if version.Major >= 2 {
        features.NewPane = true
        features.HasFlags = true
    }

    if version.Major >= 1 || version.Minor >= 8 {
        features.HasUnicode = true
    }

    return features, nil
}
```

### 4. Tmux Session Creation

```go
func (t *TmuxManager) CreateSession(profileID string, command string) (*TmuxSessionInfo, error) {
    // Create unique socket
    socket := filepath.Join(os.TempDir(), fmt.Sprintf("aps-tmux-%s-socket", profileID))
    sessionName := fmt.Sprintf("aps-%s-%d", profileID, time.Now().Unix())

    // Create new session
    cmd := exec.Command("tmux", "-S", socket, "new-session", "-d", "-s", sessionName, "-n", "aps", command)
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("failed to create tmux session: %w", err)
    }

    // Get session info
    serverPID, err := t.getServerPID(socket)
    if err != nil {
        return nil, err
    }

    windowCount, err := t.getWindowCount(socket, sessionName)
    if err != nil {
        return nil, err
    }

    paneCount, err := t.getPaneCount(socket, sessionName)
    if err != nil {
        return nil, err
    }

    sessionInfo := &TmuxSessionInfo{
        SocketPath:  socket,
        SessionName: sessionName,
        ServerPID:   serverPID,
        WindowCount: windowCount,
        PaneCount:   paneCount,
    }

    return sessionInfo, nil
}

func (t *TmuxManager) getServerPID(socket string) (int, error) {
    cmd := exec.Command("tmux", "-S", socket, "display-message", "-p", "'#{pid}'")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return 0, fmt.Errorf("failed to get server PID: %w", err)
    }

    pidStr := strings.Trim(strings.TrimSpace(string(output)), "'")
    pid, err := strconv.Atoi(pidStr)
    if err != nil {
        return 0, fmt.Errorf("failed to parse PID: %w", err)
    }

    return pid, nil
}
```

### 5. Tmux Platform-Specific Issues

#### macOS Tmux Issues

**Issue**: tmux paste buffer may fail on macOS with large data
**Solution**: Use `-p -` flag and chunk large pastes

```go
func (t *TmuxManager) SendKeys(socket, session, window, pane string, keys string) error {
    // macOS tmux has issues with large paste buffers
    const maxPasteSize = 4096

    if len(keys) > maxPasteSize {
        return t.chunkAndSend(socket, session, window, pane, keys, maxPasteSize)
    }

    cmd := exec.Command("tmux", "-S", socket, "send-keys", "-t", fmt.Sprintf("%s:%s.%s", session, window, pane), keys)
    return cmd.Run()
}
```

**Issue**: macOS tmux may not restore clipboard
**Solution**: Use pbcopy/macOS-specific clipboard utilities

```go
func (t *TmuxManager) CopyToClipboard(socket, session string) error {
    // macOS-specific clipboard handling
    cmd := exec.Command("tmux", "-S", socket, "save-buffer", "-b", "clipboard", "-")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return err
    }

    // Use pbcopy to copy to macOS clipboard
    pbcopyCmd := exec.Command("pbcopy")
    stdin, err := pbcopyCmd.StdinPipe()
    if err != nil {
        return err
    }
    defer stdin.Close()

    go func() {
        stdin.Write(output)
    }()

    return pbcopyCmd.Run()
}
```

#### Linux Tmux Issues

**Issue**: tmux on Linux may not set proper terminal type
**Solution**: Explicitly set TERM environment variable

```go
func (t *TmuxManager) SetTerminalType(socket, session string, term string) error {
    cmd := exec.Command("tmux", "-S", socket, "set-environment", "-t", session, "TERM", term)
    return cmd.Run()
}
```

**Issue**: tmux may not work with certain terminal emulators
**Solution**: Detect and work around terminal issues

```go
func (t *TmuxManager) DetectTerminalEmulator() string {
    term := os.Getenv("TERM_PROGRAM")
    if term != "" {
        return term
    }

    term = os.Getenv("TERM")
    if term != "" {
        return term
    }

    return "unknown"
}

func (t *TmuxManager) WorkaroundTerminalIssues(terminal string) error {
    switch terminal {
    case "iTerm.app":
        return t.workaroundITerm()
    case "Terminal.app":
        return t.workaroundTerminalApp()
    default:
        return nil
    }
}
```

### 6. Tmux Session Cleanup

```go
func (t *TmuxManager) CleanupSession(socket, sessionName string) error {
    // Kill all panes in session
    cmd := exec.Command("tmux", "-S", socket, "kill-session", "-t", sessionName)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to kill tmux session: %w", err)
    }

    // Remove socket file
    if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to remove tmux socket: %w", err)
    }

    return nil
}
```

## Session Registry Enhancements

### 1. Unix Session Registration

```go
func (r *SessionRegistry) RegisterUnixSession(sess *SessionInfo, unixContext *UnixSessionContext) error {
    // Set Unix-specific metadata
    sess.UID = unixContext.User.UID
    sess.GID = unixContext.User.GID
    sess.Username = unixContext.User.Username
    sess.Groupname = unixContext.User.Group
    sess.HomeDir = unixContext.User.HomeDir
    sess.Shell = unixContext.User.Shell

    // Set platform
    sess.Platform = getUnixPlatform()

    // Set namespace ID (Linux)
    if unixContext.Namespace != nil {
        sess.NamespaceID = unixContext.Namespace.UserNamespaceID
    }

    // Set cgroup path (Linux)
    if unixContext.Cgroup != nil {
        sess.CgroupPath = unixContext.Cgroup.Path
    }

    // Set tmux info
    if unixContext.Tmux != nil {
        sess.TmuxSocket = unixContext.Tmux.SocketPath
    }

    // Register session
    return r.Register(sess)
}
```

### 2. Unix Session Querying

```go
func (r *SessionRegistry) GetSessionsByUser(username string) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.Username == username {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

func (r *SessionRegistry) GetSessionsByUID(uid int) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.UID == uid {
            sessions = append(sessions, session)
        }
    }

    return sessions
}

func (r *SessionRegistry) GetSessionsByPlatform(platform string) []*SessionInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    sessions := make([]*SessionInfo, 0)
    for _, session := range r.sessions {
        if session.Platform == platform {
            sessions = append(sessions, session)
        }
    }

    return sessions
}
```

### 3. Unix Session Validation

```go
func (r *SessionRegistry) ValidateUnixSession(sess *SessionInfo) error {
    // Validate required Unix fields
    if sess.UID == 0 || sess.GID == 0 {
        return fmt.Errorf("invalid UID/GID: %d/%d", sess.UID, sess.GID)
    }

    // Validate username
    if sess.Username == "" {
        return fmt.Errorf("username is required")
    }

    // Validate platform
    if sess.Platform != "macos" && sess.Platform != "linux" {
        return fmt.Errorf("invalid platform: %s", sess.Platform)
    }

    // Validate process exists
    if sess.PID > 0 {
        if !r.processExists(sess.PID) {
            return fmt.Errorf("process not found: %d", sess.PID)
        }
    }

    return nil
}

func (r *SessionRegistry) processExists(pid int) bool {
    _, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
    if err == nil {
        return true
    }
    if os.IsNotExist(err) {
        return false
    }

    // Fallback for macOS
    cmd := exec.Command("ps", "-p", strconv.Itoa(pid))
    return cmd.Run() == nil
}
```

## Potential Issues with tmux Across Unix Platforms

### 1. Socket Path Conflicts

**Issue**: Multiple profiles may have conflicting socket paths
**Solution**: Include profile ID and timestamp in socket path

```go
func GenerateTmuxSocketPath(profileID string) string {
    return filepath.Join(os.TempDir(), fmt.Sprintf("aps-tmux-%s-%d.sock", profileID, time.Now().UnixNano()))
}
```

### 2. Tmux Server PID Tracking

**Issue**: tmux server PID may change if server restarts
**Solution**: Periodically check and update server PID

```go
func (t *TmuxManager) RefreshServerPID(socket string) (int, error) {
    return t.getServerPID(socket)
}
```

### 3. Tmux Session Reattachment

**Issue**: Reattaching to session may fail if session is attached elsewhere
**Solution**: Force detach other clients before attaching

```go
func (t *TmuxManager) ForceAttach(socket, sessionName string) error {
    // Detach other clients
    cmd := exec.Command("tmux", "-S", socket, "detach-client", "-a")
    _ = cmd.Run() // Ignore errors

    // Attach to session
    cmd = exec.Command("tmux", "-S", socket, "attach", "-t", sessionName)
    return cmd.Run()
}
```

### 4. Tmux Terminal Compatibility

**Issue**: Some terminals may not work well with tmux
**Solution**: Detect terminal and apply workarounds

```go
func (t *TmuxManager) DetectTerminal() string {
    term := os.Getenv("TERM_PROGRAM")
    if term != "" {
        return term
    }

    // Check for terminal emulators
    if _, err := exec.LookPath("iTerm2"); err == nil {
        return "iTerm2"
    }

    return "unknown"
}
```

## Acceptance Criteria

✅ **Session Registry Schema**
- [ ] `SessionInfo` includes Unix-specific metadata fields
- [ ] Supporting types for Unix context defined
- [ ] Session registry methods for Unix queries implemented

✅ **SSH Key Distribution**
- [ ] SSH key generation implemented
- [ ] macOS SSH key distribution documented
- [ ] Linux SSH key distribution documented
- [ ] SSH key rotation mechanism proposed

✅ **Tmux Considerations**
- [ ] tmux socket location differences documented
- [ ] tmux version compatibility handled
- [ ] macOS-specific tmux issues addressed
- [ ] Linux-specific tmux issues addressed
- [ ] Cross-platform tmux issues identified

✅ **Platform Support**
- [ ] macOS session registry integration proposed
- [ ] Linux session registry integration proposed
- [ ] Session validation for Unix platforms defined
