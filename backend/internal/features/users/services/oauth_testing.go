package users_services

import (
	"golang.org/x/oauth2"

	"databasus-backend/internal/config"
	users_dto "databasus-backend/internal/features/users/dto"
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

func (s *UserService) HandleGenericOAuthWithMockEndpoint(
	code, redirectUri string,
	providerConfig config.GenericOAuthProviderConfig,
) (*users_dto.OAuthCallbackResponseDTO, error) {
	return s.handleGenericOAuthWithConfig(&providerConfig, code, redirectUri)
}
