---
title: Tutorial
---

**What you will learn**: How to create your first APS profile,
start a session, inspect it, and clean up — in about five minutes.

This tutorial assumes you have completed [Installation](./install.md)
and `aps version` works.

---

## 1. Create a profile

A **profile** is the unit APS works with. It bundles an identity,
a capability set, and configuration. Create one:

```bash
aps profile create hello --display-name "Hello World"
```

You should see confirmation that the profile was created. Verify:

```bash
aps profile list
```

`hello` should appear in the list.

Inspect what APS gave it by default:

```bash
aps profile show hello
```

You will see YAML with the profile ID, display name, capabilities
(usually `a2a`, `agent-protocol`, and a few agntcy-* entries),
shell preference, git config, and isolation settings.

---

## 2. Start a session

A **session** is a running instance of a profile. Start one and
attach to it immediately:

```bash
aps hello
```

Passing a profile ID as the only argument starts (or resumes) a
session for that profile and attaches your terminal to it. You are
now inside the profile's tmux-backed session.

To detach without stopping the session, use tmux's detach chord:
**`Ctrl-b` then `d`**.

You are back at your regular shell. The session keeps running.

---

## 3. List and inspect sessions

```bash
aps session list
```

You should see your `hello` session with its status. Inspect it:

```bash
aps session inspect <session-id>
```

(`<session-id>` comes from `session list`. Tab completion helps
if you enabled completions.)

---

## 4. View logs

APS captures tmux output so you can scroll through what happened
without attaching:

```bash
aps session logs <session-id>
```

---

## 5. Reattach

Pick up where you left off:

```bash
aps session attach <session-id>
```

Or just run `aps hello` again — it resumes the existing session
for that profile.

---

## 6. Run a one-shot command under a profile

Sometimes you do not want a full interactive session — you just
want to execute something with a profile's environment and
capabilities applied:

```bash
aps run hello -- env | grep APS
```

Everything after `--` is the command to run under the profile.
The profile's capability-injected env vars will be present.

---

## 7. Clean up

Terminate the session gracefully:

```bash
aps session terminate <session-id>
```

Or delete it entirely (removes state):

```bash
aps session delete <session-id>
```

Delete the profile itself when you are done experimenting (this
removes the profile config but not any linked workspace):

```bash
# profile deletion lives under `aps profile` — check help
aps profile --help
```

---

## Where to go next

You have the mechanics. The interesting part of APS is **connecting
the profile to something** — an AI CLI, a messenger, a protocol
listener, or a remote peer.

| I want to…                                            | Go to                                   |
| ----------------------------------------------------- | --------------------------------------- |
| Expose this profile over A2A                          | [a2a-quickstart.md](../a2a-quickstart.md) |
| Expose this profile over ACP                          | [acp-quickstart.md](../acp-quickstart.md) |
| Wire it to Telegram / Discord / Slack / GitHub / email | [messengers.md](../messengers.md)       |
| Reach it from outside my network                      | [remote-access.md](../remote-access.md) |
| Add tools / bundles / capabilities                    | `aps capability --help`, `aps bundle --help` |
| Launch the HTTP Agent Protocol server                 | `aps serve --addr 127.0.0.1:8080`       |
| See the TUI                                           | run `aps` with no arguments             |

Every `aps <command>` supports `--help`. When in doubt, start
there.

---

## Troubleshooting

### The session won't start and says "not a git repository"

APS profiles with a linked workspace expect to run inside a git
repo. Either run from inside one, or create the profile without a
workspace link, or link a workspace:

```bash
aps profile workspace set hello <workspace-name>
```

See `aps workspace --help`.

### "No active sessions" after I started one

The session probably terminated. Check `aps session list
--all` (if supported) or inspect the logs of the last session:

```bash
aps session logs <session-id>
```

### I can't find the tmux chord

Default detach is `Ctrl-b` then `d`. If you have a custom tmux
prefix, use your prefix instead of `Ctrl-b`.
