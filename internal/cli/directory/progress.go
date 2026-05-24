package directory

// Phase names used by kit/console/progress emitters across the
// directory subcommands. Centralised so the linter's goconst pass
// sees one definition rather than several string literals.
const (
	phaseConnect = "connect"
	phaseFetch   = "fetch"
	phasePublish = "publish"
	phaseDelete  = "delete"
)
