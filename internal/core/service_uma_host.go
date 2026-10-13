package core

// UmaHostConfig holds the settings of the uma engines aps hosts that belong
// to the host, not to any one service: uma keeps them out of channel
// configuration so that a channel cannot redirect the host's credentials.
type UmaHostConfig struct {
	Relay UmaRelayHostConfig `yaml:"relay,omitempty"`
}

// UmaRelayHostConfig is uma's relay.Options as aps exposes it. CallbackHosts
// lists the hosts a relay.messages service's callback URL may name, each as
// it appears in the URL (host, or host:port); empty, the default, allows none,
// so no reply can leave. AllowHTTP permits http callbacks, for local
// development: the callback credential then travels in clear text.
type UmaRelayHostConfig struct {
	CallbackHosts []string `yaml:"callback_hosts,omitempty"`
	AllowHTTP     bool     `yaml:"allow_http,omitempty"`
}

// LoadUmaHostConfig reads the uma host settings from the global aps
// configuration. A configuration that cannot be read yields the defaults,
// which allow no callback host.
var LoadUmaHostConfig = func() UmaHostConfig {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil {
		return UmaHostConfig{}
	}
	return cfg.Uma
}
