package users_services

import (
	users_dto "databasus-backend/internal/features/users/dto"

	"golang.org/x/oauth2"
)

func (s *UserService) HandleGitHubOAuthWithMockEndpoint(
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGitHubOAuthWithEndpoint(code, redirectUri, endpoint, userAPIURL)
}

func (s *UserService) HandleGoogleOAuthWithMockEndpoint(
	code, redirectUri string,
	endpoint oauth2.Endpoint,
	userAPIURL string,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGoogleOAuthWithEndpoint(code, redirectUri, endpoint, userAPIURL)
}
