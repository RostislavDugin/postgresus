package users_models

import (
	users_enums "databasus-backend/internal/features/users/enums"
)

// ExternalIdentity is one account's link to one external provider: the provider
// that vouched for the person, and the identifier that provider uses for them.
// Issuer is carried only for OpenID Connect, where a subject is unique within
// its issuer rather than globally.
type ExternalIdentity struct {
	Provider users_enums.ExternalIdentityProvider
	Subject  string
	Issuer   string
}

func (i ExternalIdentity) GetLinkColumns() map[string]any {
	switch i.Provider {
	case users_enums.ExternalIdentityProviderGitHub:
		return map[string]any{"github_oauth_id": i.Subject}
	case users_enums.ExternalIdentityProviderGoogle:
		return map[string]any{"google_oauth_id": i.Subject}
	case users_enums.ExternalIdentityProviderOIDC:
		return map[string]any{"oidc_issuer": i.Issuer, "oidc_subject": i.Subject}
	}

	return nil
}

func (i ExternalIdentity) ApplyTo(user *User) {
	switch i.Provider {
	case users_enums.ExternalIdentityProviderGitHub:
		user.GitHubOAuthID = &i.Subject
	case users_enums.ExternalIdentityProviderGoogle:
		user.GoogleOAuthID = &i.Subject
	case users_enums.ExternalIdentityProviderOIDC:
		user.OIDCIssuer = &i.Issuer
		user.OIDCSubject = &i.Subject
	}
}

func (i ExternalIdentity) IsLinkedTo(user *User) bool {
	switch i.Provider {
	case users_enums.ExternalIdentityProviderGitHub:
		return user.GitHubOAuthID != nil && *user.GitHubOAuthID == i.Subject
	case users_enums.ExternalIdentityProviderGoogle:
		return user.GoogleOAuthID != nil && *user.GoogleOAuthID == i.Subject
	case users_enums.ExternalIdentityProviderOIDC:
		return user.OIDCIssuer != nil && *user.OIDCIssuer == i.Issuer &&
			user.OIDCSubject != nil && *user.OIDCSubject == i.Subject
	}

	return false
}

// A sign-in that finds an account by address must not rebind an account another
// subject of the same provider already owns.
func (i ExternalIdentity) HasConflictingLink(user *User) bool {
	switch i.Provider {
	case users_enums.ExternalIdentityProviderGitHub:
		return user.GitHubOAuthID != nil && *user.GitHubOAuthID != i.Subject
	case users_enums.ExternalIdentityProviderGoogle:
		return user.GoogleOAuthID != nil && *user.GoogleOAuthID != i.Subject
	case users_enums.ExternalIdentityProviderOIDC:
		return user.OIDCSubject != nil && !i.IsLinkedTo(user)
	}

	return false
}
