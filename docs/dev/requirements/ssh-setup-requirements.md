# SSH Server Setup Requirements for Platform Adapters

## Overview

This document defines the SSH server setup requirements for each platform adapter. SSH servers are used for remote command execution, session attachment, and secure communication with isolated processes.

## Common Requirements

### SSH Key Management

All platforms must support:
- SSH key pair generation (ED25519 or RSA-4096 recommended)
- SSH public key authentication
- SSH agent forwarding support
- Known hosts management

### SSH Security

All platforms must enforce:
- SSH protocol version 2 only
- No password authentication (key-based only)
- No root login
- Strict host key checking
- Connection timeout and keepalive

### SSH Configuration

```yaml
ssh:
  enabled: true
  key_path: "~/.ssh/aps_server_key"
  port: 2222
  listen_address: "127.0.0.1"
  max_auth_tries: 3
  login_grace_time: 60
  client_alive_interval: 300
  client_alive_count_max: 3
  allowed_users: []
  allowed_groups: []
  deny_users: ["root"]
  deny_groups: []
```

## Linux SSH Server Setup

### Prerequisites

**Required Packages**:
```bash
# Debian/Ubuntu
sudo apt-get install openssh-server

# RHEL/CentOS
sudo yum install openssh-server

# Arch Linux
sudo pacman -S openssh
```

**System Requirements**:
- Linux kernel 2.6+
- systemd or init system
- Network connectivity

### Installation Steps

1. **Install SSH Server**:
```bash
# Debian/Ubuntu
sudo apt-get update
sudo apt-get install openssh-server

# Verify installation
sshd -V
```

2. **Generate SSH Keys**:
```bash
# Generate ED25519 key pair (recommended)
ssh-keygen -t ed25519 -f ~/.ssh/aps_server_key -N ""

# Or generate RSA key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/aps_server_key -N ""
```

3. **Configure SSH Server**:
```bash
# Backup original config
sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup

# Create APS-specific config
sudo tee /etc/ssh/sshd_config.d/aps.conf << EOF
# APS SSH Server Configuration
Port 2222
ListenAddress 127.0.0.1
Protocol 2

# Authentication
PubkeyAuthentication yes
PasswordAuthentication no
PermitRootLogin no

# Security
MaxAuthTries 3
LoginGraceTime 60
ClientAliveInterval 300
ClientAliveCountMax 3

# Key Management
HostKey /home/$USER/.ssh/aps_server_key

# Logging
SyslogFacility AUTH
LogLevel INFO

# Connection limits
MaxStartups 10:30:100
MaxSessions 10
EOF
```

4. **Set Permissions**:
```bash
# Set correct permissions
chmod 700 ~/.ssh
chmod 600 ~/.ssh/aps_server_key
chmod 644 ~/.ssh/aps_server_key.pub

# Fix ownership
sudo chown -R $USER:$USER ~/.ssh
```

5. **Enable and Start SSH Server**:
```bash
# Enable service
sudo systemctl enable sshd

# Start service
sudo systemctl start sshd

# Check status
sudo systemctl status sshd
```

6. **Configure Firewall**:
```bash
# UFW (Ubuntu)
sudo ufw allow 2222/tcp

# firewalld (RHEL/CentOS)
sudo firewall-cmd --permanent --add-port=2222/tcp
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 2222 -j ACCEPT
sudo iptables-save | sudo tee /etc/iptables/rules.v4
```

7. **Test Connection**:
```bash
# Test connection
ssh -i ~/.ssh/aps_server_key -p 2222 $USER@127.0.0.1

# Test with verbose output
ssh -vvv -i ~/.ssh/aps_server_key -p 2222 $USER@127.0.0.1
```

### systemd Service

Create APS-specific systemd service:

```bash
# Create service file
sudo tee /etc/systemd/system/aps-ssh.service << EOF
[Unit]
Description=APS SSH Server
After=network.target

[Service]
Type=simple
User=$USER
Group=$USER
ExecStart=/usr/sbin/sshd -D -f /etc/ssh/sshd_config.d/aps.conf
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable aps-ssh.service
sudo systemctl start aps-ssh.service
```

### SELinux Configuration (if enabled)

```bash
# Check SELinux status
sestatus

# Allow SSH to use non-standard port
sudo semanage port -a -t ssh_port_t -p tcp 2222

# Allow read access to custom key directory
sudo semanage fcontext -a -t ssh_home_t "/home/[^/]*/\.ssh(/.*)?"
sudo restorecon -R -v ~/.ssh
```

## macOS SSH Server Setup

### Prerequisites

**System Requirements**:
- macOS 10.12 (Sierra) or later
- Remote Login enabled in System Preferences

### Installation Steps

1. **Enable Remote Login**:
```bash
# Enable Remote Login
sudo systemsetup -setremotelogin on

# Verify Remote Login is enabled
sudo systemsetup -getremotelogin
```

Or via System Preferences:
- System Preferences → Sharing → Remote Login

2. **Generate SSH Keys**:
```bash
# Generate ED25519 key pair
ssh-keygen -t ed25519 -f ~/.ssh/aps_server_key -N ""

# Or generate RSA key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/aps_server_key -N ""
```

3. **Configure SSH Server**:
```bash
# Backup original config
sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup

# Create APS-specific config
sudo tee /etc/ssh/sshd_config.d/aps.conf << EOF
# APS SSH Server Configuration
Port 2222
ListenAddress 127.0.0.1
Protocol 2

# Authentication
PubkeyAuthentication yes
PasswordAuthentication no
PermitRootLogin no

# Security
MaxAuthTries 3
LoginGraceTime 60
ClientAliveInterval 300
ClientAliveCountMax 3

# Key Management
HostKey ~/.ssh/aps_server_key

# Logging
SyslogFacility AUTHPRIV
LogLevel INFO

# Connection limits
MaxStartups 10:30:100
MaxSessions 10
EOF
```

4. **Set Permissions**:
```bash
# Set correct permissions
chmod 700 ~/.ssh
chmod 600 ~/.ssh/aps_server_key
chmod 644 ~/.ssh/aps_server_key.pub
```

5. **Restart SSH Server**:
```bash
# Stop SSH server
sudo launchctl stop com.openssh.sshd

# Start SSH server
sudo launchctl start com.openssh.sshd

# Or use systemsetup
sudo systemsetup -restartssh
```

6. **Configure Firewall**:
```bash
# Allow port 2222
/usr/libexec/ApplicationFirewall/socketfilterfw --add /usr/sbin/sshd
/usr/libexec/ApplicationFirewall/socketfilterfw --unblock /usr/sbin/sshd
```

7. **Test Connection**:
```bash
# Test connection
ssh -i ~/.ssh/aps_server_key -p 2222 $USER@127.0.0.1
```

### Launch Daemon

Create APS-specific launch daemon:

```bash
# Create plist file
sudo tee /Library/LaunchDaemons/com.aps.sshd.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.aps.sshd</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/sbin/sshd</string>
        <string>-D</string>
        <string>-f</string>
        <string>/etc/ssh/sshd_config.d/aps.conf</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>UserName</key>
    <string>$USER</string>
</dict>
</plist>
EOF

# Load the daemon
sudo launchctl load /Library/LaunchDaemons/com.aps.sshd.plist
```

## Windows SSH Server Setup

### Prerequisites

**System Requirements**:
- Windows 10 1809 or later
- Windows Server 2019 or later
- PowerShell 5.1 or later

**Installation Methods**:
- OpenSSH Server (built-in)
- Cygwin SSH
- Git Bash SSH

### Installation Steps (OpenSSH)

1. **Install OpenSSH Server**:
```powershell
# Check if OpenSSH is installed
Get-WindowsCapability -Online | Where-Object Name -like 'OpenSSH*'

# Install OpenSSH Server
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0

# Verify installation
Get-Service sshd
```

2. **Generate SSH Keys**:
```powershell
# Generate ED25519 key pair
ssh-keygen -t ed25519 -f $env:USERPROFILE\.ssh\aps_server_key -N ""

# Or generate RSA key pair
ssh-keygen -t rsa -b 4096 -f $env:USERPROFILE\.ssh\aps_server_key -N ""
```

3. **Configure SSH Server**:
```powershell
# Create APS configuration directory
New-Item -Path $env:ProgramData\ssh\aps -ItemType Directory -Force

# Create configuration file
@"
# APS SSH Server Configuration
Port 2222
ListenAddress 127.0.0.1
Protocol 2

# Authentication
PubkeyAuthentication yes
PasswordAuthentication no
PermitRootLogin no

# Security
MaxAuthTries 3
LoginGraceTime 60
ClientAliveInterval 300
ClientAliveCountMax 3

# Key Management
HostKey $env:USERPROFILE\.ssh\aps_server_key

# Logging
SyslogFacility AUTH
LogLevel INFO

# Connection limits
MaxStartups 10:30:100
MaxSessions 10
"@ | Out-File -FilePath $env:ProgramData\ssh\aps\sshd_config -Encoding ASCII
```

4. **Set Permissions**:
```powershell
# Set correct permissions
icacls $env:USERPROFILE\.ssh /inheritance:r
icacls $env:USERPROFILE\.ssh /grant:r "$env:USERNAME:(OI)(CI)F"
icacls $env:USERPROFILE\.ssh\aps_server_key /inheritance:r
icacls $env:USERPROFILE\.ssh\aps_server_key /grant:r "$env:USERNAME:F"
icacls $env:USERPROFILE\.ssh\aps_server_key.pub /inheritance:r
icacls $env:USERPROFILE\.ssh\aps_server_key.pub /grant:r "$env:USERNAME:(R)"
```

5. **Configure Windows Firewall**:
```powershell
# Add firewall rule
New-NetFirewallRule -DisplayName "APS SSH Server" `
    -Direction Inbound `
    -LocalPort 2222 `
    -Protocol TCP `
    -Action Allow
```

6. **Enable and Start SSH Service**:
```powershell
# Start service
Start-Service sshd

# Set service to automatic startup
Set-Service -Name sshd -StartupType Automatic

# Verify status
Get-Service sshd
```

7. **Test Connection**:
```powershell
# Test connection
ssh -i $env:USERPROFILE\.ssh\aps_server_key -p 2222 $env:USERNAME@127.0.0.1
```

### Windows Service

Create APS-specific Windows service:

```powershell
# Create service
sc.exe create "APS SSH Server" binPath= '"C:\Windows\System32\OpenSSH\sshd.exe" -f "C:\ProgramData\ssh\aps\sshd_config"' start= auto DisplayName= "APS SSH Server"

# Start service
sc.exe start "APS SSH Server"

# Verify service
sc.exe query "APS SSH Server"
```

## Container SSH Server Setup

### Dockerfile

```dockerfile
# Base image
FROM ubuntu:22.04

# Install SSH server
RUN apt-get update && \
    apt-get install -y openssh-server && \
    mkdir /var/run/sshd

# Create SSH user
RUN useradd -m -s /bin/bash aps

# Setup SSH directory
RUN mkdir -p /home/aps/.ssh && \
    chown aps:aps /home/aps/.ssh && \
    chmod 700 /home/aps/.ssh

# Copy SSH keys
COPY ssh_keys /tmp/ssh_keys
RUN mv /tmp/ssh_keys/server_key /home/aps/.ssh/aps_server_key && \
    mv /tmp/ssh_keys/server_key.pub /home/aps/.ssh/aps_server_key.pub && \
    chown aps:aps /home/aps/.ssh/aps_server_key* && \
    chmod 600 /home/aps/.ssh/aps_server_key && \
    chmod 644 /home/aps/.ssh/aps_server_key.pub

# Configure SSH
COPY sshd_config /etc/ssh/sshd_config.d/aps.conf

# Expose SSH port
EXPOSE 2222

# Start SSH server
CMD ["/usr/sbin/sshd", "-D", "-e", "-f", "/etc/ssh/sshd_config.d/aps.conf"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  aps-ssh:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "2222:2222"
    volumes:
      - ./profile:/home/aps/workspace:ro
      - ./actions:/home/aps/actions:ro
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - DAC_OVERRIDE
      - SETUID
      - SETGID
```

## Integration with APS

### Profile Configuration

```yaml
ssh:
  enabled: true
  server:
    port: 2222
    listen_address: "127.0.0.1"
    host_key_path: "~/.ssh/aps_server_key"
  client:
    key_path: "~/.ssh/aps_client_key"
    known_hosts: "~/.ssh/known_hosts"
    strict_host_key_checking: yes
```

### SSH Command Execution

```go
func (a *PlatformAdapter) ExecuteViaSSH(command string, args []string) error {
    config := &ssh.ClientConfig{
        User: a.config.SSHUsername,
        Auth: []ssh.AuthMethod{
            ssh.PublicKeys(a.sshKey),
        },
        HostKeyCallback: ssh.FixedHostKey(a.hostKey),
        Timeout:        30 * time.Second,
    }
    
    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", a.config.SSHHost, a.config.SSHPort), config)
    if err != nil {
        return fmt.Errorf("failed to dial: %w", err)
    }
    defer client.Close()
    
    session, err := client.NewSession()
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    defer session.Close()
    
    cmd := fmt.Sprintf("%s %s", command, strings.Join(args, " "))
    output, err := session.CombinedOutput(cmd)
    if err != nil {
        return fmt.Errorf("command failed: %w\n%s", err, output)
    }
    
    return nil
}
```

## Troubleshooting

### Linux

**Connection refused**:
```bash
# Check if SSH is running
sudo systemctl status sshd

# Check listening ports
sudo ss -tlnp | grep :2222

# Check logs
sudo journalctl -u sshd -n 50
```

**Permission denied**:
```bash
# Check key permissions
ls -la ~/.ssh/aps_server_key

# Fix permissions
chmod 600 ~/.ssh/aps_server_key
chmod 644 ~/.ssh/aps_server_key.pub
```

### macOS

**Connection refused**:
```bash
# Check if SSH is running
sudo launchctl list | grep ssh

# Check logs
log show --predicate 'process == "sshd"' --last 1h
```

**Permission denied**:
```bash
# Check key permissions
ls -la ~/.ssh/aps_server_key

# Fix permissions
chmod 600 ~/.ssh/aps_server_key
chmod 644 ~/.ssh/aps_server_key.pub
```

### Windows

**Connection refused**:
```powershell
# Check if SSH is running
Get-Service sshd

# Check firewall rules
Get-NetFirewallRule -DisplayName "*SSH*"

# Check logs
Get-WinEvent -LogName "OpenSSH/Operational" -MaxEvents 50
```

**Permission denied**:
```powershell
# Check key permissions
icacls $env:USERPROFILE\.ssh\aps_server_key

# Fix permissions
icacls $env:USERPROFILE\.ssh\aps_server_key /grant:r "$env:USERNAME:F"
```

## Security Best Practices

1. **Key Management**
   - Use ED25519 keys (preferred) or RSA-4096
   - Rotate keys regularly (every 90 days)
   - Never share private keys
   - Use different keys for different profiles

2. **Authentication**
   - Disable password authentication
   - Use only key-based authentication
   - Implement key revocation process
   - Use SSH certificates for large deployments

3. **Network Security**
   - Bind to localhost by default
   - Use firewall rules to restrict access
   - Implement IP whitelisting
   - Use VPN for remote access

4. **Monitoring and Auditing**
   - Enable SSH logging
   - Monitor failed login attempts
   - Implement intrusion detection
   - Regular security audits

## References

- [OpenSSH Documentation](https://www.openssh.com/manual.html)
- [SSH Configuration](https://man.openbsd.org/sshd_config)
- [Docker and SSH](https://docs.docker.com/engine/examples/running_ssh_service/)
