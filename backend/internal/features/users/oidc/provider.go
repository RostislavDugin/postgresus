package users_oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// A provider that answers its discovery document slowly must not hold a sign-in
// request open indefinitely.
const discoveryTimeout = 15 * time.Second

type Provider struct {
	config       ProviderConfig
	discovered   *oidc.Provider
	oauth2Config oauth2.Config
}

// NewProvider reads the provider's discovery document, which is where the
// authorization endpoint, the token endpoint and the signing keys come from.
// go-oidc rejects a document whose own issuer differs from the configured one,
// which is what stops a redirect to an issuer the operator did not name.
func NewProvider(ctx context.Context, config ProviderConfig, redirectURI string) (*Provider, error) {
	if !config.IsConfigured() {
		return nil, errors.New("OpenID Connect sign-in is not configured")
	}

	discoveryContext, cancelDiscovery := context.WithTimeout(ctx, discoveryTimeout)
	defer cancelDiscovery()

	discoveryContext = oidc.ClientContext(discoveryContext, newProxyAwareClient())

	discovered, err := oidc.NewProvider(discoveryContext, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to reach the OpenID Connect provider: %w", err)
	}

	return &Provider{
		config:     config,
		discovered: discovered,
		oauth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.getClientSecret(),
			Endpoint:     discovered.Endpoint(),
			RedirectURL:  redirectURI,
			Scopes:       config.getScopes(),
		},
	}, nil
}

func (p *Provider) GetAuthorizationURL(state string, attempt Attempt) string {
	return p.oauth2Config.AuthCodeURL(
		state,
		oidc.Nonce(attempt.Nonce),
		oauth2.SetAuthURLParam("code_challenge", GetCodeChallenge(attempt.CodeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func newProxyAwareClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
}
