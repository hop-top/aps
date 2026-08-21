package core

// openBaoAvailable reports whether the openbao backend was compiled in. It is
// set by the build-tagged registration file; without `-tags openbao` it stays
// false and selecting the backend reports how to enable it.
var openBaoAvailable = false

// optionalBackends maps each opt-in backend to the build tag that enables it.
// Backends absent from this map are always available.
var optionalBackends = map[string]string{
	SecretsBackendOpenBao: "openbao",
}

// backendAvailable reports whether a backend is compiled into this binary.
func backendAvailable(backend string) bool {
	switch backend {
	case SecretsBackendOpenBao:
		return openBaoAvailable
	default:
		return true
	}
}

// AvailableSecretsBackends returns the backends this binary can actually open,
// omitting opt-in backends that were not compiled in.
func AvailableSecretsBackends() []string {
	out := make([]string, 0, len(SecretsBackends))
	for _, backend := range SecretsBackends {
		if backendAvailable(backend) {
			out = append(out, backend)
		}
	}
	return out
}
