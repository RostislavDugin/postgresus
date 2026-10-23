package users_controllers

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	users_dto "databasus-backend/internal/features/users/dto"
	"databasus-backend/internal/features/users/oidc/oidctesting"
	users_testing "databasus-backend/internal/features/users/testing"
	test_utils "databasus-backend/internal/util/testing"
)

const (
	oidcTestClientID    = "databasus"
	oidcTestRedirectURI = "http://localhost:3000/auth/callback"
)

func Test_StartOIDCSignIn_WhenNotConfigured_IsNotImplemented(t *testing.T) {
	useOIDCProvider(t, "")

	test_utils.MakePostRequest(
		t,
		createUserTestRouter(),
		"/api/v1/auth/oidc/start",
		"",
		users_dto.OIDCStartRequestDTO{RedirectUri: oidcTestRedirectURI},
		http.StatusNotImplemented,
	)
}

func Test_StartOIDCSignIn_ReturnsAnAuthorizationUrlCarryingStateAndChallenge(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	var startResponse users_dto.OIDCStartResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		createUserTestRouter(),
		"/api/v1/auth/oidc/start",
		"",
		users_dto.OIDCStartRequestDTO{RedirectUri: oidcTestRedirectURI},
		http.StatusOK,
		&startResponse,
	)

	authorizationURL, err := url.Parse(startResponse.AuthorizationUrl)
	require.NoError(t, err)

	query := authorizationURL.Query()

	assert.NotEmpty(t, query.Get("state"))
	assert.NotEmpty(t, query.Get("nonce"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.NotEmpty(t, query.Get("code_challenge"))
	assert.Empty(t, query.Get("code_verifier"))
}

func Test_CompleteOIDCSignIn_WithANewPerson_RegistersAndReturnsAToken(t *testing.T) {
	users_testing.EnableExternalRegistrations(t.Context())

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	email := newOIDCTestEmail()
	state, nonce := startOIDCAttempt(t, router)

	identityProvider.SetIdentityTokenClaims(identityProvider.BuildValidClaims(
		oidcTestClientID,
		newOIDCTestSubject(),
		nonce,
		email,
	))

	var callbackResponse users_dto.OAuthCallbackResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(state),
		http.StatusOK,
		&callbackResponse,
	)

	assert.NotEmpty(t, callbackResponse.Token)
	assert.Equal(t, email, callbackResponse.Email)
	assert.True(t, callbackResponse.IsNewUser)
}

func Test_CompleteOIDCSignIn_WithAReplayedState_IsRefused(t *testing.T) {
	users_testing.EnableExternalRegistrations(t.Context())

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	state, nonce := startOIDCAttempt(t, router)

	identityProvider.SetIdentityTokenClaims(identityProvider.BuildValidClaims(
		oidcTestClientID,
		newOIDCTestSubject(),
		nonce,
		newOIDCTestEmail(),
	))

	test_utils.MakePostRequest(
		t, router, "/api/v1/auth/oidc/callback", "", newOIDCCallbackRequest(state), http.StatusOK,
	)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(state),
		http.StatusBadRequest,
	)
}

func Test_CompleteOIDCSignIn_WithAnUnknownState_IsRefused(t *testing.T) {
	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	test_utils.MakePostRequest(
		t,
		createUserTestRouter(),
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest("a-state-this-instance-never-issued"),
		http.StatusBadRequest,
	)
}

func Test_CompleteOIDCSignIn_WithAnUnverifiedEmail_IsRefused(t *testing.T) {
	users_testing.EnableExternalRegistrations(t.Context())

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	state, nonce := startOIDCAttempt(t, router)

	claims := identityProvider.BuildValidClaims(
		oidcTestClientID,
		newOIDCTestSubject(),
		nonce,
		newOIDCTestEmail(),
	)
	claims["email_verified"] = false
	identityProvider.SetIdentityTokenClaims(claims)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(state),
		http.StatusBadRequest,
	)
}

func Test_CompleteOIDCSignIn_WhenExternalRegistrationIsClosed_IsRefused(t *testing.T) {
	users_testing.DisableExternalRegistrations(t.Context())
	t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	state, nonce := startOIDCAttempt(t, router)

	identityProvider.SetIdentityTokenClaims(identityProvider.BuildValidClaims(
		oidcTestClientID,
		newOIDCTestSubject(),
		nonce,
		newOIDCTestEmail(),
	))

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(state),
		http.StatusBadRequest,
	)
}

func Test_CompleteOIDCSignIn_WithAnExistingPasswordAccount_LinksIt(t *testing.T) {
	users_testing.EnableExternalRegistrations(t.Context())

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	email := signUpOIDCTestPerson(t, router)
	state, nonce := startOIDCAttempt(t, router)

	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims(oidcTestClientID, newOIDCTestSubject(), nonce, email),
	)

	var callbackResponse users_dto.OAuthCallbackResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(state),
		http.StatusOK,
		&callbackResponse,
	)

	assert.Equal(t, email, callbackResponse.Email)
	assert.False(t, callbackResponse.IsNewUser)
}

func Test_CompleteOIDCSignIn_WhenAnotherSubjectOwnsTheAddress_IsRefused(t *testing.T) {
	users_testing.EnableExternalRegistrations(t.Context())

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	email := signUpOIDCTestPerson(t, router)

	firstState, firstNonce := startOIDCAttempt(t, router)
	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims(oidcTestClientID, "subject-first", firstNonce, email),
	)
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(firstState),
		http.StatusOK,
	)

	secondState, secondNonce := startOIDCAttempt(t, router)
	identityProvider.SetIdentityTokenClaims(
		identityProvider.BuildValidClaims(oidcTestClientID, "subject-second", secondNonce, email),
	)

	refusal := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(secondState),
		http.StatusBadRequest,
	)

	assert.NotContains(t, string(refusal.Body), email)
}

func Test_CompleteOIDCSignIn_WithTheSecondFactorOn_SignsInWithoutACode(t *testing.T) {
	users_testing.EnableExternalRegistrations(t.Context())
	users_testing.EnableTwoFactorAuth(t.Context())
	t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

	identityProvider := oidctesting.NewFakeIdentityProvider(t)
	useOIDCProvider(t, identityProvider.Issuer)

	router := createUserTestRouter()
	state, nonce := startOIDCAttempt(t, router)

	identityProvider.SetIdentityTokenClaims(identityProvider.BuildValidClaims(
		oidcTestClientID,
		newOIDCTestSubject(),
		nonce,
		newOIDCTestEmail(),
	))

	var callbackResponse users_dto.OAuthCallbackResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/auth/oidc/callback",
		"",
		newOIDCCallbackRequest(state),
		http.StatusOK,
		&callbackResponse,
	)

	assert.NotEmpty(t, callbackResponse.Token)
}

func useOIDCProvider(t *testing.T, issuerURL string) {
	t.Helper()

	env := config.GetEnv()

	previousIssuerURL := env.OIDCIssuerURL
	previousClientID := env.OIDCClientID
	previousClientSecret := env.OIDCClientSecret

	t.Cleanup(func() {
		env.OIDCIssuerURL = previousIssuerURL
		env.OIDCClientID = previousClientID
		env.OIDCClientSecret = previousClientSecret
	})

	env.OIDCIssuerURL = issuerURL
	env.OIDCClientID = oidcTestClientID
	env.OIDCClientSecret = "a-secret"
}

func startOIDCAttempt(t *testing.T, router *gin.Engine) (string, string) {
	t.Helper()

	var startResponse users_dto.OIDCStartResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/auth/oidc/start",
		"",
		users_dto.OIDCStartRequestDTO{RedirectUri: oidcTestRedirectURI},
		http.StatusOK,
		&startResponse,
	)

	authorizationURL, err := url.Parse(startResponse.AuthorizationUrl)
	require.NoError(t, err)

	query := authorizationURL.Query()

	return query.Get("state"), query.Get("nonce")
}

func signUpOIDCTestPerson(t *testing.T, router *gin.Engine) string {
	t.Helper()

	email := newOIDCTestEmail()

	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
		Name:     "Existing Person",
	}, http.StatusOK)

	return email
}

func newOIDCCallbackRequest(state string) users_dto.OIDCCallbackRequestDTO {
	return users_dto.OIDCCallbackRequestDTO{
		Code:        "a-code",
		State:       state,
		RedirectUri: oidcTestRedirectURI,
	}
}

func newOIDCTestEmail() string {
	return "oidc-person-" + uuid.New().String()[:8] + "@example.com"
}

func newOIDCTestSubject() string {
	return "subject-" + uuid.New().String()
}
