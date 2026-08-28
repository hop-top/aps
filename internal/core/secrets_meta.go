package core

// This file carries documentation metadata for every secrets backend and
// must stay free of build constraints: the doc renderer
// (internal/tools/configmd) walks SecretsBackendsMeta to regenerate the
// "Secret Store Backends" table in docs/user/messengers.md, so it has to
// see opt-in backends such as openbao regardless of the build tags the
// current binary was produced with.

// SecretsBackendMeta describes one secrets backend for generated
// documentation and diagnostics.
type SecretsBackendMeta struct {
	// Name is the canonical identifier accepted by secrets.backend.
	Name string
	// Default marks the backend used when secrets.backend is unset.
	Default bool
	// BuildTag names the build tag that compiles the backend in; empty
	// when the backend is always available.
	BuildTag string
	// Where says where the backend stores secrets (markdown).
	Where string
	// Required lists the config fields the backend needs (markdown).
	Required string
}

// SecretsBackendsMeta documents every backend in SecretsBackends, in the
// same order. Pinned by tests: each SecretsBackend* const must have
// exactly one row here, exactly one row is the default, and BuildTag
// must agree with optionalBackends.
var SecretsBackendsMeta = []SecretsBackendMeta{
	{
		Name:     SecretsBackendFile,
		Default:  true,
		Where:    "the profile's `secrets.env`, mode 0600",
		Required: "—",
	},
	{
		Name:     SecretsBackendEnv,
		Where:    "`APS_SECRET_<NAME>` in the environment",
		Required: "`prefix` to override `APS_SECRET_`",
	},
	{
		Name:     SecretsBackendKeyring,
		Where:    "OS keychain",
		Required: "`service` (defaults to `aps/<profile>`)",
	},
	{
		Name:     SecretsBackendOnePassword,
		Where:    "1Password, via the `op` CLI or Connect",
		Required: "`vault`; plus `connect_url` + `token` for Connect",
	},
	{
		Name:     SecretsBackendOpenBao,
		BuildTag: "openbao",
		Where:    "OpenBao / Vault KV v2",
		Required: "`addr`, `token`; `mount` defaults to `secret`",
	},
	{
		Name:     SecretsBackendInfisical,
		Where:    "Infisical",
		Required: "`addr`, `project`, `env`, `token`",
	},
	{
		Name:     SecretsBackendGHSecrets,
		Where:    "GitHub Actions secrets",
		Required: "`repo` (defaults to the current repo)",
	},
}
