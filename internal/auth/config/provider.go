package config

// AuthProviderConfig defines the raw configuration for an auth provider.
type AuthProviderConfig struct {
	Name         string
	Type         string
	Enabled      bool
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	APIURL       string
	Scopes       []string
	AllowSignUp  bool
	
	// RequiresLicense indicates if this provider requires a valid license (e.g. 'sso' capability).
	RequiresLicense bool
}
