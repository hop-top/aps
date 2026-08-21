package core

import (
	"os"
	"strings"
	"sync"
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
var profileSecretCache sync.Map // profileID -> *profileSecrets

type profileSecrets struct {
	once   sync.Once
	values map[string]string
}

func (p *profileSecrets) get(profileID, name string) (string, bool) {
	p.once.Do(func() {
		if loaded, err := LoadProfileSecrets(profileID); err == nil {
			p.values = loaded
		}
	})
	v, ok := p.values[name]
	return v, ok
}

// ProfileSecretLookup returns a SecretLookup backed by the profile's configured
// secret store. The store is read at most once per profile for the lifetime of
// the process; repeated calls share that result.
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
