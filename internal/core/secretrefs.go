package core

import (
	"context"
	"fmt"

	"hop.top/kit/go/storage/secret/ref"
)

// secretRefOpener builds the backends for resolving a profile's secrets.env
// references: secret:NAME through the profile's secret store, then the process
// environment; op:// through the op CLI. Resolution is Kit's ref package; this
// only adapts aps's lookup seams.
func secretRefOpener(store, env SecretLookup) ref.Opener {
	return ref.Opener{
		Lookup: func(_ context.Context, name string) (string, bool, error) {
			for _, lookup := range []SecretLookup{store, env} {
				if lookup == nil {
					continue
				}
				if v, ok := lookup(name); ok {
					return v, true, nil
				}
			}
			return "", false, nil
		},
	}
}

// LoadResolvedSecrets loads a profile's secrets.env and resolves any vault
// references to their values, for injection into a child process only. Plain
// values pass through untouched. Resolution fails closed: an unresolvable
// reference returns an error naming the KEY (never a value) and a nil map.
// Callers that display secrets must use LoadSecrets / LoadProfileSecrets.
func LoadResolvedSecrets(profileID, path string) (map[string]string, error) {
	secrets, err := LoadSecrets(path)
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return secrets, nil
	}
	resolved, err := ref.ResolveMap(context.Background(), secrets,
		secretRefOpener(ProfileSecretLookup(profileID), EnvSecretLookup))
	if err != nil {
		return nil, fmt.Errorf("resolve secret references: %w", err)
	}
	return resolved, nil
}
