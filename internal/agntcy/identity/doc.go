// Package identity implements the AGNTCY-protocol-specific identity
// surface: DIDs (did:key, did:web), DID Documents, agent badges, and
// Ed25519 message signing/verification keyed off per-profile key
// material at <profile-dir>/identity.key.
//
// # Why this isn't kit/core/identity
//
// hop.top/kit/go/core/identity is a generic, single-key, XDG-default
// local-first identity primitive. It provides:
//
//   - one Keypair per CLI tool (loaded via DefaultStore.LoadOrGenerate)
//   - PEM-encoded keypair files
//   - EdDSA JWT signing
//   - NaCl secretbox encryption keyed off the keypair
//
// AGNTCY identity is a different problem shape:
//
//   - many keypairs, scoped to a profile (one identity.key per
//     <APS_DATA_PATH>/profiles/<id>/), not one tool-wide keypair
//   - raw 32-byte seed + raw 32-byte public on disk (mode 0600/0644),
//     not PEM — matches the AGNTCY reference clients
//   - DID encoding (multibase + multicodec 0xed01) on top of the public
//     key, plus DID Document and Verification Method JSON shapes
//   - Agent Badges (signed agent claims) per the AGNTCY spec
//   - did:web URL derivation (did:web:localhost:agents:<profile>)
//
// kit/core/identity has no DID concept, no badge, no per-profile
// store — overloading it would force the kit primitive to pick up
// agentic-protocol semantics it intentionally avoids.
//
// # Relationship to kit/core/identity
//
// They are FULLY SEPARATE. AGNTCY identity does not share its
// keypair with kit/core/identity:
//
//   - kit identity → typically one keypair at $XDG_CONFIG_HOME/aps/
//     identity.pem, used for CLI-level signing (JWTs to APS HTTP API,
//     encrypted-store key derivation).
//   - AGNTCY identity → one keypair per profile at <profile-dir>/
//     identity.key, used for inter-agent protocol (DID resolution,
//     badge signing, agent-to-agent message signing).
//
// They serve different layers (tool-runtime vs. protocol-runtime) and
// could safely coexist for years without convergence.
//
// # Migration path if kit/core/identity grows DID support
//
// If kit/core/identity ever adds a Keypair.DID() / Keypair.DIDDocument()
// surface (currently it does not), AGNTCY identity should:
//
//  1. swap GenerateKeyPair / SaveKeyPair / LoadPrivateKey / LoadPublicKey
//     for kit primitives (still per-profile-scoped via a custom Store
//     path under the profile directory), keeping the file layout
//     compatible.
//  2. swap encodeDIDKey for kit's DID helper.
//  3. keep DIDDocument, VerificationMethod, Badge, and did:web logic
//     here — those are AGNTCY protocol-specific and shouldn't migrate
//     into kit even if kit grows DID support.
//
// Until then, treat kit/core/identity and this package as orthogonal.
//
// Refs T-0382.
package identity
