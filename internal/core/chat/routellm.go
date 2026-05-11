package chat

import (
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"hop.top/aps/internal/core"
	"hop.top/kit/go/ai/llm"
	"hop.top/kit/go/ai/llm/routellm"
	kitconfig "hop.top/kit/go/core/config"
)

const defaultSystemLLMConfigPath = "/etc/aps/llm.yaml"

type LLMResolveOptions struct {
	SystemConfigPath string
	UserConfigPath   string
	ModelOverride    string
}

type ResolvedLLMConfig struct {
	Config       core.LLMConfig
	ProviderURI  string
	Model        string
	FallbackURIs []string
	Router       routellm.RouterConfig
}

func ResolveLLMConfig(profile *core.Profile, opts LLMResolveOptions) (ResolvedLLMConfig, error) {
	cfg := defaultLLMConfig()
	if opts.SystemConfigPath == "" {
		opts.SystemConfigPath = defaultSystemLLMConfigPath
	}
	if opts.UserConfigPath == "" {
		if dir, err := core.GetConfigDir(); err == nil {
			opts.UserConfigPath = filepath.Join(dir, "llm.yaml")
		}
	}

	if err := kitconfig.Load(&cfg, kitconfig.Options{
		SystemConfigPath: opts.SystemConfigPath,
		UserConfigPath:   opts.UserConfigPath,
	}); err != nil {
		return ResolvedLLMConfig{}, err
	}
	if profile != nil && profile.LLM != nil {
		mergeLLMConfig(&cfg, *profile.LLM)
	}
	if override := strings.TrimSpace(opts.ModelOverride); override != "" {
		cfg.DefaultModel = override
	}

	router := routerConfigFromLLM(cfg)
	providerURI, model := providerAndModel(cfg)
	fallbacks := make([]string, 0, len(cfg.Fallback))
	for _, fb := range cfg.Fallback {
		if uri := providerURIForModel(fb, ""); uri != "" {
			fallbacks = append(fallbacks, uri)
		}
	}

	return ResolvedLLMConfig{
		Config:       cfg,
		ProviderURI:  providerURI,
		Model:        model,
		FallbackURIs: fallbacks,
		Router:       router,
	}, nil
}

func DefaultProviderURIs() []string {
	return []string{"anthropic://", "openai://", "google://"}
}

func CandidateProviderURIs(resolved ResolvedLLMConfig) []string {
	var candidates []string
	if parsed, err := llm.ParseURI(resolved.ProviderURI); err == nil && parsed.Scheme == "routellm" {
		if uri := providerURIForModel(resolved.Model, ""); uri != "" {
			candidates = append(candidates, uri)
		}
		candidates = append(candidates, resolved.FallbackURIs...)
		candidates = append(candidates, DefaultProviderURIs()...)
		return dedupe(candidates)
	}
	if resolved.ProviderURI != "" {
		candidates = append(candidates, resolved.ProviderURI)
	}
	candidates = append(candidates, resolved.FallbackURIs...)
	if len(candidates) == 0 {
		candidates = append(candidates, DefaultProviderURIs()...)
	}
	return dedupe(candidates)
}

func defaultLLMConfig() core.LLMConfig {
	rcfg := routellm.DefaultRouterConfig()
	return core.LLMConfig{
		BaseURL: rcfg.BaseURL,
		RouterConfig: map[string]any{
			"base_url":  rcfg.BaseURL,
			"grpc_port": rcfg.GRPCPort,
		},
	}
}

func mergeLLMConfig(dst *core.LLMConfig, src core.LLMConfig) {
	if src.Provider != "" {
		dst.Provider = src.Provider
	}
	if src.DefaultModel != "" {
		dst.DefaultModel = src.DefaultModel
	}
	if src.BaseURL != "" {
		dst.BaseURL = src.BaseURL
	}
	if len(src.Routers) > 0 {
		dst.Routers = append([]string(nil), src.Routers...)
	}
	if len(src.RouterConfig) > 0 {
		if dst.RouterConfig == nil {
			dst.RouterConfig = map[string]any{}
		}
		for k, v := range src.RouterConfig {
			dst.RouterConfig[k] = v
		}
	}
	if len(src.Fallback) > 0 {
		dst.Fallback = append([]string(nil), src.Fallback...)
	}
	if src.StrongModel != "" {
		dst.StrongModel = src.StrongModel
	}
	if src.WeakModel != "" {
		dst.WeakModel = src.WeakModel
	}
}

func routerConfigFromLLM(cfg core.LLMConfig) routellm.RouterConfig {
	rcfg := routellm.DefaultRouterConfig()
	if cfg.BaseURL != "" {
		rcfg.BaseURL = cfg.BaseURL
	}
	if len(cfg.Routers) > 0 {
		rcfg.Routers = append([]string(nil), cfg.Routers...)
	}
	if cfg.StrongModel != "" {
		rcfg.StrongModel = cfg.StrongModel
	}
	if cfg.WeakModel != "" {
		rcfg.WeakModel = cfg.WeakModel
	}
	if len(cfg.RouterConfig) > 0 {
		data, _ := yaml.Marshal(cfg.RouterConfig)
		var overlay routellm.RouterConfig
		if err := yaml.Unmarshal(data, &overlay); err == nil {
			if overlay.BaseURL != "" {
				rcfg.BaseURL = overlay.BaseURL
			}
			if overlay.GRPCPort != 0 {
				rcfg.GRPCPort = overlay.GRPCPort
			}
			if overlay.StrongModel != "" {
				rcfg.StrongModel = overlay.StrongModel
			}
			if overlay.WeakModel != "" {
				rcfg.WeakModel = overlay.WeakModel
			}
			if len(overlay.Routers) > 0 {
				rcfg.Routers = overlay.Routers
			}
			if len(overlay.RouterConfig) > 0 {
				rcfg.RouterConfig = overlay.RouterConfig
			}
			if overlay.Autostart {
				rcfg.Autostart = true
			}
			if overlay.PIDFile != "" {
				rcfg.PIDFile = overlay.PIDFile
			}
			if len(overlay.Eva.Contracts) > 0 || overlay.Eva.Enforce {
				rcfg.Eva = overlay.Eva
			}
		}
		if rcfg.RouterConfig == nil {
			rcfg.RouterConfig = map[string]any{}
		}
		for k, v := range cfg.RouterConfig {
			rcfg.RouterConfig[k] = v
		}
	}
	return rcfg
}

func providerAndModel(cfg core.LLMConfig) (string, string) {
	if len(cfg.Routers) > 0 {
		return routeLLMProviderURI(cfg), cfg.DefaultModel
	}
	return providerURIForModel(cfg.DefaultModel, cfg.Provider), modelName(cfg.DefaultModel)
}

func routeLLMProviderURI(cfg core.LLMConfig) string {
	routerName := cfg.Routers[0]
	threshold := thresholdString(cfg.RouterConfig["threshold"])
	return "routellm://" + routerName + ":" + threshold
}

func thresholdString(value any) string {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case string:
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return "0.7"
}

func providerURIForModel(model, provider string) string {
	model = strings.TrimSpace(model)
	provider = strings.TrimSpace(provider)
	if model == "" && provider == "" {
		return ""
	}
	if strings.Contains(model, "://") {
		return model
	}
	if provider == "" {
		provider = inferProvider(model)
	}
	if provider == "" {
		provider = "openai"
	}
	return provider + "://" + model
}

func modelName(model string) string {
	if parsed, err := llm.ParseURI(model); err == nil {
		return parsed.Model
	}
	return strings.TrimSpace(model)
}

func inferProvider(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "claude"):
		return "anthropic"
	case strings.Contains(lower, "gemini"):
		return "google"
	case strings.Contains(lower, "gpt"), strings.Contains(lower, "o1"), strings.Contains(lower, "o3"), strings.Contains(lower, "o4"):
		return "openai"
	default:
		return ""
	}
}

func routerExtras(rcfg routellm.RouterConfig) map[string]any {
	extras := map[string]any{}
	if rcfg.BaseURL != "" {
		extras["base_url"] = rcfg.BaseURL
	}
	if rcfg.GRPCPort != 0 {
		extras["grpc_port"] = rcfg.GRPCPort
	}
	if rcfg.StrongModel != "" {
		extras["strong_model"] = rcfg.StrongModel
	}
	if rcfg.WeakModel != "" {
		extras["weak_model"] = rcfg.WeakModel
	}
	if len(rcfg.Routers) > 0 {
		extras["routers"] = rcfg.Routers
	}
	if len(rcfg.RouterConfig) > 0 {
		extras["router_config"] = rcfg.RouterConfig
	}
	if rcfg.Autostart {
		extras["autostart"] = true
	}
	if rcfg.PIDFile != "" {
		extras["pid_file"] = rcfg.PIDFile
	}
	return map[string]any{"routellm": extras}
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
