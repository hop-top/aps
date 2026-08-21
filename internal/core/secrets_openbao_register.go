//go:build openbao

// The openbao backend is opt-in: it pulls the OpenBao/Vault API client and
// ~17 transitive modules, which every aps build would otherwise carry. Build
// with `-tags openbao` to enable it.
//
// kit ships the openbao store but, unlike every other backend, registers no
// opener for it, so aps registers one here.

package core

import (
	"fmt"

	"hop.top/kit/go/storage/secret"
	"hop.top/kit/go/storage/secret/openbao"
)

func init() {
	openBaoAvailable = true
	secret.RegisterBackend(SecretsBackendOpenBao, func(cfg secret.Config) (secret.MutableStore, error) {
		if cfg.Addr == "" {
			return nil, fmt.Errorf("secret: openbao backend requires Addr")
		}
		if cfg.Token == "" {
			return nil, fmt.Errorf("secret: openbao backend requires Token")
		}
		return openbao.New(cfg.Addr, cfg.Token, cfg.Mount)
	})
}
