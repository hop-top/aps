package messenger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AuthScheme string

const (
	AuthSchemeNone       AuthScheme = ""
	AuthSchemeBearer     AuthScheme = "bearer"
	AuthSchemeToken      AuthScheme = "token"
	AuthSchemeHMACSHA256 AuthScheme = "hmac-sha256"
)

type ServiceValidationConfig struct {
	ID      string
	Adapter string
	Options map[string]string
}

type AuthRequirements struct {
	Scheme             AuthScheme
	Header             string
	Token              string
	TokenEnv           string
	SignatureSecret    string
	SignatureSecretEnv string
	TimestampHeader    string
	TimestampTolerance time.Duration
	ReplayIDHeader     string
}

type ProviderAuthHook interface {
	AuthRequirements(ServiceValidationConfig) AuthRequirements
}

type ProviderAuthHooks map[string]ProviderAuthHook

type RequestValidationInput struct {
	Service ServiceValidationConfig
	Headers http.Header
	Body    []byte
	Now     time.Time
}

type ReplayStore interface {
	MarkSeen(key string, ttl time.Duration) bool
}

type MemoryReplayStore struct {
	mu      sync.Mutex
	seen    map[string]time.Time
	maxSize int
}

func NewMemoryReplayStore() *MemoryReplayStore {
	return &MemoryReplayStore{seen: map[string]time.Time{}, maxSize: 4096}
}

func (s *MemoryReplayStore) MarkSeen(key string, ttl time.Duration) bool {
	if strings.TrimSpace(key) == "" {
		return false
	}
	now := time.Now().UTC()
	expires := now.Add(ttl)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, exp := range s.seen {
		if !exp.After(now) {
			delete(s.seen, k)
		}
	}
	if exp, ok := s.seen[key]; ok && exp.After(now) {
		return true
	}
	if s.maxSize > 0 && len(s.seen) >= s.maxSize {
		for k := range s.seen {
			delete(s.seen, k)
			break
		}
	}
	s.seen[key] = expires
	return false
}

type ServiceValidator struct {
	Hooks       ProviderAuthHooks
	Replay      ReplayStore
	Now         func() time.Time
	MaxBodySize int64
}

func NewServiceValidator() *ServiceValidator {
	return &ServiceValidator{
		Hooks:  ProviderAuthHooks{},
		Replay: NewMemoryReplayStore(),
		Now:    func() time.Time { return time.Now().UTC() },
	}
}

func (v *ServiceValidator) ValidateRequest(ctx context.Context, input RequestValidationInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(input.Service.ID) == "" {
		return nil
	}
	now := input.Now
	if now.IsZero() {
		if v != nil && v.Now != nil {
			now = v.Now()
		} else {
			now = time.Now().UTC()
		}
	}
	req := authRequirements(input.Service)
	if v != nil && v.Hooks != nil {
		provider := firstConfigured(input.Service.Options["provider"], input.Service.Adapter)
		if hook := v.Hooks[provider]; hook != nil {
			req = mergeAuthRequirements(req, hook.AuthRequirements(input.Service))
		}
	}
	return validateAuth(input.Service.ID, req, input.Headers, input.Body, now, replayStore(v))
}

func (v *ServiceValidator) ValidateMessage(service ServiceValidationConfig, msg *NormalizedMessage) error {
	if msg == nil || strings.TrimSpace(service.ID) == "" {
		return nil
	}
	opts := service.Options
	if len(opts) == 0 {
		return nil
	}
	if allowed := splitCSV(opts["allowed_channels"]); len(allowed) > 0 && !containsAny(allowed, msg.Channel.ID, msg.Channel.PlatformID) {
		return ErrSenderNotAllowed(service.ID, fmt.Sprintf("channel %q is not allowed", safeID(msg.Channel.ID)))
	}
	if allowed := splitCSV(opts["allowed_chats"]); len(allowed) > 0 && !containsAny(allowed, msg.Channel.ID, msg.Channel.PlatformID) {
		return ErrSenderNotAllowed(service.ID, fmt.Sprintf("chat %q is not allowed", safeID(msg.Channel.ID)))
	}
	if allowed := splitCSV(opts["allowed_numbers"]); len(allowed) > 0 && !containsAny(allowed, msg.Sender.ID, msg.Sender.PlatformID, msg.Channel.ID, msg.Channel.PlatformID) {
		return ErrSenderNotAllowed(service.ID, "phone number is not allowed")
	}
	return nil
}

func authRequirements(service ServiceValidationConfig) AuthRequirements {
	opts := service.Options
	req := AuthRequirements{
		Scheme:             AuthScheme(strings.TrimSpace(strings.ToLower(opts["auth_scheme"]))),
		Header:             strings.TrimSpace(opts["auth_header"]),
		Token:              strings.TrimSpace(opts["auth_token"]),
		TokenEnv:           strings.TrimSpace(opts["auth_token_env"]),
		SignatureSecret:    strings.TrimSpace(opts["signature_secret"]),
		SignatureSecretEnv: strings.TrimSpace(opts["signature_secret_env"]),
		TimestampHeader:    strings.TrimSpace(opts["timestamp_header"]),
		ReplayIDHeader:     strings.TrimSpace(opts["replay_id_header"]),
	}
	if req.Scheme == AuthSchemeNone {
		if req.Token != "" || req.TokenEnv != "" {
			req.Scheme = AuthSchemeBearer
		} else if req.SignatureSecret != "" || req.SignatureSecretEnv != "" {
			req.Scheme = AuthSchemeHMACSHA256
		}
	}
	if req.Header == "" {
		switch req.Scheme {
		case AuthSchemeBearer:
			req.Header = "Authorization"
		case AuthSchemeToken:
			req.Header = "X-APS-Token"
		case AuthSchemeHMACSHA256:
			req.Header = "X-APS-Signature"
		}
	}
	if req.TimestampHeader == "" && truthyOption(opts["require_timestamp"]) {
		req.TimestampHeader = "X-APS-Timestamp"
	}
	if req.ReplayIDHeader == "" && truthyOption(opts["require_replay_check"]) {
		req.ReplayIDHeader = "X-APS-Delivery-ID"
	}
	req.TimestampTolerance = parseDurationOption(opts["timestamp_tolerance"], 5*time.Minute)
	return req
}

func validateAuth(serviceID string, req AuthRequirements, headers http.Header, body []byte, now time.Time, replay ReplayStore) error {
	if req.Scheme == AuthSchemeNone {
		return validateTimestampAndReplay(serviceID, req, headers, now, replay)
	}
	switch req.Scheme {
	case AuthSchemeBearer, AuthSchemeToken:
		expected := firstConfigured(req.Token, getenv(req.TokenEnv))
		if expected == "" {
			return ErrMissingSecret("message provider auth token")
		}
		got := strings.TrimSpace(headers.Get(req.Header))
		if req.Scheme == AuthSchemeBearer {
			const prefix = "Bearer "
			if !strings.HasPrefix(got, prefix) {
				return ErrAuthFailed(serviceID, "missing bearer token")
			}
			got = strings.TrimSpace(strings.TrimPrefix(got, prefix))
		}
		if got == "" || !hmac.Equal([]byte(got), []byte(expected)) {
			return ErrAuthFailed(serviceID, "invalid provider token")
		}
	case AuthSchemeHMACSHA256:
		secret := firstConfigured(req.SignatureSecret, getenv(req.SignatureSecretEnv))
		if secret == "" {
			return ErrMissingSecret("message provider signature secret")
		}
		signature := strings.TrimSpace(headers.Get(req.Header))
		signature = strings.TrimPrefix(signature, "sha256=")
		got, err := hex.DecodeString(signature)
		if err != nil || len(got) == 0 {
			return ErrAuthFailed(serviceID, "invalid provider signature")
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		if !hmac.Equal(got, mac.Sum(nil)) {
			return ErrAuthFailed(serviceID, "invalid provider signature")
		}
	default:
		return ErrAuthFailed(serviceID, "unsupported provider auth scheme")
	}
	return validateTimestampAndReplay(serviceID, req, headers, now, replay)
}

func validateTimestampAndReplay(serviceID string, req AuthRequirements, headers http.Header, now time.Time, replay ReplayStore) error {
	ttl := req.TimestampTolerance
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if req.TimestampHeader != "" {
		tsRaw := strings.TrimSpace(headers.Get(req.TimestampHeader))
		if tsRaw == "" {
			return ErrAuthFailed(serviceID, "missing provider timestamp")
		}
		ts, err := parseTimestamp(tsRaw)
		if err != nil {
			return ErrAuthFailed(serviceID, "invalid provider timestamp")
		}
		age := now.Sub(ts)
		if age < -ttl || age > ttl {
			return ErrAuthFailed(serviceID, "provider timestamp outside tolerance")
		}
	}
	if req.ReplayIDHeader != "" {
		replayID := strings.TrimSpace(headers.Get(req.ReplayIDHeader))
		if replayID == "" {
			return ErrAuthFailed(serviceID, "missing provider replay id")
		}
		if replay != nil && replay.MarkSeen(serviceID+":"+replayID, ttl) {
			return ErrReplayRejected(serviceID, "duplicate provider delivery")
		}
	}
	return nil
}

func parseTimestamp(raw string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts, nil
	}
	if sec, err := parseUnixSeconds(raw); err == nil {
		return time.Unix(sec, 0).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp")
}

func parseUnixSeconds(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

func mergeAuthRequirements(base, override AuthRequirements) AuthRequirements {
	if override.Scheme != AuthSchemeNone {
		base.Scheme = override.Scheme
	}
	if override.Header != "" {
		base.Header = override.Header
	}
	if override.Token != "" {
		base.Token = override.Token
	}
	if override.TokenEnv != "" {
		base.TokenEnv = override.TokenEnv
	}
	if override.SignatureSecret != "" {
		base.SignatureSecret = override.SignatureSecret
	}
	if override.SignatureSecretEnv != "" {
		base.SignatureSecretEnv = override.SignatureSecretEnv
	}
	if override.TimestampHeader != "" {
		base.TimestampHeader = override.TimestampHeader
	}
	if override.TimestampTolerance != 0 {
		base.TimestampTolerance = override.TimestampTolerance
	}
	if override.ReplayIDHeader != "" {
		base.ReplayIDHeader = override.ReplayIDHeader
	}
	return base
}

func replayStore(v *ServiceValidator) ReplayStore {
	if v == nil {
		return nil
	}
	return v.Replay
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func containsAny(allowed []string, values ...string) bool {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		for _, allow := range allowed {
			if value == allow {
				return true
			}
		}
	}
	return false
}

func parseDurationOption(value string, fallback time.Duration) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}

func firstConfigured(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func getenv(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return os.Getenv(name)
}

func truthyOption(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func safeID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 64 {
		return value
	}
	return value[:61] + "..."
}
