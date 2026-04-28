# Remote Access: Exposing an Agent Profile

Guide for making an APS agent profile reachable from outside your local network.
Two supported paths: **Tailscale** (private mesh) or **Cloudflare Tunnel**
(public ingress). Pick by audience.

## When to use which

| Use case                                            | Choice       |
| --------------------------------------------------- | ------------ |
| You + your own devices                              | Tailscale    |
| Agent-to-agent across your machines                 | Tailscale    |
| Trusted collaborator, identity-gated                | Either       |
| Public webhooks (Discord, Telegram, GitHub)         | Cloudflared  |
| Browser demo, no client install                     | Cloudflared  |
| Both personal access AND public webhooks            | Run both     |

Neither replaces app-layer auth. Gate at both tunnel and app.

---

## Prerequisites

- Profile with A2A or messenger listener enabled. See
  [a2a-quickstart.md](a2a-quickstart.md) and [messengers.md](messengers.md).
- Know the profile's `listen_addr` and port.
- For Tailscale: bind to `0.0.0.0` or the `tailscale0` interface, NOT
  `127.0.0.1`. For cloudflared: `127.0.0.1` is fine (it dials out).

Edit `<data>/profiles/<name>/profile.yaml`:

```yaml
a2a:
  enabled: true
  listen_addr: "0.0.0.0:8081"        # tailscale
  # listen_addr: "127.0.0.1:8081"    # cloudflared
  public_endpoint: "https://agent.example.com"
```

---

## Option A: Tailscale

Private mesh. No public surface. Every client runs Tailscale.

### Setup

```bash
brew install tailscale
sudo tailscaled install-system-daemon
tailscale up
```

### Access

Direct via MagicDNS:

```
http://<host>.<tailnet>.ts.net:8081
```

Or front it with `tailscale serve` for auto-TLS, no port, no firewall:

```bash
tailscale serve https / http://localhost:8081
```

Then: `https://<host>.<tailnet>.ts.net`

### Hardening

- ACLs in admin console — restrict which nodes can reach the profile port.
- Tag service identities (`tag:agent`) if agents run headless.
- Keep `public_endpoint` in `profile.yaml` set to the `ts.net` URL so A2A
  agent cards advertise the right address.

---

## Option B: Cloudflare Tunnel

Public DNS + TLS. No open ports. Requires a domain on Cloudflare.

### Setup

```bash
brew install cloudflared
cloudflared tunnel login
cloudflared tunnel create aps-profile
```

Create `~/.cloudflared/config.yml`:

```yaml
tunnel: <tunnel-id>
credentials-file: ~/.cloudflared/<tunnel-id>.json
ingress:
  - hostname: agent.example.com
    service: http://localhost:8081
  - service: http_status:404
```

Route DNS and run:

```bash
cloudflared tunnel route dns aps-profile agent.example.com
cloudflared tunnel run aps-profile
# Or install as service:
sudo cloudflared service install
```

### Gate it

Tunnels are public by default. Put Cloudflare Access in front:

1. Zero Trust dashboard → Access → Applications → Add self-hosted
2. Hostname: `agent.example.com`
3. Policy: email domain, GitHub OAuth, or service token for machine agents

For webhooks that cannot authenticate via Access (Discord, Telegram):

- Add a path-scoped Access bypass for the webhook endpoint only
- Verify HMAC signatures in-app — never rely on obscurity
- See [messengers.md](messengers.md) for per-platform signature fields

### Profile alignment

```yaml
a2a:
  public_endpoint: "https://agent.example.com"
```

This is what A2A agent cards publish. Must match the cloudflared hostname
or peers will fail discovery.

---

## Verification

```bash
# Tailscale
curl http://<host>.<tailnet>.ts.net:8081/.well-known/agent.json

# Cloudflared
curl https://agent.example.com/.well-known/agent.json
```

Expect the profile's A2A agent card JSON. If 404, check ingress/listen_addr.
If 530/timeout on cloudflared, check `cloudflared tunnel info <id>` for
active connections.

---

## Checklist

- [ ] `listen_addr` bound to correct interface
- [ ] `public_endpoint` matches the external URL
- [ ] Tunnel (TS or CF) running as service, not foreground shell
- [ ] Access policy or ACL configured — not relying on obscurity
- [ ] App-layer auth on top (HMAC, tokens, Access JWT)
- [ ] Agent card served at `/.well-known/agent.json` reachable externally

---

See also: [a2a-quickstart.md](a2a-quickstart.md),
[messengers.md](messengers.md).
