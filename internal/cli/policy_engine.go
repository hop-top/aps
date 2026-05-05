package cli

import (
	_ "embed"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

// apsPrincipalResolver picks principal from ctx → KIT_POLICY_ROLE env
// → $USER. Aps profile lookup is a deliberate follow-up — kit's
// DefaultPrincipalResolver already covers the env path; the seam exists
// so a future change can layer aps profile / workspace ownership into
// the principal without touching the wiring sites.
func apsPrincipalResolver(ctx context.Context) policy.Principal {
	return policy.DefaultPrincipalResolver(ctx)
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
