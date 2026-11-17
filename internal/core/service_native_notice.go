package core

import "strings"

// nativeUmaAdapters maps an aps-native SMS or WhatsApp transport (adapter,
// then provider) to the uma adapter that replaces it. Only transports the
// uma bridge hosts today are listed: a generic relay has no uma adapter in
// UmaAdapterCatalogue yet, so a service on it has nowhere to move to.
var nativeUmaAdapters = map[string]map[string]string{
	probeAdapterSMS: {
		providerTwilio: "twilio.sms",
	},
	probeAdapterWhatsApp: {
		providerTwilio:   "twilio.whatsapp",
		"whatsapp-cloud": umaKeyMetaWhatsApp,
	},
}

// NativeTransportUmaAdapter reports the uma adapter a message service on an
// aps-native SMS or WhatsApp transport can move to with `--type uma`. ok is
// false for a service that is not on such a transport (including one already
// on uma) or whose transport has no hosted uma adapter.
func NativeTransportUmaAdapter(service *ServiceConfig) (adapter string, ok bool) {
	if service == nil || service.Type != ServiceTypeMessage {
		return "", false
	}
	providers, ok := nativeUmaAdapters[strings.ToLower(strings.TrimSpace(service.Adapter))]
	if !ok {
		return "", false
	}
	key, ok := providers[strings.ToLower(strings.TrimSpace(service.Options["provider"]))]
	if !ok {
		return "", false
	}
	if _, hosted := UmaAdapterSpecFor(key); !hosted {
		return "", false
	}
	return key, true
}
