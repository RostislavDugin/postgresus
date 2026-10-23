package users_services

import (
	"context"
	"errors"
	"fmt"

	users_dto "databasus-backend/internal/features/users/dto"
	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	users_oidc "databasus-backend/internal/features/users/oidc"
)

func (s *UserService) StartOIDCSignIn(
	ctx context.Context,
	redirectURI string,
) (*users_dto.OIDCStartResponseDTO, error) {
	provider, err := users_oidc.NewProvider(ctx, users_oidc.GetProviderConfig(), redirectURI)
	if err != nil {
		return nil, err
	}

	state, attempt, err := s.oidcAttemptStore.Issue(ctx, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to start the sign-in: %w", err)
	}

	return &users_dto.OIDCStartResponseDTO{
		AuthorizationUrl: provider.GetAuthorizationURL(state, attempt),
	}, nil
}

func (s *UserService) CompleteOIDCSignIn(
	ctx context.Context,
	request *users_dto.OIDCCallbackRequestDTO,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	attempt, err := s.oidcAttemptStore.Spend(ctx, request.State)
	if err != nil {
		return nil, fmt.Errorf("failed to read the sign-in attempt: %w", err)
	}

	if attempt == nil {
		return nil, errors.New("this sign-in attempt is unknown or has expired, please try again")
	}

	if attempt.RedirectURI != request.RedirectUri {
		return nil, errors.New("this sign-in attempt does not match the request")
	}

	providerConfig := users_oidc.GetProviderConfig()

	provider, err := users_oidc.NewProvider(ctx, providerConfig, attempt.RedirectURI)
	if err != nil {
		return nil, err
	}

	identity, err := provider.VerifyAuthorizationCode(ctx, request.Code, *attempt)
	if err != nil {
		return nil, err
	}

	return s.getOrCreateUserFromOAuth(
		ctx,
		users_models.ExternalIdentity{
			Provider: users_enums.ExternalIdentityProviderOIDC,
			Subject:  identity.Subject,
			Issuer:   providerConfig.IssuerURL,
		},
		identity.Email,
		identity.Name,
	)
}
