package domain

const (
	PublicKeyPrefix  = "pk_"
	SecretKeyPrefix  = "sk_"
	WebhookKeyPrefix = "whsec_"
)

func KeyPrefixForType(keyType KeyType) string {
	switch keyType {
	case KeyTypePublic:
		return PublicKeyPrefix
	case KeyTypeSecret:
		return SecretKeyPrefix
	case KeyTypeWebhook:
		return WebhookKeyPrefix
	default:
		return ""
	}
}
