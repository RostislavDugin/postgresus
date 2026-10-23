package users_oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/users/oidc/oidctesting"
)

const testEmail = "person@example.com"

func Test_VerifyAuthorizationCode_WithAValidToken_ReturnsTheIdentity(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)
	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail),
	)

	identity, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.NoError(t, err)
	assert.Equal(t, "subject-1", identity.Subject)
	assert.Equal(t, testEmail, identity.Email)
	assert.Equal(t, "Test Person", identity.Name)
}

func Test_VerifyAuthorizationCode_WithAForeignSignature_IsRefused(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)
	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail),
	)

	foreignKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	identityProvider.SigningKeyOverride = foreignKey

	_, err = provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.Error(t, err)
}

func Test_VerifyAuthorizationCode_WithAnotherAudience_IsRefused(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)
	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims("another-client", "subject-1", attempt.Nonce, testEmail),
	)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.Error(t, err)
}

func Test_VerifyAuthorizationCode_WithAnotherNonce_IsRefused(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)
	identityProvider.SetIdentityTokenClaims(identityProvider.BuildValidClaims(
		testClientID,
		"subject-1",
		"a-nonce-from-another-sign-in",
		testEmail,
	))

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.Error(t, err)
}

func Test_VerifyAuthorizationCode_WithAnExpiredToken_IsRefused(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	claims["exp"] = time.Now().Add(-time.Hour).Unix()
	identityProvider.SetIdentityTokenClaims(claims)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.Error(t, err)
}

func Test_VerifyAuthorizationCode_WithNoEmail_IsRefused(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	delete(claims, "email")
	identityProvider.SetIdentityTokenClaims(claims)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.ErrorIs(t, err, ErrEmailMissing)
}

func Test_VerifyAuthorizationCode_WithAnUnverifiedEmail_IsRefused(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	claims["email_verified"] = false
	identityProvider.SetIdentityTokenClaims(claims)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.ErrorIs(t, err, ErrEmailUnverified)
}

func Test_VerifyAuthorizationCode_WhenAGroupIsAllowedAndCarried_ReturnsTheIdentity(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	_, provider, attempt := newVerificationFixtureWithConfig(
		t,
		identityProvider,
		newTestProviderConfigAllowingGroups(identityProvider.Issuer, "csm-databasus-user"),
	)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	claims["groups"] = []string{"/csm-databasus-user"}
	identityProvider.SetIdentityTokenClaims(claims)

	identity, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.NoError(t, err)
	assert.Equal(t, "subject-1", identity.Subject)
}

func Test_VerifyAuthorizationCode_WhenAGroupIsAllowedAndAnotherIsCarried_IsRefused(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	_, provider, attempt := newVerificationFixtureWithConfig(
		t,
		identityProvider,
		newTestProviderConfigAllowingGroups(identityProvider.Issuer, "csm-databasus-user"),
	)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	claims["groups"] = []string{"/csm-other-app-user", "/everyone"}
	identityProvider.SetIdentityTokenClaims(claims)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.ErrorIs(t, err, ErrGroupMissing)
}

func Test_VerifyAuthorizationCode_WhenAGroupIsAllowedAndTheClaimIsAbsent_IsRefused(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	_, provider, attempt := newVerificationFixtureWithConfig(
		t,
		identityProvider,
		newTestProviderConfigAllowingGroups(identityProvider.Issuer, "csm-databasus-user"),
	)

	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail),
	)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.ErrorIs(t, err, ErrGroupMissing)
}

func Test_VerifyAuthorizationCode_WhenAnyOfSeveralGroupsIsCarried_ReturnsTheIdentity(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	_, provider, attempt := newVerificationFixtureWithConfig(
		t,
		identityProvider,
		newTestProviderConfigAllowingGroups(
			identityProvider.Issuer,
			"csm-databasus-admin",
			"csm-databasus-user",
		),
	)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	claims["groups"] = []string{"/csm-databasus-user"}
	identityProvider.SetIdentityTokenClaims(claims)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.NoError(t, err)
}

func Test_VerifyAuthorizationCode_WhenNoGroupIsListed_IgnoresTheClaim(t *testing.T) {
	identityProvider, provider, attempt := newVerificationFixture(t)

	claims := identityProvider.BuildValidClaims(testClientID, "subject-1", attempt.Nonce, testEmail)
	claims["groups"] = []string{"/some-unrelated-group"}
	identityProvider.SetIdentityTokenClaims(claims)

	_, err := provider.VerifyAuthorizationCode(t.Context(), "a-code", attempt)

	require.NoError(t, err)
}

func newVerificationFixture(
	t *testing.T,
) (*oidctesting.FakeIdentityProvider, *Provider, Attempt) {
	t.Helper()

	identityProvider := oidctesting.NewFakeIdentityProvider(t)

	return newVerificationFixtureWithConfig(
		t,
		identityProvider,
		newTestProviderConfig(identityProvider.Issuer),
	)
}

func newVerificationFixtureWithConfig(
	t *testing.T,
	identityProvider *oidctesting.FakeIdentityProvider,
	config ProviderConfig,
) (*oidctesting.FakeIdentityProvider, *Provider, Attempt) {
	t.Helper()

	provider, err := NewProvider(t.Context(), config, testRedirectURI)
	require.NoError(t, err)

	attempt := Attempt{
		CodeVerifier: "a-verifier",
		Nonce:        "a-nonce",
		RedirectURI:  testRedirectURI,
	}

	return identityProvider, provider, attempt
}
