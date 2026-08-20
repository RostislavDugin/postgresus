package users_services

import (
	"context"

	"golang.org/x/oauth2"

	users_dto "databasus-backend/internal/features/users/dto"
)

func (s *UserService) HandleGitHubOAuthWithMockEndpoint(
	ctx context.Context,
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGitHubOAuthWithEndpoint(ctx, code, redirectUri, endpoint, userAPIURL)
}

func (s *UserService) HandleGoogleOAuthWithMockEndpoint(
	ctx context.Context,
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGoogleOAuthWithEndpoint(ctx, code, redirectUri, endpoint, userAPIURL)
}
