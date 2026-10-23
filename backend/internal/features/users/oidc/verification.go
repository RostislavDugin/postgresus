package users_oidc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Providers and this instance rarely agree on the second, and a token rejected
// for being a moment too young is indistinguishable from an attack to the user.
const clockSkewLeeway = 2 * time.Minute

const exchangeTimeout = 15 * time.Second

var (
	ErrEmailMissing    = errors.New("the OpenID Connect provider returned no email address")
	ErrEmailUnverified = errors.New("the OpenID Connect provider has not verified this email address")
	ErrGroupMissing    = errors.New("this account is not a member of a group allowed to sign in")
)

// VerifiedIdentity is what survives verification: who the provider says this
// is, and the address the account mapping may trust.
type VerifiedIdentity struct {
	Subject string
	Email   string
	Name    string
}

type identityClaims struct {
	Subject         string   `json:"sub"`
	Email           string   `json:"email"`
	IsEmailVerified bool     `json:"email_verified"`
	Name            string   `json:"name"`
	PreferredName   string   `json:"preferred_username"`
	Groups          []string `json:"groups"`
}

// VerifyAuthorizationCode exchanges the code using the attempt's verifier, then
// checks the identity token's signature, issuer, audience and lifetime against
// the provider's published keys, and its nonce against the attempt. The nonce
// is what stops an identity token issued for another sign-in being replayed
// into this one.
func (p *Provider) VerifyAuthorizationCode(
	ctx context.Context,
	code string,
	attempt Attempt,
) (VerifiedIdentity, error) {
	exchangeContext, cancelExchange := context.WithTimeout(ctx, exchangeTimeout)
	defer cancelExchange()

	exchangeContext = oidc.ClientContext(exchangeContext, newProxyAwareClient())

	token, err := p.oauth2Config.Exchange(
		exchangeContext,
		code,
		oauth2.SetAuthURLParam("code_verifier", attempt.CodeVerifier),
	)
	if err != nil {
		return VerifiedIdentity{}, fmt.Errorf("failed to exchange the authorization code: %w", err)
	}

	rawIdentityToken, isPresent := token.Extra("id_token").(string)
	if !isPresent {
		return VerifiedIdentity{}, errors.New("the OpenID Connect provider returned no identity token")
	}

	verifier := p.discovered.Verifier(&oidc.Config{
		ClientID:        p.config.ClientID,
		SkipExpiryCheck: false,
		SkipIssuerCheck: false,
		Now:             func() time.Time { return time.Now().Add(clockSkewLeeway) },
	})

	identityToken, err := verifier.Verify(exchangeContext, rawIdentityToken)
	if err != nil {
		return VerifiedIdentity{}, fmt.Errorf("failed to verify the identity token: %w", err)
	}

	if identityToken.Nonce != attempt.Nonce {
		return VerifiedIdentity{}, errors.New("the identity token does not belong to this sign-in")
	}

	var claims identityClaims
	if err := identityToken.Claims(&claims); err != nil {
		return VerifiedIdentity{}, fmt.Errorf("failed to read the identity token: %w", err)
	}

	if claims.Email == "" {
		return VerifiedIdentity{}, ErrEmailMissing
	}

	if !claims.IsEmailVerified {
		return VerifiedIdentity{}, ErrEmailUnverified
	}

	if !p.config.isGroupAllowed(claims.Groups) {
		return VerifiedIdentity{}, ErrGroupMissing
	}

	return VerifiedIdentity{
		Subject: claims.Subject,
		Email:   claims.Email,
		Name:    claims.getDisplayName(),
	}, nil
}

func (c identityClaims) getDisplayName() string {
	if c.Name != "" {
		return c.Name
	}

	if c.PreferredName != "" {
		return c.PreferredName
	}

	return "User"
}
