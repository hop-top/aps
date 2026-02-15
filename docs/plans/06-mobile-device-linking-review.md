# UX Review: Plan 6 — Mobile Device Linking via QR Code

**Reviewed**: 2026-02-12
**Scope**: CLI/TUI UX gaps, improvements, and decisions
**Skills Applied**: cli-design, tui-design
**Reference**: Existing APS CLI patterns (Cobra + Lipgloss + tabwriter)

---

## 1. GAPS — Missing UX Concerns

### 1.1 QR Code Terminal Display — The Hero Moment

The plan says "QR Code (in terminal as ASCII art or PNG saved to
file)" but specifies nothing about the rendering. This is the single
most important visual moment in the entire feature. A QR code that
renders wrong, gets clipped, or is unreadable kills the flow.

**Why it matters:** QR codes require high contrast (black on white),
a minimum module size (typically 2 terminal characters per module),
and sufficient quiet zone (4-module white border). A standard QR
Version 5 (37x37 modules) needs at minimum 74 columns and ~40 rows
of terminal space. Many terminals default to 80 columns; SSH
sessions may be smaller.

**Recommendation:**

1. Detect terminal size before rendering. If too small, degrade:

```
$ aps device link --profile=myagent

  Terminal too narrow for QR display (need 78 cols, have 60).

  Alternatives:
    aps device link --qr-output=qr.png   Save as image
    aps device link --code-only           Show pairing code only

  Or resize your terminal and re-run.
```

2. Use Unicode half-blocks (upper/lower) to double vertical
   density. Each "pixel" becomes one character instead of two lines.
   This halves the required terminal height.

3. Force white background behind QR, regardless of terminal theme.
   Use `lipgloss` background color to ensure contrast on dark
   terminals.

4. Show the QR code with surrounding context:

```
$ aps device link --profile=myagent

  Scan with APS mobile app

  [==============================================]
  [  ██ ▄▄▄▄▄ █▄ █▄█ ██▄  █  █▀█ █ ▄▄▄▄▄ ██  ]
  [  ██ █   █ █▀▄   ▄ █▀█▀▀ ▀██  █ █   █ ██  ]
  [  ██ █▄▄▄█ █▀██▄█▀██ ▀▀█▄ █   █ █▄▄▄█ ██  ]
  [  ██▄▄▄▄▄▄▄█▄▀ ▀▄█ █ █▄█▄█▄█▄█▄▄▄▄▄▄▄██  ]
  [  ... (rest of QR)                          ]
  [==============================================]

  Or enter pairing code manually:  ABC-123-XYZ

  Profile:   myagent
  Endpoint:  https://192.168.1.42:8443/aps/device
  Expires:   in 15 minutes

  Waiting for device...  (Ctrl+C to cancel)
```

5. Add `--no-qr` flag to skip QR rendering entirely (accessibility,
   screen readers, CI environments).

---

### 1.2 The Waiting State — `link` Is a Long-Running Command

The plan describes `aps device link` as generating a QR code but
never addresses what happens AFTER the QR displays. The mobile
device needs to scan, POST to /device/pair, receive a token, and
connect via WebSocket. The CLI must show this entire flow in
real-time.

**Why it matters:** Without waiting feedback, the user sees a QR
code and has no idea if anything is working. This is a two-device
flow where the CLI is one half and the phone is the other half.

**Recommendation:** Use a BubbleTea TUI for the waiting state:

```
  Scan with APS mobile app

  [QR CODE HERE]

  Pairing code:  ABC-123-XYZ
  Expires:       in 14m 32s (countdown)

  Status:  Waiting for device...

  Press Ctrl+C to cancel. Press R to regenerate QR.
```

On scan detected:

```
  Status:  Device detected: Alice's iPhone 15 (iOS 17.2)
           Pairing...
```

On pairing complete:

```
  Status:  Paired successfully!

  Device:    Alice's iPhone 15
  Device ID: iphone-alice-001
  Connected: WebSocket active
  Expires:   Feb 26, 2026

  The device can now send commands to profile 'myagent'.
  Run 'aps device list' to manage linked devices.
```

On timeout:

```
  Status:  QR code expired.

  No device scanned within 15 minutes.

  To generate a new QR code:
    aps device link --profile=myagent
```

Key design points:
- Live countdown timer for expiry (BubbleTea tick)
- SIGINT handler: clean up pairing code, stop server if transient
- The command exits on success OR timeout OR Ctrl+C
- `R` key regenerates QR without restarting command
- Non-TTY mode: print QR, print pairing info, block until paired
  or timeout, exit with code 0 (paired) or 1 (timeout)

---

### 1.3 Network Reachability — localhost Is Unreachable from Mobile

The plan hardcodes `https://localhost:8443/aps/device` as the
endpoint in the QR payload. A mobile device on the same WiFi
network cannot reach `localhost` on another machine. The plan
completely glosses over this.

**Why it matters:** This is a showstopper. If the phone cannot
reach the endpoint, the entire feature fails silently. The user
scans the QR, the phone tries to connect, gets a timeout, and the
CLI shows "Waiting for device..." forever.

**Recommendation:** The endpoint in the QR payload must be the
host machine's LAN IP (or mDNS name, or tunnel URL). The CLI
must auto-detect and display this clearly:

```
  Endpoint:  https://192.168.1.42:8443/aps/device
             (your-mac.local:8443 via mDNS)

  Ensure your mobile device is on the same network.
```

If multiple network interfaces exist, prompt or use heuristics:

```
  Multiple network interfaces detected:
    1. en0 (WiFi)     192.168.1.42
    2. en8 (Ethernet)  10.0.0.15
    3. utun3 (VPN)    172.16.0.5

  Using: 192.168.1.42 (WiFi)
  Override with: --bind-addr=10.0.0.15
```

For remote access (not on same network), the plan needs to either:
- Integrate a tunnel (like ngrok/bore/cloudflared)
- Explicitly document "same network only" as a limitation
- Provide `--tunnel` flag for remote pairing

---

### 1.4 TLS Certificate Trust — Self-Signed Certs Break Mobile

The plan says "HTTPS support with self-signed certs (profile-
specific)" but mobile operating systems (iOS and Android) reject
self-signed certificates by default. iOS requires explicit trust
profile installation. Android requires user-installed CA.

**Why it matters:** The mobile app will refuse to connect with a
TLS error. The user sees "Waiting for device..." forever on the
CLI side and a certificate error on the phone.

**Recommendation:** Three strategies, in order of UX quality:

1. **Pairing-time trust**: Include the cert fingerprint in the QR
   payload. The mobile SDK does certificate pinning using the
   fingerprint, bypassing system trust. The QR payload becomes:

```json
{
  "version": "1.0",
  "endpoint": "https://192.168.1.42:8443/aps/device",
  "pairing_code": "ABC-123-XYZ",
  "cert_fingerprint": "sha256:ab12cd34...",
  ...
}
```

2. **Let's Encrypt with tunnel**: If `--tunnel` is used, the
   tunnel provider handles TLS with a trusted cert.

3. **Manual trust**: Document how to install the CA on iOS/
   Android. This is the worst UX but simplest implementation.

The plan must choose one. Option 1 is the lean recommendation.

---

### 1.5 No Output Mode Matrix for Device Commands

Consistent with Gap 1.1 from Plan 1 review. None of the five
device commands define behavior for `--json`, `--quiet`, pipe
detection, or `NO_COLOR`.

**Why it matters:** `aps device list --json` is essential for
automation (monitoring scripts, dashboards). `aps device logs`
piped to `grep` must suppress ANSI.

**Recommendation:**

| Command   | TTY          | `--json`      | `--quiet` | Pipe     |
|-----------|-------------|---------------|-----------|----------|
| `link`    | QR + TUI    | JSON payload  | Code only | No QR    |
| `list`    | Styled table| JSON array    | IDs only  | Plain    |
| `revoke`  | Confirm+msg | `{"revoked"}` | Exit code | `--force`|
| `logs`    | Colored log | JSON lines    | Errors    | Plain    |
| `approve` | Confirm+msg | `{"approved"}`| Exit code | `--force`|

---

### 1.6 No Device Status TUI — Missed BubbleTea Opportunity

The plan has `aps device list` as a static table but no real-time
device monitoring. With WebSocket connections, device health, and
live command execution, this is a prime TUI surface.

**Why it matters:** An operator managing 5+ devices needs a
dashboard, not repeated `aps device list` invocations.

**Recommendation:** Add `aps device status --watch` or
`aps device monitor`:

```
  Device Monitor — myagent                    [q]uit

  DEVICE              STATUS    LAST SEEN    COMMANDS
  iphone-alice-001    online    just now     echo hello
  android-bob-001     online    2m ago       --
  ipad-carol-003      offline   1h ago       --

  Connections: 2 active / 3 registered
  Commands executed: 47 today
  Next expiry: iphone-alice-001 in 12 days

  [a]pprove pending  [r]evoke device  [l]ogs
```

Use BubbleTea for:
- Real-time status updates via internal channel
- Keyboard navigation (j/k or arrow keys)
- Action hotkeys (r to revoke selected, l to view logs)
- Auto-refresh with configurable interval

---

### 1.7 Revocation of Active Devices — UX for Force Disconnect

The plan says "Immediate disconnection of WebSocket" on revoke but
does not address the user experience of revoking a device that is
currently executing a command.

**Why it matters:** Revoking mid-command could leave the remote
profile in an inconsistent state. The user needs to understand the
consequences.

**Recommendation:**

```
$ aps device revoke iphone-alice-001 --profile=myagent

  Device: Alice's iPhone 15
  Status: online (connected 2h ago)
  Active: 1 running command (echo "long task")

  Revoking will:
    - Disconnect the active WebSocket immediately
    - Terminate 1 running command
    - Blacklist the device token
    - The device must re-pair via new QR code to reconnect

  Revoke this device? [y/N]:
```

For devices that are offline:

```
$ aps device revoke android-bob-001 --profile=myagent

  Device: Bob's Android
  Status: offline (last seen 3 days ago)

  Revoking will:
    - Blacklist the device token
    - Reject future connection attempts
    - The device must re-pair via new QR code

  Revoke this device? [y/N]:
```

Add `--dry-run` to show what would happen without acting.

---

### 1.8 Approval Workflow TUI — Undefined Interactive Surface

The plan mentions "TUI approval interface" but provides zero
detail. The approval workflow is a critical security gate for
high-security profiles.

**Why it matters:** If `approval_required=true`, a pending device
is stuck until someone runs `aps device approve`. The plan does
not define how the user is notified of pending devices, or what
the approval TUI looks like.

**Recommendation:**

Notification on pending device (if CLI is running):

```
  ! New device pending approval for profile 'myagent':
    Device: Alice's iPhone 15 (iOS 17.2)
    Requested: just now

    aps device approve iphone-alice-001 --profile=myagent
    aps device reject iphone-alice-001 --profile=myagent
```

Add `aps device pending` to list pending approvals:

```
$ aps device pending --profile=myagent

  Pending Approvals

  DEVICE              REQUESTED     DEVICE INFO
  iphone-alice-001    2 min ago     Alice's iPhone 15, iOS 17.2
  android-bob-001     15 min ago    Bob's Pixel 8, Android 14

  2 devices pending approval.

  Approve: aps device approve <device-id>
  Reject:  aps device reject <device-id>
  Approve all: aps device approve --all
```

Add `aps device reject` as counterpart to `approve`. The plan
only has `approve` but rejection is equally needed.

---

### 1.9 Error States — Critical Failures Undefined

The plan lists no error messages for common failure modes.

**Why it matters:** Users will hit these errors frequently during
initial setup. Unclear errors cause abandonment.

**Recommendation:** Define error messages for each failure:

**Port in use:**
```
Error: Cannot start device server on port 8443
  Port is already in use (PID 12345: aps device link)

  Options:
    aps device link --port=8444           Use different port
    kill 12345                            Stop existing server
    aps device list                       Check active links

Exit code: 5
```

**Device config not enabled:**
```
Error: Device linking not enabled for profile 'myagent'

  Enable it:
    aps profile edit myagent
    # Add: device_config.enabled = true

  Or use:
    aps device link --profile=myagent --enable

Exit code: 2
```

**Max devices reached:**
```
Error: Maximum devices reached for profile 'myagent' (10/10)

  Currently linked devices:
    iphone-alice-001    active    expires Feb 26
    android-bob-001     active    expires Feb 19
    ...

  Revoke a device first:
    aps device revoke <device-id> --profile=myagent

  Or increase the limit:
    aps profile edit myagent
    # Change: device_config.max_devices = 20

Exit code: 4
```

**No profile specified and no default:**
```
Error: No profile specified

  Use --profile=<name> or set a default:
    aps profile default myagent

  Available profiles:
    myagent      3 devices
    work-agent   0 devices

Exit code: 2
```

---

### 1.10 No `--follow` for Device Logs

The existing `session logs` has `--follow` (`-f`). The plan's
`device logs` has `--tail=100` but no follow mode.

**Why it matters:** Real-time log tailing is essential for
debugging device connections. Without it, users resort to
`watch aps device logs`.

**Recommendation:** Add `--follow` / `-f` flag matching the
`session logs` pattern:

```
$ aps device logs --profile=myagent --device=iphone-alice-001 -f

  [10:30:00] CONNECT  iphone-alice-001  session=sess_123
  [10:31:00] EXECUTE  command="echo hello"  status=running
  [10:31:01] OUTPUT   stdout: Hello from mobile
  [10:31:01] COMPLETE exit_code=0  duration=250ms
  [10:35:00] HEARTBEAT  latency=12ms
  ...
  (following — Ctrl+C to stop)
```

Use structured log format with color-coded log levels
(CONNECT=green, EXECUTE=blue, ERROR=red, HEARTBEAT=dim).

---

### 1.11 Shell Completion for Device IDs

Consistent with Gap 1.5 from Plan 1 review. `aps device revoke`,
`aps device approve`, and `aps device logs --device=` all take
device IDs as arguments but no completion is defined.

**Why it matters:** Device IDs like `iphone-alice-001` are not
memorable. Tab completion prevents typos and saves time.

**Recommendation:** Register `ValidArgsFunction` on all commands
that take device IDs. Source completions from registry.json.
Include profile-scoped filtering when `--profile` is set.

---

### 1.12 No Signal Handling for Device Server

`aps device link` starts an HTTP/WebSocket server. SIGINT/SIGTERM
handling is unspecified.

**Why it matters:** An unclean shutdown can leave orphaned pairing
codes, dangling WebSocket connections, or locked ports.

**Recommendation:**
- SIGINT (Ctrl+C): graceful shutdown. Close WebSocket connections
  with close frame, invalidate pending pairing codes, release
  port. Show: "Shutting down... done."
- SIGTERM: same as SIGINT.
- Double SIGINT: force quit immediately.
- Clean up on timeout expiry (no device connected).

---

## 2. IMPROVEMENTS — Partially Covered, Needs More Detail

### 2.1 Verb Consistency: `link` vs `pair` vs `connect`

The plan uses `aps device link` but the internal flow is "pairing."
Plan 5 defines `aps device create/start/stop`. The verb `link`
is not used elsewhere in the APS CLI.

The capability system uses `link` to mean "symlink a capability
into a profile" (`aps cap link`). Reusing `link` for device
pairing creates semantic overload.

**Recommendation:** Consider `aps device pair` instead:

```
aps device pair --profile=myagent        # generate QR, wait
aps device list --profile=myagent        # list paired devices
aps device revoke iphone-alice-001       # unpair
aps device approve iphone-alice-001      # approve pending
```

`pair` matches the domain language (QR pairing, Bluetooth
pairing) and avoids collision with `cap link`.

---

### 2.2 Pairing Code Format — Human Readability

The plan shows `ABC123XYZ789` as a pairing code. This is a 12-char
alphanumeric string with no separators, which is hard to read and
harder to type manually.

**Recommendation:** Use grouped format with dashes:

```
  Pairing code:  ABC-123-XYZ
```

Design constraints:
- Groups of 3-4 chars separated by dashes
- Avoid ambiguous characters (0/O, 1/l/I)
- Total entropy still >128 bits (use longer code if needed)
- Displayed in monospace with generous spacing
- Copyable (when terminal supports OSC 52)

---

### 2.3 `device list` Table Needs Status Indicators

The plan shows a plain table. The existing APS CLI uses
`StatusDot()` for visual state indication. Device status
deserves the same treatment.

**Recommendation:**

```
$ aps device list --profile=myagent

  Linked Devices

  DEVICE              NAME                  OS       STATUS     EXPIRES
  iphone-alice-001    Alice's iPhone 15     iOS      * online   12 days
  android-bob-001     Bob's Android         Android  * online   5 days
  ipad-carol-003      Carol's iPad          iOS      o offline  1 day

  3 devices (2 online, 1 offline)
```

Where `*` is `StatusDot(true)` (green filled) and `o` is
`StatusDot(false)` (grey hollow). Add red dot for `revoked`.

Also add `--status` filter:
```
aps device list --status=online
aps device list --status=expired
```

---

### 2.4 Expiry Display — Relative vs Absolute

The plan shows `expires_at` as ISO timestamp in the table. Humans
read relative times better.

**Recommendation:** Default to relative, `--absolute` for ISO:

```
  Default:    12 days        (relative)
  Imminent:   2 hours        (orange, use Warn style)
  Expired:    expired 3h ago (red, use Error style)
  --absolute: 2026-02-26     (ISO date)
```

---

### 2.5 The `--capabilities` Flag Needs Discoverability

The plan shows `--capabilities=run:stateless,run:streaming,...`
but never defines how a user discovers valid capability strings.

**Recommendation:** Add capability listing and validation:

```
$ aps device link --capabilities=?

  Available device capabilities:

  CAPABILITY          DESCRIPTION
  run:stateless       Execute one-shot commands
  run:streaming       Execute with streaming output
  monitor:sessions    View active sessions
  monitor:logs        View profile logs (read-only)

  Default: run:stateless, run:streaming, monitor:sessions

  Usage:
    aps device link --capabilities=run:stateless
    aps device link --capabilities=all
```

Invalid capability names should produce a "did you mean?"
suggestion, matching the existing APS error pattern.

---

### 2.6 QR Code Expiry vs Device Token Expiry — Confusing Double Expiry

The plan has two separate expiry concepts:
1. QR code / pairing code expiry (presumably short, ~15 minutes)
2. Device token expiry (14 days default)

These are never clearly distinguished in the CLI output.

**Recommendation:** Make both visible and distinct:

```
  QR code expires in: 14m 32s (scan before this)
  Device access: 14 days after pairing
```

Use separate `--qr-expires` and `--expires` flags:

```
aps device link --qr-expires=30m --expires=30d
```

Default QR expiry should be short (15 minutes) for security.
Default device expiry is 14 days per plan.

---

### 2.7 Device ID Generation — User-Hostile Identifiers

The plan shows `iphone-alice-001` as a device ID, implying human-
friendly IDs. But the implementation uses `google/uuid`, which
generates UUIDs. These two approaches conflict.

**Recommendation:** Use human-friendly IDs derived from device
info, with UUID as internal key:

```
  Device ID:   alice-iphone15
  Internal ID: a1b2c3d4-e5f6-...  (hidden unless --verbose)
```

Or let user provide a name at pairing time (via the mobile app),
with auto-generated fallback: `iphone-17.2-a1b2`.

---

## 3. QUESTIONS / DECISIONS — Below 85% Confidence

### 3.1 Should `aps device link` start a persistent server?

The plan implies a transient server (start on `link`, stop when
paired). But Plan 5's DeviceCapability has Start/Stop lifecycle,
implying a persistent server.

| Option | Description | Tradeoff |
|--------|-------------|----------|
| A | Transient (link only) | Simple; port free when not pairing. No background process. But paired devices cannot reconnect after CLI exits. |
| B | Persistent (always on) | Devices stay connected. Requires `aps device serve` or background daemon. More complex. |
| **C (lean)** | **`link` starts server if not running; server persists until all devices disconnect or explicit `aps device stop`** | **Best UX: user runs `link`, server starts, stays alive. No orphan server after all devices gone.** |

---

### 3.2 Same-network only or support remote pairing?

| Option | Description | Tradeoff |
|--------|-------------|----------|
| A | Same network only | Simple. mDNS/LAN IP. Covers home/office use. Fails for remote/cloud setups. |
| **B (lean)** | **Same network default + `--tunnel` for remote** | **Cover 90% case simply. `--tunnel` uses cloudflared/bore for remote. Explicit opt-in for complexity.** |
| C | Always tunnel | Every pairing goes through a relay. Adds latency, dependency, and privacy concern. |

---

### 3.3 How should the device server handle multiple profiles?

The plan configures per-profile ports. Running 5 profiles means
5 ports.

| Option | Description | Tradeoff |
|--------|-------------|----------|
| A | One port per profile | Simple isolation. Port exhaustion risk. Firewall headache. |
| **B (lean)** | **Single shared port, profile routing via path** | **One server at `:8443`, routes `/aps/device/<profile-id>/...`. Single port to manage/expose.** |
| C | Unix socket per profile, reverse proxy | Maximum isolation. Complex setup. |

---

### 3.4 Should `aps device revoke` support `--all`?

| Option | Description | Tradeoff |
|--------|-------------|----------|
| A | Single device only | Safe. Tedious for bulk revocation. |
| **B (lean)** | **Support `--all` with double confirmation** | **"Revoke ALL 5 devices for profile 'myagent'? Type 'myagent' to confirm:"** |
| C | Support glob patterns | `aps device revoke 'iphone-*'`. Powerful but overly complex for v1. |

---

### 3.5 BubbleTea TUI for the link flow or plain CLI?

| Option | Description | Tradeoff |
|--------|-------------|----------|
| A | Plain CLI (print and block) | Simple. No interactivity. Cannot refresh QR or show countdown. |
| **B (lean)** | **BubbleTea TUI with graceful fallback** | **Rich experience: countdown, status updates, key bindings. Falls back to plain mode if non-TTY or `--no-tui`.** |
| C | Always plain, separate `device monitor` TUI | Separates concerns but misses the pairing UX moment. |

---

### 3.6 What happens to `aps device` verb overlap with Plan 5?

Plan 5 defines: `aps device list/create/start/stop/status`
Plan 6 adds: `aps device link/revoke/approve/logs`

These share the `device` namespace but have different semantics.
Plan 5's `device` is about device types (mobile, messenger,
protocol). Plan 6's `device` is about specific mobile device
instances.

| Option | Description | Tradeoff |
|--------|-------------|----------|
| A | Merge into single `device` namespace | All subcommands under `aps device`. Could be confusing: `device create` (Plan 5) vs `device link` (Plan 6). |
| **B (lean)** | **Plan 5 = `aps device`, Plan 6 = subcommands filtered by type** | **`aps device list --type=mobile` shows Plan 6 devices. `aps device pair` is mobile-specific. Clear hierarchy.** |
| C | Separate namespace (`aps mobile`) | Clean separation but fragments the device concept. |

---

### 3.7 Token refresh UX — transparent or explicit?

The plan mentions token refresh but not the user experience.

| Option | Description | Tradeoff |
|--------|-------------|----------|
| **A (lean)** | **Transparent refresh by mobile SDK** | **SDK refreshes token before expiry. User never sees it. CLI shows new expiry in `device list`.** |
| B | CLI notification on refresh | "Device iphone-alice-001 token refreshed, new expiry: Mar 12". Noisy. |
| C | Manual refresh required | User runs command to extend. Poor UX. |

---

## Cross-Plan Consistency Issues

### C1. `device` Namespace Collision (Plan 5 vs Plan 6)

Plan 5 treats `device` as a capability type with generic
lifecycle (create/start/stop). Plan 6 treats `device` as a
mobile pairing system with its own lifecycle (link/revoke/
approve). These must be reconciled before implementation.
See Decision 3.6.

### C2. Verb Vocabulary Drift

Plans 1-3 established: create/list/inspect/archive/delete.
Existing APS: new/list/show/delete.
Plan 5 adds: create/start/stop/status.
Plan 6 adds: link/revoke/approve/logs.

`revoke` is new; the equivalent in existing APS is `delete`.
`approve` is new; no prior equivalent. `link` collides with
`cap link`. This needs a verb audit across all plans.

### C3. `--profile` Flag Inconsistency

Some commands require `--profile`, some use a default profile.
The profile resolution order is never defined across plans:

```
1. --profile=<name>          (explicit flag)
2. APS_PROFILE env var       (environment)
3. Default profile setting   (config)
4. Error if ambiguous        (fail safe)
```

This must be standardized. Plan 6 is especially sensitive
because device tokens are profile-scoped.

### C4. Log Command Pattern

`session logs` uses `--follow`/`-f` and `--tail`.
Plan 6's `device logs` uses `--tail=100` only.
These should share the same flag vocabulary and behavior.

### C5. Destructive Op Pattern

`session delete` uses confirmation + `--force`.
Plan 6's `device revoke` needs the same pattern but also
must communicate active connection consequences (Gap 1.7).
Add `--dry-run` consistently across all destructive ops.

---

## Highest-Priority Items

| # | Finding | Type | Impact |
|---|---------|------|--------|
| 1 | Network reachability (localhost unreachable from mobile) | GAP 1.3 | Showstopper — feature cannot work |
| 2 | TLS trust (self-signed certs rejected by mobile OS) | GAP 1.4 | Showstopper — connection fails |
| 3 | QR code terminal rendering (contrast, size, fallback) | GAP 1.1 | Hero moment — first impression |
| 4 | Waiting state UX (post-QR, pre-pair feedback) | GAP 1.2 | Core flow — user sees nothing |
| 5 | Device namespace collision (Plan 5 vs Plan 6) | CROSS C1 | Architecture — must resolve first |
| 6 | Error states (port, config, max devices) | GAP 1.9 | Setup failures cause abandonment |
| 7 | Output mode matrix | GAP 1.5 | Automation and scripting broken |
| 8 | Approval workflow + reject command | GAP 1.8 | Security flow incomplete |
| 9 | Real-time device monitor TUI | GAP 1.6 | Operational visibility |
| 10 | Verb collision (`link` vs `cap link`) | IMP 2.1 | Semantic confusion |
