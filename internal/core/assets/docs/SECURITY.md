# Security

Security best practices and guidelines for APS.

## Overview

APS is designed with security as a core principle:

- **Local-first**: All data stays on your machine
- **Process-scoped isolation**: No global state modifications
- **Secure defaults**: Secrets are protected by default
- **Explicit configuration**: Nothing happens without your consent

## Secrets Management

### Secure by Default

**`secrets.env` is created with `0600` permissions:**
```bash
-rw------- 1 user user 123 Jan 1 12:00 secrets.env
```

Owner read/write only. No group or world permissions.

### Verify Permissions

Check permissions on secrets files:

```bash
ls -la ~/.agents/profiles/*/secrets.env
```

**Correct:**
```
-rw------- 1 user user 123 secrets.env
```

**Incorrect:**
```
-rw-r--r-- 1 user user 123 secrets.env
```

### Fix Permissions

```bash
chmod 600 ~/.agents/profiles/*/secrets.env
```

### Never Commit Secrets

Add to `.gitignore`:

```bash
echo "**/.agents/profiles/*/secrets.env" >> .gitignore
echo "**/.agents/profiles/*/ssh.key" >> .gitignore
```

### Best Practices

1. **Minimal secrets**: Only add secrets the profile needs
2. **Rotate regularly**: Change API keys and tokens periodically
3. **Use environment-specific profiles**: Separate dev/staging/prod
4. **Audit regularly**: Review secrets.env files
5. **Don't share profiles**: Export config only, not secrets
6. **Use token scopes**: Create tokens with minimal permissions
7. **Monitor usage**: Track API usage for unusual activity

## File Permissions

### Expected Permissions

| File | Permissions |
|------|-------------|
| `secrets.env` | `0600` (owner rw only) |
| `ssh.key` | `0600` (owner rw only) |
| `profile.yaml` | `0644` (owner rw, group/world r) |
| `actions/*.sh` | `0755` (owner rwx, group/world rx) |
| `notes.md` | `0644` (owner rw, group/world r) |

### Verify All Permissions

```bash
find ~/.agents/profiles -type f -exec ls -la {} \;
```

### Fix All Permissions

```bash
# Fix secrets
chmod 600 ~/.agents/profiles/*/secrets.env

# Fix SSH keys
chmod 600 ~/.agents/profiles/*/ssh.key

# Fix actions
chmod 755 ~/.agents/profiles/*/actions/*

# Fix other files
chmod 644 ~/.agents/profiles/*/profile.yaml
chmod 644 ~/.agents/profiles/*/notes.md
```

## SSH Keys

### Generate SSH Keys

Generate a new SSH key for a profile:

```bash
ssh-keygen -t ed25519 -f ~/.agents/profiles/myagent/ssh.key -N ""
```

**Key properties:**
- Algorithm: Ed25519 (recommended)
- No passphrase (for automated use)
- File permissions: `0600`

### Add to Git Config

In `profile.yaml`:

```yaml
ssh:
  enabled: true
  key_path: "~/.agents/profiles/myagent/ssh.key"
```

APS will inject `GIT_SSH_COMMAND` with the correct key path.

### Use with SSH

```bash
aps myagent -- git push origin main
```

APS uses the profile's SSH key for the git operation.

### Best Practices

1. **Use Ed25519**: More secure than RSA
2. **No passphrase**: For automated operations
3. **Dedicated keys**: One key per profile
4. **Rotate keys**: Replace keys periodically
5. **Limit access**: Add public keys to services with minimal permissions
6. **Monitor access**: Review SSH access logs

## Environment Variables

### Profile Isolation

Each profile has its own environment:

- `<PREFIX>_PROFILE_*` variables are profile-specific
- Secrets from `secrets.env` are only injected when running that profile
- No cross-profile leakage

### Sensitive Variables

APS injects these variables when running commands:

- `APS_PROFILE_DIR` - Profile directory path
- `APS_PROFILE_YAML` - Profile configuration path
- `APS_PROFILE_SECRETS` - Secrets file path

**Note**: These are paths, not secrets. Actual secrets are read from `secrets.env`.

### Redaction

When displaying profile information, secret values are redacted:

```bash
aps profile show myagent
```

Output:
```
secrets:
  GITHUB_TOKEN: ***redacted***
  API_KEY: ***redacted***
```

### Best Practices

1. **Don't log secrets**: Actions should avoid printing secrets
2. **Use environment variables**: Pass secrets via env, not arguments
3. **Clean up**: Remove secrets from environment after use
4. **Minimize exposure**: Only add secrets to profiles that need them

## Webhooks

### Signature Validation

Always use signature validation in production:

```bash
aps webhook serve \
  --secret "your-secret-key" \
  --event-map github.push=bot:deploy.sh
```

### Generate Secure Secrets

Use cryptographically secure secrets:

**Generate 32-byte random secret:**
```bash
openssl rand -hex 32
```

**Generate base64 secret:**
```bash
openssl rand -base64 32
```

### IP Filtering

Use reverse proxy for IP filtering:

**Nginx example:**
```nginx
location /webhook {
    allow 192.168.1.0/24;  # Allow internal network
    allow 1.2.3.4;         # Allow GitHub IPs
    deny all;              # Deny everything else

    proxy_pass http://127.0.0.1:8080/webhook;
}
```

### Rate Limiting

Implement rate limiting at proxy level:

**Nginx rate limiting:**
```nginx
limit_req_zone $binary_remote_addr zone=webhook:10m rate=10r/m;

location /webhook {
    limit_req zone=webhook burst=20 nodelay;
    proxy_pass http://127.0.0.1:8080/webhook;
}
```

### HTTPS

Always use HTTPS for production webhooks:

**Nginx SSL:**
```nginx
server {
    listen 443 ssl;
    server_name webhook.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /webhook {
        proxy_pass http://127.0.0.1:8080/webhook;
    }
}
```

### Best Practices

1. **Always use HTTPS**: Encrypt webhook traffic
2. **Validate signatures**: Verify request authenticity
3. **Use secrets**: Don't send webhooks without secret
4. **IP filtering**: Restrict source IPs
5. **Rate limiting**: Prevent abuse
6. **Dry run first**: Test webhook handlers
7. **Monitor logs**: Watch for suspicious activity
8. **Audit events**: Log all webhook events

## Access Control

### File System

**Profile directory structure:**
```
~/.agents/profiles/<id>/
  profile.yaml    - 0644
  secrets.env     - 0600 (sensitive)
  gitconfig       - 0644
  ssh.key         - 0600 (sensitive)
  actions/        - 0755
    *.sh          - 0755
    *.py          - 0755
    *.js          - 0755
  notes.md        - 0644
```

### User Isolation

- Each Linux user has their own `~/.agents/` directory
- No cross-user access unless explicitly shared
- Profile actions run as the invoking user

### Best Practices

1. **Separate users**: Run different profiles as different users if needed
2. **Set umask**: Ensure `umask 077` for secure default permissions
3. **Review ownership**: Ensure profile directories are owned by correct user
4. **Don't share secrets**: Never copy secrets.env between users

## Auditing and Logging

### Command Execution

APS logs command execution:

```bash
aps myagent -- some-command
```

Logs include:
- Profile ID
- Command executed
- Exit code
- Timestamp

### Webhook Events

Webhook server logs:
- Event type
- Delivery ID
- Source IP
- Selected profile/action
- Execution result

### Review Logs

**Systemd logs:**
```bash
journalctl -u aps-webhook -f
```

**Direct output:**
```bash
aps webhook serve 2>&1 | tee webhook.log
```

### Best Practices

1. **Log all webhook events**: Track all incoming requests
2. **Monitor failures**: Watch for failed executions
3. **Audit regularly**: Review logs for suspicious activity
4. **Secure logs**: Protect log files from unauthorized access
5. **Rotate logs**: Implement log rotation for long-running servers

## Common Security Mistakes

### ❌ Don't

1. **Commit secrets.env to version control**
2. **Set secrets.env to 0644 or more permissive**
3. **Use same API keys across environments**
4. **Skip webhook signature validation**
5. **Expose webhook server on public internet without auth**
6. **Log secret values**
7. **Share SSH keys between profiles**
8. **Run with unnecessary privileges**

### ✅ Do

1. **Add secrets.env to .gitignore**
2. **Set secrets.env to 0600**
3. **Use environment-specific profiles**
4. **Always use webhook secrets**
5. **Use reverse proxy with HTTPS**
6. **Redact secret values in output**
7. **Use dedicated SSH keys per profile**
8. **Run with minimal required privileges**

## Vulnerability Reporting

If you discover a security vulnerability, please:

1. **Don't**: Create a public issue
2. **Do**: Send details to the maintainers privately
3. **Include**: Steps to reproduce, expected vs actual behavior
4. **Wait**: For a fix before disclosing publicly

## Additional Resources

- [OWASP Security Guidelines](https://owasp.org/www-community/)
- [GitHub Security Best Practices](https://docs.github.com/en/security)
- [SSH Key Management](https://www.ssh.com/academy/ssh/key)
- [Webhook Security](https://webhook.site/blog/2018/08/08/secure-webhooks/)
