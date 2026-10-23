package users_oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/config"
)

func Test_ProviderConfig_WhenEverySettingIsPresent_IsConfigured(t *testing.T) {
	withProviderEnvironment(t, "https://keycloak.example.com/realms/main", "databasus", "a-secret", "")

	providerConfig := GetProviderConfig()

	assert.True(t, providerConfig.IsConfigured())
	assert.Equal(t, "https://keycloak.example.com/realms/main", providerConfig.IssuerURL)
	assert.Equal(t, "databasus", providerConfig.ClientID)
}

func Test_ProviderConfig_WhenTheSecretIsMissing_IsNotConfigured(t *testing.T) {
	withProviderEnvironment(t, "https://keycloak.example.com/realms/main", "databasus", "", "")

	assert.False(t, GetProviderConfig().IsConfigured())
}

func Test_ProviderConfig_WhenNothingIsSet_IsNotConfigured(t *testing.T) {
	withProviderEnvironment(t, "", "", "", "")

	assert.False(t, GetProviderConfig().IsConfigured())
}

func Test_ProviderConfig_WhenNoDisplayNameIsSet_FallsBackToANeutralName(t *testing.T) {
	withProviderEnvironment(t, "https://issuer.example.com", "databasus", "a-secret", "")

	assert.Equal(t, defaultDisplayName, GetProviderConfig().GetDisplayName())
}

func Test_ProviderConfig_WhenADisplayNameIsSet_UsesIt(t *testing.T) {
	withProviderEnvironment(t, "https://issuer.example.com", "databasus", "a-secret", "Acme SSO")

	assert.Equal(t, "Acme SSO", GetProviderConfig().GetDisplayName())
}

func Test_ProviderConfig_WhenGroupsAreListed_ParsesAndNormalizesThem(t *testing.T) {
	withProviderEnvironment(t, "https://issuer.example.com", "databasus", "a-secret", "")
	withAllowedGroupsEnvironment(t, " csm-databasus-user , /csm-databasus-admin ,, ")

	providerConfig := GetProviderConfig()

	assert.True(t, providerConfig.HasAllowedGroups())
	assert.Equal(
		t,
		[]string{"csm-databasus-user", "csm-databasus-admin"},
		providerConfig.GetAllowedGroups(),
	)
}

func Test_ProviderConfig_WhenNoGroupIsListed_AllowsEveryone(t *testing.T) {
	withProviderEnvironment(t, "https://issuer.example.com", "databasus", "a-secret", "")
	withAllowedGroupsEnvironment(t, "")

	providerConfig := GetProviderConfig()

	assert.False(t, providerConfig.HasAllowedGroups())
	assert.Empty(t, providerConfig.GetAllowedGroups())
}

func Test_ProviderConfig_WhenGroupsAreAllowed_AsksForTheGroupsScope(t *testing.T) {
	withProviderEnvironment(t, "https://issuer.example.com", "databasus", "a-secret", "")
	withAllowedGroupsEnvironment(t, "csm-databasus-user")

	assert.Contains(t, GetProviderConfig().getScopes(), "groups")
}

func Test_ProviderConfig_WhenNoGroupIsListed_DoesNotAskForTheGroupsScope(t *testing.T) {
	withProviderEnvironment(t, "https://issuer.example.com", "databasus", "a-secret", "")
	withAllowedGroupsEnvironment(t, "")

	assert.NotContains(t, GetProviderConfig().getScopes(), "groups")
}

func withAllowedGroupsEnvironment(t *testing.T, allowedGroups string) {
	t.Helper()

	env := config.GetEnv()
	previousAllowedGroups := env.OIDCAllowedGroups

	t.Cleanup(func() {
		env.OIDCAllowedGroups = previousAllowedGroups
	})

	env.OIDCAllowedGroups = allowedGroups
}

func withProviderEnvironment(t *testing.T, issuerURL, clientID, clientSecret, displayName string) {
	t.Helper()

	env := config.GetEnv()

	previousIssuerURL := env.OIDCIssuerURL
	previousClientID := env.OIDCClientID
	previousClientSecret := env.OIDCClientSecret
	previousDisplayName := env.OIDCDisplayName

	t.Cleanup(func() {
		env.OIDCIssuerURL = previousIssuerURL
		env.OIDCClientID = previousClientID
		env.OIDCClientSecret = previousClientSecret
		env.OIDCDisplayName = previousDisplayName
	})

	env.OIDCIssuerURL = issuerURL
	env.OIDCClientID = clientID
	env.OIDCClientSecret = clientSecret
	env.OIDCDisplayName = displayName
}
