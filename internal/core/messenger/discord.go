package messenger

import (
	"strings"
	"time"
)

// DiscordAuthHook maps Discord interaction signing configuration onto the
// shared service validator. Discord signs timestamp+body with Ed25519 and the
// application public key from the Developer Portal.
type DiscordAuthHook struct{}

func (DiscordAuthHook) AuthRequirements(service ServiceValidationConfig) AuthRequirements {
	opts := service.Options
	if len(opts) == 0 {
		return AuthRequirements{}
	}
	publicKey := strings.TrimSpace(firstConfigured(opts["discord_public_key"], opts["interaction_public_key"]))
	publicKeyEnv := strings.TrimSpace(firstConfigured(opts["discord_public_key_env"], opts["interaction_public_key_env"]))
	if publicKey == "" && publicKeyEnv == "" {
		publicKey, publicKeyEnv = discordPublicKeyBinding(service.Env)
	}
	if publicKey == "" && publicKeyEnv == "" && !discordInteractionReceive(opts) && AuthScheme(strings.ToLower(strings.TrimSpace(opts["auth_scheme"]))) != AuthSchemeEd25519 {
		return AuthRequirements{}
	}
	return AuthRequirements{
		Scheme:             AuthSchemeEd25519,
		Header:             "X-Signature-Ed25519",
		SignatureSecret:    publicKey,
		SignatureSecretEnv: publicKeyEnv,
		TimestampHeader:    "X-Signature-Timestamp",
		TimestampTolerance: parseDurationOption(opts["timestamp_tolerance"], 5*time.Minute),
	}
}

func discordPublicKeyBinding(env map[string]string) (string, string) {
	value := strings.TrimSpace(env["DISCORD_PUBLIC_KEY"])
	if value == "" {
		return "", ""
	}
	if strings.HasPrefix(value, "env:") {
		return "", strings.TrimSpace(strings.TrimPrefix(value, "env:"))
	}
	if len(value) == 64 {
		return value, ""
	}
	return "", ""
}

func discordInteractionReceive(opts map[string]string) bool {
	switch strings.ToLower(strings.TrimSpace(opts["receive"])) {
	case "interaction", "interactions", "discord-interaction", "discord-interactions":
		return true
	default:
		return false
	}
}
