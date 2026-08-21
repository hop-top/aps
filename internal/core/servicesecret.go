package core

import (
	"os"
	"strings"
	"sync"
	"time"

	"hop.top/aps/internal/logging"
)

// SecretRefPrefix marks a ServiceConfig Env/Options value as a reference to a
// named secret rather than a literal credential.
const SecretRefPrefix = "secret:"

// SecretLookup reports the value bound to name, and whether it was found.
// A backend that is unavailable reports not-found so callers can fall through
// to the next source rather than failing the request.
type SecretLookup func(name string) (string, bool)

// SplitSecretRef reports whether value is a "secret:NAME" reference and, if so,
// returns NAME. A bare "secret:" with no name is not a reference: it names
// nothing, so treating it as one would resolve every such value identically.
func SplitSecretRef(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, SecretRefPrefix) {
		return "", false
	}
	name := strings.TrimSpace(strings.TrimPrefix(trimmed, SecretRefPrefix))
	if name == "" {
		return "", false
	}
	return name, true
}

// ResolveSecretValue resolves a single configured credential value.
//
// A "secret:NAME" reference resolves against store first, then env, using only
// the name it declares. It deliberately does NOT fall back to the configuration
// key the value was bound to: an operator who wrote "secret:NAME" asked for
// NAME, and silently substituting an unrelated ambient variable would hand the
// caller a credential it never named.
//
// Any other non-empty value is a literal and is returned as-is without
// consulting either backend.
func ResolveSecretValue(value string, store, env SecretLookup) (string, bool) {
	name, isRef := SplitSecretRef(value)
	if !isRef {
		literal := strings.TrimSpace(value)
		return literal, literal != ""
	}
	for _, lookup := range []SecretLookup{store, env} {
		if lookup == nil {
			continue
		}
		if resolved, ok := lookup(name); ok {
			if resolved = strings.TrimSpace(resolved); resolved != "" {
				return resolved, true
			}
		}
	}
	return "", false
}

// EnvSecretLookup resolves names against the process environment.
func EnvSecretLookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

// profileSecretCache memoises the drained secret set per profile. Credential
// resolution happens on every inbound webhook, so the backing store must be
// read once per profile rather than once per request — a keyring backend would
// otherwise touch the OS keychain on every message.
//
// Only successful reads are cached. A failed read (vault unreachable, keyring
// locked, malformed secrets file) is retried on the next lookup: caching it
// would poison the profile for the lifetime of the process, so a momentary
// blip at the first webhook would keep failing long after the store recovered.
var profileSecretCache sync.Map // profileID -> *profileSecrets

// secretWarnInterval bounds how often a single profile reports a failing
// secret store. Credentials resolve per inbound webhook, so an unreachable
// vault would otherwise warn once per message.
const secretWarnInterval = time.Minute

type profileSecrets struct {
	mu       sync.Mutex
	loaded   bool
	values   map[string]string
	lastWarn time.Time
	warned   bool

	// Seams for tests; nil means the production behavior.
	load func(profileID string) (map[string]string, error)
	warn func(profileID string, err error)
	now  func() time.Time
}

func (p *profileSecrets) loadSecrets(profileID string) (map[string]string, error) {
	if p.load != nil {
		return p.load(profileID)
	}
	return LoadProfileSecrets(profileID)
}

func (p *profileSecrets) timeNow() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// warnThrottled reports a failing store at most once per secretWarnInterval.
// The first failure after a healthy period always warns, so an outage is
// visible immediately rather than after the window.
func (p *profileSecrets) warnThrottled(profileID string, err error) {
	now := p.timeNow()
	if p.warned && now.Sub(p.lastWarn) < secretWarnInterval {
		return
	}
	p.warned = true
	p.lastWarn = now
	if p.warn != nil {
		p.warn(profileID, err)
		return
	}
	logging.GetLogger().Warn("reading profile secrets failed; will retry",
		"profile", profileID, "error", err)
}

func (p *profileSecrets) get(profileID, name string) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.loaded {
		values, err := p.loadSecrets(profileID)
		if err != nil {
			// Left unloaded so the next lookup retries. Logged rather than
			// returned: callers fall through to their next source, and a
			// silent empty credential is what made this hard to diagnose.
			p.warnThrottled(profileID, err)
			return "", false
		}
		p.values = values
		p.loaded = true
		p.warned = false // re-arm, so a later outage warns immediately
	}
	v, ok := p.values[name]
	return v, ok
}

// ProfileSecretLookup returns a SecretLookup backed by the profile's configured
// secret store. A successful read is shared across calls for the lifetime of
// the process; a failed read is retried on the next lookup.
//
// A profile with no store, or a backend that fails to open, yields a lookup
// that finds nothing, so callers fall through to their next source.
func ProfileSecretLookup(profileID string) SecretLookup {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return nil
	}
	entry, _ := profileSecretCache.LoadOrStore(profileID, &profileSecrets{})
	secrets, _ := entry.(*profileSecrets)
	return func(name string) (string, bool) {
		return secrets.get(profileID, name)
	}
}

// ResetProfileSecretCache clears memoised profile secrets. Intended for tests
// and for callers that have just mutated a profile's secret store.
func ResetProfileSecretCache() {
	profileSecretCache.Range(func(k, _ any) bool {
		profileSecretCache.Delete(k)
		return true
	})
}
