# APS Cheatsheet — Human

Quick reference for daily use. Scannable in 30 seconds.

---

## Start

```bash
aps version                          # verify install
aps profile list                     # see available profiles
```

Config: `~/.config/aps/` (XDG); `$APS_DATA_PATH` for data

---

## Profiles

```bash
aps profile create <name>               # create profile
aps profile create <n> --email <addr>   # create with email
aps profile list                     # list all
aps profile show <name>              # inspect details
aps profile status <name>            # bundle resolution status
aps profile workspace set <n> <ws>   # link to workspace
aps profile share <name>             # export shareable bundle
aps profile import <file>            # import shared bundle
```

---

## Run Commands

```bash
aps run <profile> -- <cmd> [args]    # run cmd under profile context
aps env <profile>                    # print env vars for profile
aps alias                            # generate shell aliases for profiles
```

---

## Sessions

```bash
aps session list                     # active sessions
aps session inspect <id>             # session details
aps session attach <id>              # attach to running session
aps session detach <id>              # detach
aps session logs <id>                # tmux capture logs
aps session terminate <id>           # graceful stop
aps session delete <id>              # delete session record
```

---

## Actions

```bash
aps action list <profile>            # list profile actions
aps action show <profile> <action>   # show action details
aps action run <profile> <action>    # execute action
```

---

## Capabilities

```bash
aps capability list                  # builtin + external
aps capability show <name>           # details
aps capability install <src>         # install from dir or URL
aps capability adopt <path>          # move file → APS + symlink back
aps capability watch <path>          # watch external file (symlink into APS)
aps capability link <name> <target>  # symlink cap to path
aps capability enable <profile> <n>  # enable on profile
aps capability disable <profile> <n> # disable on profile
aps capability delete <name>         # remove capability
aps capability patterns              # show smart patterns + builtins
```

---

## Bundles

```bash
aps bundle list                      # builtin + user bundles
aps bundle show <name>               # full YAML definition
aps bundle create <name>             # scaffold new bundle
aps bundle edit <name>               # open in $EDITOR
aps bundle validate <file>           # validate YAML
aps bundle delete <name>             # delete user-defined bundle
```

---

## Workspaces

```bash
aps workspace sync                   # sync workspace state
aps workspace activity               # activity log
```

---

## Squads

```bash
aps squad list                       # all squads
aps squad show <id>                  # details
aps squad create <name>              # new squad
aps squad members add <id> <profile>  # add member
aps squad members remove <id> <p>     # remove member
aps squad check <id>                 # validate topology (8-item checklist)
aps squad delete <id>                # delete squad
```

---

## Collaboration

```bash
aps workspace use <workspace>           # set active workspace (persists)
aps workspace list                      # list workspaces
aps workspace new <name>                # create workspace
aps workspace show <ws>                 # details
aps workspace join <ws>                 # join as agent
aps workspace leave <ws>                # leave
aps workspace members                   # list members (uses active ws)
aps workspace agents --cap <capability> # find agents by capability
aps workspace send <recipient> \
  --action <act> --set key=val       # send task to agent

# Context (shared key-value)
aps workspace ctx list                  # all context vars
aps workspace ctx set <key> <val>       # set variable
aps workspace ctx get <key>             # get variable
aps workspace ctx history <key>         # mutation history
aps workspace ctx delete <key>          # delete variable

# Tasks
aps workspace tasks                     # list workspace tasks
aps workspace task <id>                 # task details

# Conflicts & Policies
aps workspace conflicts                 # list conflicts
aps workspace resolve <id>              # resolve conflict
aps workspace policy <ws> set <mode>    # set conflict resolution policy

# Audit & Archive
aps workspace audit                     # audit trail
aps workspace caps                      # capabilities in workspace
aps workspace archive <ws>              # archive workspace
```

---

## Adapters & Messengers

```bash
aps adapter list                     # all devices
aps adapter create <name>            # new device
aps adapter start <id>               # start device
aps adapter stop <id>                # stop device
aps adapter status <id>              # device status
aps adapter link <id> <profile>      # link to profile
aps adapter unlink <id> <profile>    # unlink
aps adapter attach <id> <workspace>  # attach to workspace
aps adapter detach <id> <workspace>  # detach
aps adapter test <id>                # test messenger pipeline
aps adapter logs <id>                # view device logs
aps adapter channels <id>            # list known channels

# Script adapter actions (email, etc.)
aps adapter exec email send \
  --profile noor \
  --input to=user@example.com \
  --input subject="Hello" \
  --input body="Message"            # send email as profile
aps adapter exec email list \
  --profile noor                    # list inbox
aps adapter exec email read \
  --profile noor --input id=7131    # read message
aps adapter exec email reply \
  --profile noor \
  --input id=7131 \
  --input body="Thanks!"           # reply as profile

# Mobile pairing
aps adapter pair <id>                # QR code for mobile pairing
aps adapter pending                  # list pending mobile devices
aps adapter approve <id>             # approve pending device
aps adapter reject <id>              # reject pending device
aps adapter revoke <id>              # revoke paired device
aps adapter permissions set <id>     # set workspace permissions
```

---

## Protocols (A2A / ACP)

```bash
# A2A (Agent-to-Agent)
aps a2a toggle <profile> [on|off]    # enable/disable A2A
aps a2a server <profile>             # start A2A server
aps a2a card show <profile>          # show agent card
aps a2a card fetch <url>             # fetch agent card from URL
aps a2a tasks send <profile> \
  --to <url> --msg "..."             # send task
aps a2a tasks show <id>                # task details
aps a2a tasks list <profile>         # list tasks
aps a2a tasks cancel <id>             # cancel task
aps a2a tasks subscribe <id>          # push notifications for task

# ACP (editor integration)
aps acp toggle <profile> [on|off]    # enable/disable ACP
aps acp server <profile>             # start ACP server
```

---

## HTTP API Server

```bash
aps serve                            # start REST API (default :8080)
aps serve --addr :9000               # custom address
aps serve --auth-token <tok>         # require bearer token
aps serve --log-level debug          # verbose logging
```

---

## Identity (DID)

```bash
aps identity init <profile>          # generate DID + Ed25519 key pair
aps identity show <profile>          # show DID + identity
aps identity verify <did>            # verify + resolve DID
aps identity badge issue <profile>   # issue verifiable credential
aps identity badge list <profile>    # list badges
aps identity badge verify <badge>    # verify badge
```

---

## Directory (AGNTCY)

```bash
aps directory register <profile>     # register for discovery
aps directory delete <profile>   # remove from directory
aps directory show <profile>         # OASF record
aps directory discover --cap <cap>   # find agents by capability
```

---

## Access Policies

```bash
aps policy list <workspace>          # list policies
aps policy show <workspace>          # effective policy
aps policy set <ws> allow-all        # all linked devices (default)
aps policy set <ws> allow-list       # whitelist mode
aps policy set <ws> deny-list        # blacklist mode
aps policy trust                     # manage inbound trust verification
```

---

## Voice

```bash
aps voice start <profile>            # start voice session
aps voice session list               # active voice sessions
aps voice service start              # start backend service
aps voice service stop               # stop backend service
aps voice service status             # service health
```

---

## Webhooks

```bash
aps webhook toggle <profile> [on|off]  # enable/disable
aps webhook server <profile>           # start webhook server
```

---

## Observability & Audit

```bash
aps observability                    # OpenTelemetry config
aps workspace audit                # workspace access audit log
aps workspace conflicts list <workspace>        # list conflicts
aps workspace conflicts resolve <id>            # resolve conflict
```

---

## Misc

```bash
aps docs                             # generate documentation
aps alias                            # shell aliases for profiles
aps completion <shell>               # shell completion script
aps migrate                          # migrate legacy configs
aps upgrade                          # check + install updates
aps version                          # version info
```

---

## Common Tips and Failure Modes

| Symptom | Fix |
|---------|-----|
| Profile not found | `aps profile list`; check `$APS_DATA_PATH` |
| Session orphaned | `aps session terminate <id>` then `delete <id>` |
| Capability missing | `aps capability list`; install with `aps capability install` |
| A2A task stuck | `aps a2a tasks show <id>`; cancel + retry |
| Messenger not receiving | `aps adapter test <id>`; check `aps adapter logs <id>` |
| Wrong workspace active | `aps workspace use <workspace>` |
| Conflict blocks collab | `aps workspace conflicts`; then `aps workspace resolve <id>` |
| Serve won't start | Check port with `lsof -i :8080`; use `--addr` for alt port |
