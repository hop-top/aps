---
title: Flag Audit — hop.top ecosystem
date: 2026-04-29
task: T-0343
track: kit-reorg-adoption
contributes: hop-top/tlc#T-0504
---

# Flag Audit — hop.top ecosystem

Surveys global + per-subcommand flags across pod, tlc, tip, foo, aps to
land on canonical names, drop conflicts, and align with kit's
auto-registered globals from `hop.top/kit/go/console/cli`.

Repos surveyed (all `~/.w/ideacrafterslabs/<name>/hops/main` except
foo/tip which are not bare worktrees):

- aps    `internal/cli/...`         (this repo, kit-reorg-output branch)
- pod    `internal/cli/...`
- tlc    `internal/cli/...`
- tip    `cmd/tip/main.go`           — minimal; only `--cmd-line`/`--exit`
- foo    `cmd/foo/commands/...`

## Kit-provided globals (auto-registered by `cli.New`)

Every binary built on `hop.top/kit/go/console/cli` gets these on root,
bound to viper, opt-out via `Disable`:

| Long          | Short | Type   | Default | Source          |
|---------------|-------|--------|---------|-----------------|
| --quiet       | -     | bool   | false   | cli.go          |
| --verbose     | -V    | count  | 0       | cli.go          |
| --no-color    | -     | bool   | false   | cli.go          |
| --chdir       | -C    | string | ""      | cli.go          |
| --format      | -     | string | table   | output.RegisterFlags     |
| --no-hints    | -     | bool   | false   | output.RegisterHintFlags |
| --version     | -v    | bool   | false   | fang/cobra      |
| --help        | -h    | bool   | false   | cobra           |
| --help-all    | -     | bool   | false   | cli.go          |

Notes:
- `-v` is bound to `--version` by fang. Conflicts with `-v` for
  `--verbose` (which kit assigns `-V` to dodge the clash).
- `--format` accepts `table|json|yaml`; `output.Render` handles the
  three modes.

## Per-binary globals (extra root persistent flags)

| Binary | Flag | Short | Type | Default | Notes |
|--------|------|-------|------|---------|-------|
| aps    | (none beyond kit) | | | | |
| pod    | --json (per-cmd, not global) | | | | see below |
| tlc    | (none beyond kit) | | | | tlc adds shell aliases via `cli.New(WithAliases…)` |
| tip    | --cmd-line | - | string | "" | cmd-line metadata stub |
| tip    | --exit     | - | int    | 0  | exit-code metadata stub |
| foo    | --pattern  | -p | string | "" | LLM pattern selector (root.Cmd.Flags, not Persistent) |
| foo    | --strategy | -s | string | "" | system-prompt strategy |
| foo    | --model    | -m | string | "" | model override |
| foo    | --no-stream | - | bool  | false | disable streaming |
| foo    | --dry-run  | -  | bool   | false | print prompt only |
| foo    | --tool     | -T | []string | nil | enable tools by name |
| foo    | --chain-limit | - | int  | 5     | tool-call iterations |
| foo    | --tools-debug | - | bool | false | log tool calls |
| foo    | --tools-approve | - | bool | false | confirm each call |
| foo    | --fragment | -f | []string | nil | user-prompt fragments |
| foo    | --sf       | -  | []string | nil | system-prompt fragments (persistent) |
| foo    | --schema   | -  | string | "" | structured JSON output (persistent) |
| foo    | --schema-multi | - | string | "" | array JSON output (persistent) |

## --json flag survey (target for replacement with kit `--format`)

| Repo | Command | File:line | Note |
|------|---------|-----------|------|
| aps  | version              | internal/cli/version.go:38         | --json bool |
| aps  | profile list         | internal/cli/profile.go:494        | --json bool |
| aps  | profile trust        | internal/cli/profile_trust.go:122  | --json bool |
| aps  | action list          | internal/cli/action.go:148         | --json bool |
| aps  | workspace sync       | internal/cli/workspace/sync_cmd.go | --json bool |
| aps  | workspace activity   | internal/cli/workspace/activity.go | --json bool |
| aps  | capability list      | internal/cli/capability/list.go    | --json bool |
| aps  | bundle list          | internal/cli/bundle/list.go        | --json bool |
| aps  | conflict {list,show,resolve} | internal/cli/conflict/*.go | --json bool |
| aps  | adapter (16 cmds)    | internal/cli/adapter/*.go          | --json bool, all subcmds |
| aps  | audit log            | internal/cli/audit/log.go          | --json bool |
| aps  | policy {list,show,set} | internal/cli/policy/*.go         | --json bool |
| aps  | session inspect      | internal/cli/session/inspect.go    | --json bool |
| aps  | collab (helpers)     | internal/cli/collab/helpers.go:100 | --json bool |
| aps  | a2a card fetch       | internal/cli/a2a/fetch_card.go     | -f --format string (text\|json) |
| aps  | a2a tasks show         | internal/cli/a2a/get_task.go       | -f --format string (text\|json) |
| aps  | a2a tasks list       | internal/cli/a2a/list_tasks.go     | -f --format string (table\|json) |
| aps  | a2a tasks send        | internal/cli/a2a/send_task.go      | -f --format string (text\|json) |
| aps  | a2a card show        | internal/cli/a2a/show_card.go      | -f --format string (text\|json) |
| tlc  | version              | internal/cli/version.go:42         | --json bool |
| tlc  | prompt task          | internal/cli/prompt_context.go:283 | --json bool |
| pod  | (per-cmd --format)   | internal/cli/flags.go:42           | --format default table |
| foo  | --schema, --schema-multi | cmd/foo/commands/root.go:337-338 | structured-output, distinct from --format |

Conflict: `aps a2a *` already binds `-f` to `--format`. After kit
auto-registers a persistent `--format` on root, the local `-f --format`
flags shadow only at the subcommand level — same name, different
allowed values (text vs table). Action: drop local `-f --format` on
a2a subcommands; let them honour the inherited persistent flag and add
`text` as a synonym (or rewrite the renderers to use kit's
table/json/yaml triple).

## Common per-subcommand flags

| Flag | Short | Type | Default | Frequency | Notes |
|------|-------|------|---------|-----------|-------|
| --force      | -f (some)  | bool   | false  | many | `-f` clashes with kit `--format` if promoted to persistent |
| --yes        | -y         | bool   | false  | profile delete | confirmation skip |
| --dry-run    | -          | bool   | false  | many | no `-n` shortcut anywhere |
| --verbose    | -v         | bool   | false  | profile status, skill list, adapter status, adapter links | `-v` clashes with fang `--version` (lowercase) |
| --quiet      | -q         | bool   | false  | upgrade only (separately from kit's persistent --quiet) |
| --profile    | -p         | string | ""     | many subcmds | required field |
| --json       | -          | bool   | false  | very many | TARGET for removal — replace with --format |
| --short      | -          | bool   | false  | version | "Output only version number" |

## Conflicts

1. **`-v` ambiguity (root)**: fang's `--version` claims `-v`. Several
   aps subcommands also use `-v` for `--verbose` (skill list, profile
   status, adapter links, adapter status). Cobra resolves locally so
   `aps profile status -v` works, but it's surprising. **Decision**:
   keep kit's `-V` for `--verbose` everywhere, drop subcommand-level
   `-v` for verbose, leave fang's `-v` as version on root only.

2. **`-f` ambiguity**: local `-f` is used for `--force` (profile
   delete, collab helpers, session delete/terminate) AND for
   `--format` on `aps a2a *`. **Decision**: per T-0347 spec, `-f` is
   `--format` (matches kit's reservation pattern). Drop `-f` shortname
   on `--force` flags; leave `--force` long form intact.

3. **`--json` per-command vs `--format` persistent**: kit now provides
   `--format` persistent on root. Removing each `--json` is the
   T-0345 work; document the breaking change in CHANGELOG.

4. **`-f`/`--force` on collab helpers (BoolP "force", "f")**: same as
   #2. Drop the short.

5. **Local `--quiet -q` on `aps upgrade`**: shadows kit's persistent
   `--quiet`. `aps upgrade --quiet` already binds to the local one;
   the kit one becomes inert. **Decision**: drop the local `-q --quiet`
   and rely on kit's persistent flag.

## Missing (canonical flags absent where they should exist)

- `--yes` short `-y` is only on `profile delete`. Many `--force`
  destructive flows (`adapter detach/revoke/stop`, `bundle delete`,
  `capability delete`, `session delete/terminate`) should also accept
  `--yes/-y` as the non-destructive confirmation skip; `--force`
  remains for "skip safety guards" semantics. Out of scope for T-0347
  (which only renames existing flags); file as follow-up.

- `--dry-run` short `-n` (POSIX-ish convention from `make -n`) absent
  everywhere. T-0347 lists `--dry-run/-n` as canonical → add `-n`
  short on existing `--dry-run` flags.

- `--no-hints` is now provided by kit as persistent — no per-cmd
  override needed.

## Canonical naming (T-0347 target)

| Canonical | Short | Semantics |
|-----------|-------|-----------|
| --verbose | -V    | repeat-count log verbosity (kit) |
| --yes     | -y    | non-interactive confirm (skip prompts) |
| --force   | -     | bypass safety guards (no short — `-f` reserved for --format) |
| --dry-run | -n    | preview without side effects |
| --format  | -     | output mode (table\|json\|yaml); kit persistent |
| --no-hints | -    | suppress hint footer; kit persistent |
| --quiet   | -     | suppress non-essential output; kit persistent |
| --no-color | -    | disable ANSI; kit persistent |
| --chdir   | -C    | working dir override; kit persistent |
| --version | -v    | print version; fang on root only |

## Action items for downstream tasks

- **T-0344** (this phase): kit's `cli.New` already calls
  `output.RegisterFlags` and `output.RegisterHintFlags` internally
  when `Disable.Format`/`Disable.Hints` are false. APS uses the
  defaults so both flags are already on root. Add a TDD test asserting
  the `--format` persistent flag exists on root.
- **T-0345**: drop every per-cmd `--json` listed above; migrate
  manual `json.NewEncoder` to `output.Render(w, fmt, data)` reading
  `format` from `root.Viper`. Breaking change — document in CHANGELOG.
- **T-0346**: register hints on `profile list` and `workspace list`
  (workspace list lives at `internal/cli/workspace/list.go` — verify);
  output.RenderHints already integrated at the kit level.
- **T-0347**: rename per-cmd shortname collisions:
  - `profile delete -f --force` → `--force` only (drop `-f`)
  - `collab helpers -f --force` → `--force` only
  - `session delete -f --force` → `--force` only
  - `session terminate -f --force` → `--force` only
  - `aps upgrade -q --quiet` → drop, use kit persistent
  - `skill list -v --verbose` → `-V` (or drop short and use kit `-V`)
  - `profile status -v --verbose` → drop, use kit `-V`
  - `adapter status -v --verbose` → drop, use kit `-V`
  - `adapter links -v --verbose` → drop, use kit `-V`
  - Add `-n` short on existing `--dry-run` flags (action run, adapter
    link/unlink/stop/revoke, conflict resolve, migrate, webhook server,
    collab archive/leave/remove via helpers, sync push).
  - aps a2a `-f --format` (text\|json): drop the local one; the kit
    persistent --format applies. Rewrite renderers to use
    output.Render with the canonical table/json/yaml triple.

## References

- kit globals: `~/.w/ideacrafterslabs/kit/hops/main/go/console/cli/cli.go#210-246`
- kit output: `~/.w/ideacrafterslabs/kit/hops/main/go/console/output/renderer.go#68-99`
- kit hints:  `~/.w/ideacrafterslabs/kit/hops/main/go/console/output/hint.go#77-141`
- aps root:   `internal/cli/root.go#20-101`
