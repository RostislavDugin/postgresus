package users_enums

type ExternalIdentityProvider string

const (
	ExternalIdentityProviderGitHub ExternalIdentityProvider = "github"
	ExternalIdentityProviderGoogle ExternalIdentityProvider = "google"
	ExternalIdentityProviderOIDC   ExternalIdentityProvider = "oidc"
)
