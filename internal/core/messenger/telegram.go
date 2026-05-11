package messenger

import "strings"

const TelegramSecretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"

// TelegramAuthHook maps Telegram's webhook secret-token header onto the shared
// service validator.
type TelegramAuthHook struct{}

func (TelegramAuthHook) AuthRequirements(service ServiceValidationConfig) AuthRequirements {
	token := strings.TrimSpace(service.Options["webhook_secret_token"])
	tokenEnv := strings.TrimSpace(service.Options["webhook_secret_token_env"])
	if token == "" && tokenEnv == "" {
		return AuthRequirements{}
	}
	return AuthRequirements{
		Scheme:   AuthSchemeToken,
		Header:   TelegramSecretTokenHeader,
		Token:    token,
		TokenEnv: tokenEnv,
	}
}
