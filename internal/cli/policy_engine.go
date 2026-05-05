package cli

import (
	_ "embed"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	collab "hop.top/aps/internal/core/collaboration"
	"hop.top/aps/internal/storage"

	"hop.top/kit/go/runtime/bus"
	"hop.top/kit/go/runtime/policy"
	"hop.top/kit/go/runtime/policy/withcel"
)

// policyEngineFile is the bundled aps default policy bundle. Loaded
// when no user policies file is present at $XDG_CONFIG_HOME/aps/
// policies.yaml, and seeded into that path on first boot for adopters
// to extend. Mirrors tlc T-1192 / wsm T-1288.
//
//go:embed embedded/policies.yaml
var policyEngineFile []byte

// Process-wide policy.Engine state. Initialised on first
// PersistentPreRunE invocation. Nil until kit/runtime/policy has been
// wired (or wiring failed — tests bypass the bootstrap).
var (
	policyEng    *policy.Engine
	policyUnwire func()
	policyOnce   sync.Once
	policyErr    error
)

// initPolicyEngine loads the policy YAML, builds a CEL-backed engine,
// and wires the engine to the supplied bus. Idempotent: subsequent
// calls return the previously-constructed engine (or the captured boot
// error). Misconfig fails loud — the caller surfaces the error from
// PersistentPreRunE so the CLI exits before running the user's command.
//
// Resolution order for the YAML source:
//
//  1. KIT_POLICY_DISABLE=1 — short-circuit; engine is nil and Wire is
//     not called. Tests + dev mode opt-out. Documented escape hatch
//     so T-1293's transition note can point operators to a clean off
//     switch.
//  2. $APS_POLICY_FILE if set (lets tests / CI override).
//  3. $XDG_CONFIG_HOME/aps/policies.yaml, seeded from the bundled
//     default on first boot when missing.
//
// The user file is only auto-seeded when absent or empty so adopter-
// authored rules are never clobbered.
func initPolicyEngine(b bus.Bus) (*policy.Engine, error) {
	policyOnce.Do(func() {
		if os.Getenv("KIT_POLICY_DISABLE") == "1" {
			return
		}
		cfg, err := loadPolicyConfig()
		if err != nil {
			policyErr = err
			return
		}
		eng, err := withcel.New(cfg, policy.WithPrincipalResolver(apsPrincipalResolver))
		if err != nil {
			policyErr = fmt.Errorf("policy: build engine: %w", err)
			return
		}
		policyEng = eng
		if b != nil {
			policyUnwire = policy.Wire(b, eng)
		}
	})
	return policyEng, policyErr
}

// loadPolicyConfig resolves the policy YAML source and returns the
// parsed Config. An empty bundled fallback (no policies) is never
// produced — when the user file is missing or empty we seed it with
// the bundled default and load that.
func loadPolicyConfig() (*policy.Config, error) {
	if path := os.Getenv("APS_POLICY_FILE"); path != "" {
		cfg, err := policy.LoadConfig(path)
		if err != nil {
			return nil, fmt.Errorf("policy: load %s: %w", path, err)
		}
		return cfg, nil
	}

	path, err := ensureDefaultPoliciesFile()
	if err != nil {
		// Couldn't write the user file (permissions, FS error). Fall
		// back to parsing the embedded default so policy enforcement
		// still applies — adopters notice when they try to edit and
		// the file isn't there.
		cfg, perr := policy.ParseConfig(policyEngineFile)
		if perr != nil {
			return nil, fmt.Errorf("policy: parse bundled default after seed failure %v: %w", err, perr)
		}
		return cfg, nil
	}
	cfg, err := policy.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("policy: load %s: %w", path, err)
	}
	return cfg, nil
}

// policiesPath returns the user-level policies.yaml path, alongside
// any other aps config.  XDG_CONFIG_HOME wins; falls back to
// $HOME/.config/aps when unset.
func policiesPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("policies path: resolve home: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "aps", "policies.yaml"), nil
}

// ensureDefaultPoliciesFile writes the bundled policy bytes to the
// user policies path when the file does not exist or is empty
// (size 0). Returns the resolved path. Existing non-empty user files
// are left untouched — never clobber custom rules an adopter may
// have authored.
func ensureDefaultPoliciesFile() (string, error) {
	path, err := policiesPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("create config directory for policies: %w", err)
	}
	st, err := os.Stat(path)
	switch {
	case err == nil && st.Size() > 0:
		return path, nil
	case err != nil && !os.IsNotExist(err):
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	out := make([]byte, len(policyEngineFile))
	copy(out, policyEngineFile)
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// workspaceRoleLookup resolves a profile's AgentRole in a workspace.
// Indirected through a package var so tests can swap the storage
// backend without spinning up a real on-disk collaboration tree.
// Returns ("", nil) when the profile has no membership; non-nil error
// for IO / parse failures (callers fail open to "no role" — see
// apsPrincipalResolver).
var workspaceRoleLookup = func(workspaceID, profileID string) (collab.AgentRole, error) {
	if workspaceID == "" || profileID == "" {
		return "", nil
	}
	store, err := storage.NewCollaborationStorage("")
	if err != nil {
		return "", fmt.Errorf("policy: open collaboration storage: %w", err)
	}
	ws, err := store.LoadWorkspace(workspaceID)
	if err != nil {
		// Workspace not found is a normal "no membership" path — keep it
		// quiet so the env fallback can still kick in.
		var notFound *collab.WorkspaceNotFoundError
		if errors.As(err, &notFound) {
			return "", nil
		}
		return "", fmt.Errorf("policy: load workspace %q: %w", workspaceID, err)
	}
	agent, err := ws.GetAgent(profileID)
	if err != nil {
		// Profile not a member; treat as no role.
		return "", nil
	}
	return agent.Role, nil
}

// callingProfileResolver resolves the calling profile id from CLI
// context. Order:
//
//  1. root.Viper key "profile" — set by the global -p/--profile flag
//     (T-0376). Most aps invocations carry one.
//  2. APS_PROFILE env var — operator-explicit fallback for cron / non-
//     interactive entry points where the flag isn't plumbed.
//  3. "" — caller falls through to kit's DefaultPrincipalResolver,
//     which surfaces $USER.
//
// Indirected through a package var for tests. Wired in init() to avoid
// the package-init cycle that would arise from a literal initializer
// (apsPrincipalResolver → callingProfileResolver → root → ...).
var callingProfileResolver func() string

func init() {
	callingProfileResolver = func() string {
		if root.Viper != nil {
			if pid := root.Viper.GetString("profile"); pid != "" {
				return pid
			}
		}
		if pid := os.Getenv("APS_PROFILE"); pid != "" {
			return pid
		}
		return ""
	}
}

// apsPrincipalResolver layers aps workspace membership over kit's
// DefaultPrincipalResolver. Resolution order for principal.role:
//
//  1. context.request_attrs.workspace_id + active profile → AgentRole
//     looked up from the workspace's membership list. Surfaces
//     "owner"|"contributor"|"observer" as principal.role.
//  2. Kit default — KIT_POLICY_ROLE env var (or empty). Operator
//     escape hatch for emergency overrides; documented in
//     docs/policies.md.
//
// principal.id mirrors the calling profile id when known, otherwise
// falls back to kit's default ($USER).
//
// Failure modes (T-1308):
//
//   - workspace_id missing or unparseable → fall back to kit default.
//     The CLI plumbs workspace_id only for state-changing commands
//     that target a specific workspace (workspace ctx delete, etc.);
//     other commands keep KIT_POLICY_ROLE behavior unchanged.
//   - workspace_id points at a non-existent workspace → no role,
//     fall through to kit default.
//   - profile not a member of the workspace → no role, fall through.
//   - Storage IO panic → recovered, no role, fall through. The
//     resolver runs BEFORE the entity-mutating call; a panic here must
//     not crash the policy engine.
func apsPrincipalResolver(ctx context.Context) (p policy.Principal) {
	// Panic-safety: recover into the kit default so the policy engine
	// never crashes on a malformed registry / FS error / etc. The
	// principal feeds CEL evaluation; an empty role + env fallback is
	// strictly better than a CLI panic before the user's command runs.
	defer func() {
		if r := recover(); r != nil {
			p = policy.DefaultPrincipalResolver(ctx)
		}
	}()

	base := policy.DefaultPrincipalResolver(ctx)

	wsID := workspaceIDFromContext(ctx)
	if wsID == "" {
		return base
	}

	if callingProfileResolver == nil {
		return base
	}
	profileID := callingProfileResolver()
	if profileID == "" {
		return base
	}

	role, err := workspaceRoleLookup(wsID, profileID)
	if err != nil || role == "" {
		// Lookup failure or no membership — keep kit's resolution and
		// let the rule layer decide via KIT_POLICY_ROLE / explicit
		// allow rules. Empty role triggers deny on rules that gate on
		// principal.role in [...].
		return base
	}

	return policy.Principal{
		ID:     profileID,
		Role:   string(role),
		Source: "aps.workspace",
	}
}

// workspaceIDFromContext reads request_attrs.workspace_id from ctx.
// Returns "" when missing or wrong type. Mirrors the request_attrs
// shape stamped by policygate.withRequestAttrs.
func workspaceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	attrs, ok := ctx.Value(policy.ContextAttrsKey).(map[string]any)
	if !ok {
		return ""
	}
	ra, ok := attrs["request_attrs"].(map[string]any)
	if !ok {
		return ""
	}
	wsID, _ := ra["workspace_id"].(string)
	return wsID
}

// closePolicy unsubscribes the engine handlers. Called from Execute's
// shutdown defer, before the bus is closed, so the unsubscribe is a
// no-op against a still-open bus.
func closePolicy() {
	if policyUnwire != nil {
		policyUnwire()
		policyUnwire = nil
	}
}

// policyOnceReset clears the bootstrap state so tests can re-run
// initPolicyEngine in a single process. Test-only helper.
func policyOnceReset() {
	if policyUnwire != nil {
		policyUnwire()
		policyUnwire = nil
	}
	policyEng = nil
	policyErr = nil
	policyOnce = sync.Once{}
}
