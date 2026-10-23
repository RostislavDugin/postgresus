package oidctesting

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
)

// FakeIdentityProvider serves the three endpoints a client reads from a real
// provider — discovery, keys and the token exchange — so a test can drive the
// whole sign-in flow without a Keycloak instance.
type FakeIdentityProvider struct {
	Issuer string

	// IssuerOverride makes the discovery document claim an issuer other than
	// the address it is served from, which is the swap a client must refuse.
	IssuerOverride string

	// SigningKeyOverride signs the identity token with a key the published key
	// set does not contain.
	SigningKeyOverride *rsa.PrivateKey

	server              *httptest.Server
	signingKey          *rsa.PrivateKey
	identityTokenClaims map[string]any
}

func NewFakeIdentityProvider(t *testing.T) *FakeIdentityProvider {
	t.Helper()

	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	provider := &FakeIdentityProvider{signingKey: signingKey}

	provider.server = httptest.NewServer(http.HandlerFunc(provider.serve))
	provider.Issuer = provider.server.URL

	t.Cleanup(provider.server.Close)

	return provider
}

func (p *FakeIdentityProvider) SetIdentityTokenClaims(claims map[string]any) {
	p.identityTokenClaims = claims
}

func (p *FakeIdentityProvider) BuildValidClaims(
	clientID, subject, nonce, email string,
) map[string]any {
	return map[string]any{
		"iss":            p.server.URL,
		"aud":            clientID,
		"sub":            subject,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"iat":            time.Now().Unix(),
		"nonce":          nonce,
		"email":          email,
		"email_verified": true,
		"name":           "Test Person",
	}
}

func (p *FakeIdentityProvider) serve(writer http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/.well-known/openid-configuration":
		p.writeJSON(writer, map[string]any{
			"issuer":                                p.getAdvertisedIssuer(),
			"authorization_endpoint":                p.server.URL + "/authorize",
			"token_endpoint":                        p.server.URL + "/token",
			"jwks_uri":                              p.server.URL + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/keys":
		p.writeJSON(writer, jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{{
				Key:       p.signingKey.Public(),
				KeyID:     "test-key",
				Algorithm: string(jose.RS256),
				Use:       "sig",
			}},
		})
	case "/token":
		p.writeJSON(writer, map[string]any{
			"access_token": "fake-access-token",
			"token_type":   "Bearer",
			"id_token":     p.signIdentityToken(),
		})
	default:
		writer.WriteHeader(http.StatusNotFound)
	}
}

func (p *FakeIdentityProvider) getAdvertisedIssuer() string {
	if p.IssuerOverride != "" {
		return p.IssuerOverride
	}

	return p.server.URL
}

func (p *FakeIdentityProvider) signIdentityToken() string {
	signingKey := p.signingKey
	if p.SigningKeyOverride != nil {
		signingKey = p.SigningKeyOverride
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: signingKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key"),
	)
	if err != nil {
		return ""
	}

	rawToken, err := jwt.Signed(signer).Claims(p.identityTokenClaims).Serialize()
	if err != nil {
		return ""
	}

	return rawToken
}

func (p *FakeIdentityProvider) writeJSON(writer http.ResponseWriter, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(payload)
}
