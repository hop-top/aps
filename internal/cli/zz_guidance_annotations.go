// Package cli — kit/examples + kit/next-steps post-init pass (T-0655).
//
// Walks the fully-built rootCmd tree once at package init time (file
// is named zz_* so it sorts after every subdir constructor's init in
// Go's alphabetical ordering) and stamps kit/examples + kit/next-steps
// onto every runnable leaf the kit/cli EnforceGuidance validator
// requires.
//
// Why this lives in a single top-level file:
//
//   - Kit 0.4 Config.EnforceGuidance (now flipped to true on the aps
//     root) demands kit/examples on every runnable leaf, plus
//     kit/next-steps on every non-read leaf. There are ~170 such
//     leaves across internal/cli/**.
//   - Per-leaf inline annotations would touch ~120 source files. A
//     parallel track (T-0656) is concurrently editing kit/dry-run-
//     rationale on the write/destructive subset of those same files;
//     centralising the guidance pass here keeps the merge surface
//     disjoint (different annotations, different files).
//   - kit/cli reads the annotations at runtime through cmd.Annotations;
//     the spec subcommand picks them up through GetExamples /
//     GetNextSteps. There is no behavioural difference between an
//     inline SetExamples call in the leaf's constructor and one
//     applied here at init() — the wire format is identical.
//
// Add a new leaf? Two options:
//
//  1. Add a path → guidance entry to leafGuidance below (preferred for
//     bulk maintenance; one diff, one review).
//  2. Call kitcli.SetExamples + SetNextSteps inline in the leaf
//     constructor (preferred if the example is dynamically generated
//     from a flag default). The walk below skips leaves that already
//     have an annotation, so the two paths cooperate.
package cli

import (
	"github.com/spf13/cobra"
	kitcli "hop.top/kit/go/console/cli"
)

func init() {
	applyLeafGuidance(rootCmd)
}

// applyLeafGuidance walks the subtree rooted at root and applies the
// leafGuidance table to every runnable leaf that doesn't already carry
// kit/examples (or, for non-read leaves, kit/next-steps). Idempotent:
// leaves that an inline constructor already annotated are skipped.
func applyLeafGuidance(root *cobra.Command) {
	if root == nil {
		return
	}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if isBuiltinCmd(sub) {
				continue
			}
			if sub.HasSubCommands() {
				walk(sub)
				continue
			}
			if !sub.Runnable() {
				continue
			}
			path := sub.CommandPath()
			g, ok := leafGuidance[path]
			if !ok {
				continue
			}
			if _, has := kitcli.GetExamples(sub); !has && len(g.Examples) > 0 {
				_ = kitcli.SetExamples(sub, g.Examples)
			}
			if _, has := kitcli.GetNextSteps(sub); !has && len(g.NextSteps) > 0 {
				_ = kitcli.SetNextSteps(sub, g.NextSteps)
			}
		}
	}
	walk(root)
}

// ex is a one-line constructor used to keep the leafGuidance table
// readable when many entries carry a single example with no Output.
func ex(title, command string) kitcli.Example {
	return kitcli.Example{Title: title, Command: command}
}

// ns is a one-line constructor for NextStep entries; "Suggest" is the
// only field we routinely populate, so the helper keeps the table
// dense without sacrificing field-name clarity at the call site.
func ns(suggest, reason string) kitcli.NextStep {
	return kitcli.NextStep{Suggest: suggest, Reason: reason}
}

// leafGuidance maps every runnable leaf path under rootCmd to its
// kit/examples + (where required by EnforceGuidance) kit/next-steps
// annotations. Read leaves carry Examples only; write/destructive/
// interactive leaves carry both. Keep alphabetised by command path
// for diff hygiene.
var leafGuidance = map[string]kitcli.Guidance{
	// === a2a ===
	"aps a2a card fetch": {
		Examples: []kitcli.Example{
			ex("Fetch a peer's agent card", "aps a2a card fetch https://alice.example/.well-known/agent-card.json"),
		},
	},
	"aps a2a card show": {
		Examples: []kitcli.Example{
			ex("Show this profile's agent card", "aps a2a card show --profile alice"),
		},
	},
	"aps a2a server": {
		Examples: []kitcli.Example{
			ex("Run the local A2A JSON-RPC server", "aps a2a server --addr 127.0.0.1:8080"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps a2a tasks list", "list tasks landing on this server"),
			ns("aps a2a card show", "verify the agent card the server advertises"),
		},
	},
	"aps a2a tasks cancel": {
		Examples: []kitcli.Example{
			ex("Cancel an in-flight task", "aps a2a tasks cancel task-7f3a"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps a2a tasks show task-7f3a", "confirm cancellation landed"),
			ns("aps a2a tasks list", "see remaining tasks"),
		},
	},
	"aps a2a tasks list": {
		Examples: []kitcli.Example{
			ex("List local A2A tasks", "aps a2a tasks list"),
		},
	},
	"aps a2a tasks send": {
		Examples: []kitcli.Example{
			ex("Send a task to a peer", "aps a2a tasks send --to alice --message \"summarize doc.txt\""),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps a2a tasks show <id>", "track the task you just sent"),
			ns("aps a2a tasks stream <id>", "watch streaming output"),
		},
	},
	"aps a2a tasks show": {
		Examples: []kitcli.Example{
			ex("Inspect a task by id", "aps a2a tasks show task-7f3a"),
		},
	},
	"aps a2a tasks stream": {
		Examples: []kitcli.Example{
			ex("Stream task output as it arrives", "aps a2a tasks stream task-7f3a"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps a2a tasks show <id>", "review the final task record"),
		},
	},
	"aps a2a tasks subscribe": {
		Examples: []kitcli.Example{
			ex("Subscribe to all task events", "aps a2a tasks subscribe"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps a2a tasks list", "see tasks that arrived during the subscription"),
		},
	},
	"aps a2a toggle": {
		Examples: []kitcli.Example{
			ex("Enable A2A networking for this profile", "aps a2a toggle on --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps a2a server", "start the local server now that A2A is on"),
			ns("aps a2a card show", "verify the published agent card"),
		},
	},

	// === acp ===
	"aps acp server": {
		Examples: []kitcli.Example{
			ex("Run the local ACP server", "aps acp server --addr 127.0.0.1:9090"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter list", "see adapters bound to this server"),
		},
	},
	"aps acp toggle": {
		Examples: []kitcli.Example{
			ex("Enable ACP for this profile", "aps acp toggle on --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps acp server", "start the ACP server"),
		},
	},

	// === action ===
	"aps action list": {
		Examples: []kitcli.Example{
			ex("List available actions", "aps action list"),
		},
	},
	"aps action run": {
		Examples: []kitcli.Example{
			ex("Run a named action", "aps action run deploy --arg env=prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps action show <name>", "review what the action did"),
		},
	},
	"aps action show": {
		Examples: []kitcli.Example{
			ex("Inspect an action's definition", "aps action show deploy"),
		},
	},

	// === adapter ===
	"aps adapter approve": {
		Examples: []kitcli.Example{
			ex("Approve a pending adapter pairing", "aps adapter approve slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter list", "verify the adapter is approved"),
			ns("aps adapter start slack-prod", "bring the adapter online"),
		},
	},
	"aps adapter attach": {
		Examples: []kitcli.Example{
			ex("Attach an adapter to a profile", "aps adapter attach slack-prod --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter status slack-prod", "confirm attachment succeeded"),
		},
	},
	"aps adapter channels": {
		Examples: []kitcli.Example{
			ex("List channels exposed by an adapter", "aps adapter channels slack-prod"),
		},
	},
	"aps adapter create": {
		Examples: []kitcli.Example{
			ex("Create a slack adapter", "aps adapter create slack-prod --type slack --token $SLACK_TOKEN"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter start slack-prod", "bring the new adapter online"),
			ns("aps adapter test slack-prod", "validate connectivity"),
		},
	},
	"aps adapter detach": {
		Examples: []kitcli.Example{
			ex("Detach an adapter from a profile", "aps adapter detach slack-prod --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter list", "confirm detachment"),
		},
	},
	"aps adapter exec": {
		Examples: []kitcli.Example{
			ex("Execute a command against an adapter", "aps adapter exec slack-prod send --channel ops \"hello\""),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter logs slack-prod", "review execution logs"),
		},
	},
	"aps adapter link add": {
		Examples: []kitcli.Example{
			ex("Link an adapter to a workspace", "aps adapter link add slack-prod --workspace ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter link list", "see all current links"),
		},
	},
	"aps adapter link delete": {
		Examples: []kitcli.Example{
			ex("Remove an adapter-workspace link", "aps adapter link delete slack-prod --workspace ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter link list", "confirm removal"),
		},
	},
	"aps adapter link list": {
		Examples: []kitcli.Example{
			ex("List adapter links", "aps adapter link list"),
		},
	},
	"aps adapter list": {
		Examples: []kitcli.Example{
			ex("List configured adapters", "aps adapter list"),
		},
	},
	"aps adapter logs": {
		Examples: []kitcli.Example{
			ex("Tail adapter logs", "aps adapter logs slack-prod --follow"),
		},
	},
	"aps adapter messenger channels": {
		Examples: []kitcli.Example{
			ex("List messenger channels", "aps adapter messenger channels slack-prod"),
		},
	},
	"aps adapter messenger create": {
		Examples: []kitcli.Example{
			ex("Create a messenger adapter", "aps adapter messenger create slack-prod --type slack --token $SLACK_TOKEN"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger start slack-prod", "bring the messenger online"),
		},
	},
	"aps adapter messenger link add": {
		Examples: []kitcli.Example{
			ex("Link a messenger to a workspace", "aps adapter messenger link add slack-prod --workspace ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger link list", "verify the link"),
		},
	},
	"aps adapter messenger link delete": {
		Examples: []kitcli.Example{
			ex("Remove a messenger-workspace link", "aps adapter messenger link delete slack-prod --workspace ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger link list", "confirm removal"),
		},
	},
	"aps adapter messenger link list": {
		Examples: []kitcli.Example{
			ex("List messenger links", "aps adapter messenger link list"),
		},
	},
	"aps adapter messenger list": {
		Examples: []kitcli.Example{
			ex("List messenger adapters", "aps adapter messenger list"),
		},
	},
	"aps adapter messenger logs": {
		Examples: []kitcli.Example{
			ex("Tail messenger logs", "aps adapter messenger logs slack-prod --follow"),
		},
	},
	"aps adapter messenger start": {
		Examples: []kitcli.Example{
			ex("Start a messenger adapter", "aps adapter messenger start slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger status slack-prod", "verify the messenger is healthy"),
		},
	},
	"aps adapter messenger status": {
		Examples: []kitcli.Example{
			ex("Show messenger status", "aps adapter messenger status slack-prod"),
		},
	},
	"aps adapter messenger stop": {
		Examples: []kitcli.Example{
			ex("Stop a messenger adapter", "aps adapter messenger stop slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger status slack-prod", "confirm the messenger is stopped"),
		},
	},
	"aps adapter messenger test": {
		Examples: []kitcli.Example{
			ex("Send a test message", "aps adapter messenger test slack-prod --channel ops"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger logs slack-prod", "inspect the test transmission"),
		},
	},
	"aps adapter pair": {
		Examples: []kitcli.Example{
			ex("Pair an inbound adapter request", "aps adapter pair"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter pending", "see what's still awaiting approval"),
			ns("aps adapter approve <id>", "approve a paired adapter"),
		},
	},
	"aps adapter pending": {
		Examples: []kitcli.Example{
			ex("List adapters awaiting approval", "aps adapter pending"),
		},
	},
	"aps adapter permissions set": {
		Examples: []kitcli.Example{
			ex("Grant an adapter scope", "aps adapter permissions set slack-prod --grant chat.send"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter status slack-prod", "verify updated permissions"),
		},
	},
	"aps adapter presence": {
		Examples: []kitcli.Example{
			ex("Show adapter presence", "aps adapter presence"),
		},
	},
	"aps adapter reject": {
		Examples: []kitcli.Example{
			ex("Reject a pending adapter pairing", "aps adapter reject slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter pending", "see remaining pending requests"),
		},
	},
	"aps adapter revoke": {
		Examples: []kitcli.Example{
			ex("Revoke an adapter's credentials", "aps adapter revoke slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter list", "verify the adapter is gone"),
		},
	},
	"aps adapter start": {
		Examples: []kitcli.Example{
			ex("Start an adapter", "aps adapter start slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter status slack-prod", "verify it came up"),
			ns("aps adapter logs slack-prod --follow", "watch initial activity"),
		},
	},
	"aps adapter status": {
		Examples: []kitcli.Example{
			ex("Show adapter status", "aps adapter status slack-prod"),
		},
	},
	"aps adapter stop": {
		Examples: []kitcli.Example{
			ex("Stop an adapter", "aps adapter stop slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter status slack-prod", "confirm it stopped cleanly"),
		},
	},
	"aps adapter test": {
		Examples: []kitcli.Example{
			ex("Run an adapter smoke test", "aps adapter test slack-prod"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter logs slack-prod", "inspect the test run"),
		},
	},

	// === alias ===
	"aps alias add": {
		Examples: []kitcli.Example{
			ex("Add a shell alias for a profile", "aps alias add ops --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps alias shell", "print eval-able shell snippet"),
			ns("aps alias list", "verify the alias landed"),
		},
	},
	"aps alias delete": {
		Examples: []kitcli.Example{
			ex("Remove a shell alias", "aps alias delete ops"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps alias list", "confirm removal"),
		},
	},
	"aps alias list": {
		Examples: []kitcli.Example{
			ex("List shell aliases", "aps alias list"),
		},
	},
	"aps alias shell": {
		Examples: []kitcli.Example{
			ex("Emit eval-able shell aliases", "eval \"$(aps alias shell)\""),
		},
	},

	// === bundle ===
	"aps bundle create": {
		Examples: []kitcli.Example{
			ex("Create a bundle from a directory", "aps bundle create devops --from ./bundles/devops"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps bundle validate devops", "lint the new bundle"),
			ns("aps bundle show devops", "review what was captured"),
		},
	},
	"aps bundle delete": {
		Examples: []kitcli.Example{
			ex("Delete a bundle", "aps bundle delete devops"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps bundle list", "confirm removal"),
		},
	},
	"aps bundle edit": {
		Examples: []kitcli.Example{
			ex("Edit a bundle definition", "aps bundle edit devops"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps bundle validate devops", "lint after edit"),
		},
	},
	"aps bundle list": {
		Examples: []kitcli.Example{
			ex("List bundles", "aps bundle list"),
		},
	},
	"aps bundle show": {
		Examples: []kitcli.Example{
			ex("Show bundle contents", "aps bundle show devops"),
		},
	},
	"aps bundle validate": {
		Examples: []kitcli.Example{
			ex("Validate a bundle definition", "aps bundle validate devops"),
		},
	},

	// === capability ===
	"aps capability adopt": {
		Examples: []kitcli.Example{
			ex("Adopt an external capability", "aps capability adopt git --path ~/.skills/git"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps capability list", "see the newly adopted capability"),
			ns("aps capability enable git --profile alice", "enable it on a profile"),
		},
	},
	"aps capability delete": {
		Examples: []kitcli.Example{
			ex("Delete an external capability", "aps capability delete git"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps capability list", "confirm removal"),
		},
	},
	"aps capability disable": {
		Examples: []kitcli.Example{
			ex("Disable a capability for a profile", "aps capability disable git --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps capability list --enabled-on alice", "verify the capability is off"),
		},
	},
	"aps capability enable": {
		Examples: []kitcli.Example{
			ex("Enable a capability for a profile", "aps capability enable git --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps capability list --enabled-on alice", "verify activation"),
		},
	},
	"aps capability install": {
		Examples: []kitcli.Example{
			ex("Install a capability from a registry", "aps capability install git"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps capability show git", "inspect the installed capability"),
			ns("aps capability enable git --profile alice", "enable it on a profile"),
		},
	},
	"aps capability link": {
		Examples: []kitcli.Example{
			ex("Link a capability to a profile", "aps capability link git --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "confirm the link landed"),
		},
	},
	"aps capability list": {
		Examples: []kitcli.Example{
			ex("List all capabilities", "aps capability list"),
			ex("List capabilities enabled on a profile", "aps capability list --enabled-on alice"),
		},
	},
	"aps capability patterns list": {
		Examples: []kitcli.Example{
			ex("List capability patterns", "aps capability patterns list"),
		},
	},
	"aps capability show": {
		Examples: []kitcli.Example{
			ex("Show capability metadata", "aps capability show git"),
		},
	},
	"aps capability watch": {
		Examples: []kitcli.Example{
			ex("Watch a capability directory for changes", "aps capability watch git"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps capability list", "see latest registered capabilities"),
		},
	},

	// === chat ===
	"aps chat": {
		Examples: []kitcli.Example{
			ex("Open a chat session for the active profile", "aps chat"),
			ex("Chat under a specific profile", "aps chat --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps session list", "list chat sessions you've created"),
		},
	},

	// === config ===
	"aps config path": {
		Examples: []kitcli.Example{
			ex("Show active config path", "aps config path"),
		},
	},
	"aps config paths": {
		Examples: []kitcli.Example{
			ex("Show all loaded config paths", "aps config paths"),
		},
	},

	// === contact ===
	"aps contact add": {
		Examples: []kitcli.Example{
			ex("Add a contact", "aps contact add alice --email alice@example.com"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps contact show alice", "review the new contact"),
		},
	},
	"aps contact delete": {
		Examples: []kitcli.Example{
			ex("Delete a contact", "aps contact delete alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps contact list", "verify removal"),
		},
	},
	"aps contact find": {
		Examples: []kitcli.Example{
			ex("Search contacts by name or email", "aps contact find alice"),
		},
	},
	"aps contact list": {
		Examples: []kitcli.Example{
			ex("List contacts", "aps contact list"),
		},
	},
	"aps contact note": {
		Examples: []kitcli.Example{
			ex("Add a note to a contact", "aps contact note alice \"prefers Slack DMs\""),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps contact show alice", "review notes on the contact"),
		},
	},
	"aps contact show": {
		Examples: []kitcli.Example{
			ex("Show a contact's details", "aps contact show alice"),
		},
	},
	"aps contact update": {
		Examples: []kitcli.Example{
			ex("Update a contact field", "aps contact update alice --phone +15551234567"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps contact show alice", "verify the update"),
		},
	},

	// === directory ===
	"aps directory delete": {
		Examples: []kitcli.Example{
			ex("Remove a directory registration", "aps directory delete alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps directory discover", "confirm the entry is gone"),
		},
	},
	"aps directory discover": {
		Examples: []kitcli.Example{
			ex("Discover registered directory entries", "aps directory discover"),
		},
	},
	"aps directory register": {
		Examples: []kitcli.Example{
			ex("Register this profile in the directory", "aps directory register --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps directory show alice", "confirm registration"),
		},
	},
	"aps directory show": {
		Examples: []kitcli.Example{
			ex("Show a directory entry", "aps directory show alice"),
		},
	},

	// === docs ===
	"aps docs": {
		Examples: []kitcli.Example{
			ex("Generate cli docs into ./docs", "aps docs --out ./docs"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps spec --format json", "emit the structured tool manifest"),
		},
	},

	// === env ===
	"aps env": {
		Examples: []kitcli.Example{
			ex("Show resolved environment", "aps env"),
		},
	},

	// === identity ===
	"aps identity badge issue": {
		Examples: []kitcli.Example{
			ex("Issue a verifiable badge", "aps identity badge issue --kind ops --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps identity badge verify <badge>", "verify the badge you just issued"),
		},
	},
	"aps identity badge verify": {
		Examples: []kitcli.Example{
			ex("Verify a badge", "aps identity badge verify ./badges/ops-alice.json"),
		},
	},
	"aps identity init": {
		Examples: []kitcli.Example{
			ex("Initialise an identity keypair", "aps identity init --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps identity show --profile alice", "view the new identity"),
			ns("aps identity verify --profile alice", "verify the keypair locally"),
		},
	},
	"aps identity show": {
		Examples: []kitcli.Example{
			ex("Show profile identity", "aps identity show --profile alice"),
		},
	},
	"aps identity verify": {
		Examples: []kitcli.Example{
			ex("Verify the local identity", "aps identity verify --profile alice"),
		},
	},

	// === listen ===
	"aps listen": {
		Examples: []kitcli.Example{
			ex("Listen for inbound bus events", "aps listen"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter list", "review adapters publishing to the bus"),
		},
	},

	// === migrate ===
	"aps migrate messengers": {
		Examples: []kitcli.Example{
			ex("Migrate legacy messenger configs", "aps migrate messengers"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter messenger list", "verify the migrated messengers"),
		},
	},

	// === observability ===
	"aps observability toggle": {
		Examples: []kitcli.Example{
			ex("Enable telemetry export", "aps observability toggle on"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps env", "see resolved observability settings"),
		},
	},

	// === policy ===
	"aps policy list": {
		Examples: []kitcli.Example{
			ex("List policy entries", "aps policy list"),
		},
	},
	"aps policy set": {
		Examples: []kitcli.Example{
			ex("Set a policy value", "aps policy set tool.chat.allow on"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps policy show tool.chat.allow", "verify the change"),
		},
	},
	"aps policy show": {
		Examples: []kitcli.Example{
			ex("Show a policy value", "aps policy show tool.chat.allow"),
		},
	},
	"aps policy trust set": {
		Examples: []kitcli.Example{
			ex("Set a peer trust level", "aps policy trust set alice --level trusted"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps policy trust show alice", "verify the new trust level"),
		},
	},
	"aps policy trust show": {
		Examples: []kitcli.Example{
			ex("Show trust level for a peer", "aps policy trust show alice"),
		},
	},

	// === profile ===
	"aps profile capability add": {
		Examples: []kitcli.Example{
			ex("Grant a profile a capability", "aps profile capability add alice git"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "confirm the capability is attached"),
		},
	},
	"aps profile capability remove": {
		Examples: []kitcli.Example{
			ex("Revoke a profile capability", "aps profile capability remove alice git"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "confirm removal"),
		},
	},
	"aps profile create": {
		Examples: []kitcli.Example{
			ex("Create a profile", "aps profile create alice --display-name \"Alice Ops\""),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "review the new profile"),
			ns("aps profile capability add alice git", "grant capabilities"),
		},
	},
	"aps profile delete": {
		Examples: []kitcli.Example{
			ex("Delete a profile", "aps profile delete alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile list", "confirm the profile is gone"),
		},
	},
	"aps profile edit": {
		Examples: []kitcli.Example{
			ex("Edit a profile in $EDITOR", "aps profile edit alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "review the change"),
		},
	},
	"aps profile export": {
		Examples: []kitcli.Example{
			ex("Export the native profile record", "aps profile export alice"),
			ex("Export an agent role manifest", "aps profile export alice --format agentco --out AGENTS.md"),
		},
	},
	"aps profile import": {
		Examples: []kitcli.Example{
			ex("Import a profile bundle", "aps profile import ./alice.bundle"),
			ex("Import an agent role manifest", "aps profile import ./AGENTS.md"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile list", "see the imported profile"),
		},
	},
	"aps profile list": {
		Examples: []kitcli.Example{
			ex("List profiles", "aps profile list"),
		},
	},
	"aps profile share": {
		Examples: []kitcli.Example{
			ex("Share a profile with a workspace", "aps profile share alice --workspace ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "see the new share"),
		},
	},
	"aps profile show": {
		Examples: []kitcli.Example{
			ex("Show a profile", "aps profile show alice"),
		},
	},
	"aps profile status": {
		Examples: []kitcli.Example{
			ex("Show profile runtime status", "aps profile status alice"),
		},
	},
	"aps profile trust": {
		Examples: []kitcli.Example{
			ex("Show profile trust state", "aps profile trust alice"),
		},
	},
	"aps profile workspace set": {
		Examples: []kitcli.Example{
			ex("Set the active workspace for a profile", "aps profile workspace set alice --workspace ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps profile show alice", "verify the workspace switch"),
		},
	},

	// === run ===
	"aps run": {
		Examples: []kitcli.Example{
			ex("Run a command under a profile", "aps run alice bash"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps session list", "see sessions spawned by the run"),
		},
	},

	// === serve ===
	"aps serve": {
		Examples: []kitcli.Example{
			ex("Run the aps HTTP server", "aps serve --addr 127.0.0.1:7080"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps service routes", "list HTTP routes the server exposes"),
		},
	},

	// === service ===
	"aps service add": {
		Examples: []kitcli.Example{
			ex("Register a backing service", "aps service add ollama --type llm --url http://127.0.0.1:11434"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps service status ollama", "verify connectivity"),
		},
	},
	"aps service routes": {
		Examples: []kitcli.Example{
			ex("List server routes", "aps service routes"),
		},
	},
	"aps service show": {
		Examples: []kitcli.Example{
			ex("Show service details", "aps service show ollama"),
		},
	},
	"aps service start": {
		Examples: []kitcli.Example{
			ex("Start a managed service", "aps service start ollama"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps service status ollama", "verify the service is healthy"),
		},
	},
	"aps service status": {
		Examples: []kitcli.Example{
			ex("Show service status", "aps service status ollama"),
		},
	},
	"aps service stop": {
		Examples: []kitcli.Example{
			ex("Stop a managed service", "aps service stop ollama"),
		},
	},
	"aps service test": {
		Examples: []kitcli.Example{
			ex("Smoke-test a service", "aps service test ollama"),
		},
	},

	// === session ===
	"aps session attach": {
		Examples: []kitcli.Example{
			ex("Attach to a running session", "aps session attach sess-7f3a"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps session list", "list other live sessions"),
		},
	},
	"aps session delete": {
		Examples: []kitcli.Example{
			ex("Delete a completed session", "aps session delete sess-7f3a"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps session list", "confirm removal"),
		},
	},
	"aps session detach": {
		Examples: []kitcli.Example{
			ex("Detach from a session without terminating it", "aps session detach sess-7f3a"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps session list", "see the detached session"),
		},
	},
	"aps session inspect": {
		Examples: []kitcli.Example{
			ex("Inspect session metadata", "aps session inspect sess-7f3a"),
		},
	},
	"aps session list": {
		Examples: []kitcli.Example{
			ex("List sessions", "aps session list"),
		},
	},
	"aps session logs": {
		Examples: []kitcli.Example{
			ex("Tail session logs", "aps session logs sess-7f3a --follow"),
		},
	},
	"aps session terminate": {
		Examples: []kitcli.Example{
			ex("Terminate a session", "aps session terminate sess-7f3a"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps session list", "confirm termination"),
		},
	},

	// === skill ===
	"aps skill install": {
		Examples: []kitcli.Example{
			ex("Install a skill package", "aps skill install pdf-summarize"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps skill show pdf-summarize", "review the installed skill"),
			ns("aps skill validate pdf-summarize", "validate the package"),
		},
	},
	"aps skill list": {
		Examples: []kitcli.Example{
			ex("List skills", "aps skill list"),
		},
	},
	"aps skill run": {
		Examples: []kitcli.Example{
			ex("Run a skill interactively", "aps skill run pdf-summarize --input doc.pdf"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps skill stats pdf-summarize", "review run telemetry"),
		},
	},
	"aps skill show": {
		Examples: []kitcli.Example{
			ex("Show a skill manifest", "aps skill show pdf-summarize"),
		},
	},
	"aps skill stats": {
		Examples: []kitcli.Example{
			ex("Show skill run statistics", "aps skill stats pdf-summarize"),
		},
	},
	"aps skill suggest": {
		Examples: []kitcli.Example{
			ex("Suggest skills for a task", "aps skill suggest \"extract tables from a pdf\""),
		},
	},
	"aps skill validate": {
		Examples: []kitcli.Example{
			ex("Validate a skill package", "aps skill validate pdf-summarize"),
		},
	},

	// === squad ===
	"aps squad check": {
		Examples: []kitcli.Example{
			ex("Run squad readiness checks", "aps squad check ops-team"),
		},
	},
	"aps squad create": {
		Examples: []kitcli.Example{
			ex("Create a squad", "aps squad create ops-team --members alice,bob"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps squad show ops-team", "review the new squad"),
		},
	},
	"aps squad delete": {
		Examples: []kitcli.Example{
			ex("Delete a squad", "aps squad delete ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps squad list", "confirm removal"),
		},
	},
	"aps squad list": {
		Examples: []kitcli.Example{
			ex("List squads", "aps squad list"),
		},
	},
	"aps squad members add": {
		Examples: []kitcli.Example{
			ex("Add a member to a squad", "aps squad members add ops-team --profile carol"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps squad show ops-team", "review the updated roster"),
		},
	},
	"aps squad members remove": {
		Examples: []kitcli.Example{
			ex("Remove a member from a squad", "aps squad members remove ops-team --profile carol"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps squad show ops-team", "confirm removal"),
		},
	},
	"aps squad show": {
		Examples: []kitcli.Example{
			ex("Show squad details", "aps squad show ops-team"),
		},
	},

	// === status ===
	"aps status": {
		Examples: []kitcli.Example{
			ex("Show aps configuration and runtime status", "aps status"),
			ex("Emit status as JSON for agents", "aps status --format json"),
		},
	},

	// === toolspec ===
	"aps toolspec": {
		Examples: []kitcli.Example{
			ex("Emit the aps tool manifest", "aps toolspec --format json"),
		},
	},

	// === upgrade ===
	"aps upgrade preamble": {
		Examples: []kitcli.Example{
			ex("Run the upgrade preamble checks", "aps upgrade preamble"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps version", "verify post-upgrade version"),
		},
	},

	// === version ===
	"aps version": {
		Examples: []kitcli.Example{
			ex("Show aps version", "aps version"),
		},
	},

	// === voice ===
	"aps voice service start": {
		Examples: []kitcli.Example{
			ex("Start the voice service", "aps voice service start"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps voice service status", "verify the voice service is up"),
		},
	},
	"aps voice service status": {
		Examples: []kitcli.Example{
			ex("Show voice service status", "aps voice service status"),
		},
	},
	"aps voice service stop": {
		Examples: []kitcli.Example{
			ex("Stop the voice service", "aps voice service stop"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps voice service status", "confirm shutdown"),
		},
	},
	"aps voice start": {
		Examples: []kitcli.Example{
			ex("Start a voice session", "aps voice start --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps voice service status", "watch live session metrics"),
		},
	},

	// === webhook ===
	"aps webhook server": {
		Examples: []kitcli.Example{
			ex("Run the webhook receiver", "aps webhook server --addr 127.0.0.1:7180"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps adapter list", "see adapters publishing webhooks"),
		},
	},
	"aps webhook toggle": {
		Examples: []kitcli.Example{
			ex("Toggle webhook delivery", "aps webhook toggle on"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps webhook server", "start the receiver"),
		},
	},

	// === workspace ===
	"aps workspace activity": {
		Examples: []kitcli.Example{
			ex("Show workspace activity", "aps workspace activity ops-team"),
		},
	},
	"aps workspace agents": {
		Examples: []kitcli.Example{
			ex("List agents in a workspace", "aps workspace agents ops-team"),
		},
	},
	"aps workspace archive": {
		Examples: []kitcli.Example{
			ex("Archive a workspace", "aps workspace archive ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace list", "see the archived state"),
		},
	},
	"aps workspace audit": {
		Examples: []kitcli.Example{
			ex("Show workspace audit log", "aps workspace audit ops-team"),
		},
	},
	"aps workspace caps": {
		Examples: []kitcli.Example{
			ex("Show capabilities exposed by a workspace", "aps workspace caps ops-team"),
		},
	},
	"aps workspace conflicts list": {
		Examples: []kitcli.Example{
			ex("List unresolved workspace conflicts", "aps workspace conflicts list ops-team"),
		},
	},
	"aps workspace conflicts resolve": {
		Examples: []kitcli.Example{
			ex("Resolve a conflict", "aps workspace conflicts resolve ops-team conflict-7f3a --strategy local"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace conflicts list ops-team", "verify the conflict is cleared"),
		},
	},
	"aps workspace conflicts show": {
		Examples: []kitcli.Example{
			ex("Show conflict details", "aps workspace conflicts show ops-team conflict-7f3a"),
		},
	},
	"aps workspace create": {
		Examples: []kitcli.Example{
			ex("Create a workspace", "aps workspace create ops-team --display-name \"Ops Team\""),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace show ops-team", "review the new workspace"),
			ns("aps workspace join ops-team --profile alice", "add members"),
		},
	},
	"aps workspace ctx delete": {
		Examples: []kitcli.Example{
			ex("Delete a workspace context key", "aps workspace ctx delete ops-team feature-flag-x"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace ctx list ops-team", "confirm removal"),
		},
	},
	"aps workspace ctx get": {
		Examples: []kitcli.Example{
			ex("Get a workspace context value", "aps workspace ctx get ops-team feature-flag-x"),
		},
	},
	"aps workspace ctx history": {
		Examples: []kitcli.Example{
			ex("Show context history", "aps workspace ctx history ops-team feature-flag-x"),
		},
	},
	"aps workspace ctx list": {
		Examples: []kitcli.Example{
			ex("List context keys", "aps workspace ctx list ops-team"),
		},
	},
	"aps workspace ctx set": {
		Examples: []kitcli.Example{
			ex("Set a workspace context value", "aps workspace ctx set ops-team feature-flag-x on"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace ctx get ops-team feature-flag-x", "verify the new value"),
		},
	},
	"aps workspace join": {
		Examples: []kitcli.Example{
			ex("Join a workspace", "aps workspace join ops-team --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace members ops-team", "see the updated roster"),
		},
	},
	"aps workspace leave": {
		Examples: []kitcli.Example{
			ex("Leave a workspace", "aps workspace leave ops-team --profile alice"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace members ops-team", "verify removal"),
		},
	},
	"aps workspace list": {
		Examples: []kitcli.Example{
			ex("List workspaces", "aps workspace list"),
		},
	},
	"aps workspace members": {
		Examples: []kitcli.Example{
			ex("List workspace members", "aps workspace members ops-team"),
		},
	},
	"aps workspace policy": {
		Examples: []kitcli.Example{
			ex("Set a workspace policy", "aps workspace policy ops-team --key invite.public --value off"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace show ops-team", "review applied policies"),
		},
	},
	"aps workspace remove": {
		Examples: []kitcli.Example{
			ex("Remove a workspace", "aps workspace remove ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace list", "verify removal"),
		},
	},
	"aps workspace role": {
		Examples: []kitcli.Example{
			ex("Set a member role", "aps workspace role ops-team --profile alice --role admin"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace members ops-team", "confirm the role change"),
		},
	},
	"aps workspace send": {
		Examples: []kitcli.Example{
			ex("Send a message into a workspace", "aps workspace send ops-team \"deploy starting\""),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace activity ops-team", "see the message landing"),
		},
	},
	"aps workspace show": {
		Examples: []kitcli.Example{
			ex("Show a workspace", "aps workspace show ops-team"),
		},
	},
	"aps workspace sync": {
		Examples: []kitcli.Example{
			ex("Sync a workspace with peers", "aps workspace sync ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace activity ops-team", "see incoming activity"),
		},
	},
	"aps workspace task": {
		Examples: []kitcli.Example{
			ex("Show a workspace task", "aps workspace task ops-team task-7f3a"),
		},
	},
	"aps workspace tasks": {
		Examples: []kitcli.Example{
			ex("List workspace tasks", "aps workspace tasks ops-team"),
		},
	},
	"aps workspace use": {
		Examples: []kitcli.Example{
			ex("Set the active workspace", "aps workspace use ops-team"),
		},
		NextSteps: []kitcli.NextStep{
			ns("aps workspace show ops-team", "review the workspace you switched to"),
		},
	},
}
