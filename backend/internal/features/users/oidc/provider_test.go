package users_oidc

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/users/oidc/oidctesting"
)

const (
	testRedirectURI = "http://localhost:3000/auth/callback"
	testClientID    = "databasus"
)

func Test_Provider_WhenTheDiscoveryDocumentMatchesTheIssuer_IsBuilt(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	provider, err := NewProvider(
		t.Context(),
		newTestProviderConfig(identityProvider.Issuer),
		testRedirectURI,
	)

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func Test_Provider_WhenTheDiscoveryDocumentClaimsAnotherIssuer_IsRefused(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	identityProvider.IssuerOverride = "https://attacker.example.com"

	_, err := NewProvider(
		t.Context(),
		newTestProviderConfig(identityProvider.Issuer),
		testRedirectURI,
	)

	require.Error(t, err)
}

func Test_Provider_WhenTheIssuerIsUnreachable_IsRefused(t *testing.T) {
	_, err := NewProvider(t.Context(), newTestProviderConfig("http://127.0.0.1:1"), testRedirectURI)

	require.Error(t, err)
}

func Test_Provider_WhenNothingIsConfigured_IsRefused(t *testing.T) {
	_, err := NewProvider(t.Context(), ProviderConfig{}, testRedirectURI)

	require.Error(t, err)
}

func Test_GetAuthorizationURL_CarriesTheStateChallengeAndNonce(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	provider, err := NewProvider(
		t.Context(),
		newTestProviderConfig(identityProvider.Issuer),
		testRedirectURI,
	)
	require.NoError(t, err)

	attempt := Attempt{CodeVerifier: "a-verifier", Nonce: "a-nonce", RedirectURI: testRedirectURI}

	authorizationURL, err := url.Parse(provider.GetAuthorizationURL("a-state", attempt))
	require.NoError(t, err)

	query := authorizationURL.Query()

	assert.Equal(t, "a-state", query.Get("state"))
	assert.Equal(t, "a-nonce", query.Get("nonce"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.Equal(t, GetCodeChallenge("a-verifier"), query.Get("code_challenge"))
	assert.NotContains(t, authorizationURL.RawQuery, "a-verifier")
}

func newTestProviderConfig(issuerURL string) ProviderConfig {
	return ProviderConfig{
		IssuerURL:    issuerURL,
		ClientID:     testClientID,
		clientSecret: "a-secret",
	}
}

func newTestProviderConfigAllowingGroups(issuerURL string, allowedGroups ...string) ProviderConfig {
	config := newTestProviderConfig(issuerURL)
	config.allowedGroups = allowedGroups

	return config
}
