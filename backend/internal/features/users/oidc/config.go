package users_oidc

import (
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"databasus-backend/internal/config"
)

const defaultDisplayName = "Single sign-on"

type ProviderConfig struct {
	IssuerURL    string
	ClientID     string
	clientSecret string
	displayName  string

	// An empty set lets every person the provider authenticates sign in; a
	// non-empty one admits only those carrying at least one of its groups.
	allowedGroups []string
}

func GetProviderConfig() ProviderConfig {
	env := config.GetEnv()

	return ProviderConfig{
		IssuerURL:     strings.TrimSpace(env.OIDCIssuerURL),
		ClientID:      strings.TrimSpace(env.OIDCClientID),
		clientSecret:  strings.TrimSpace(env.OIDCClientSecret),
		displayName:   strings.TrimSpace(env.OIDCDisplayName),
		allowedGroups: parseAllowedGroups(env.OIDCAllowedGroups),
	}
}

// Keycloak writes a group as the path it has in the tree, so the same group an
// operator calls "csm-databasus-user" arrives as "/csm-databasus-user". Both
// spellings mean the group, so both are accepted here and in the claim.
func parseAllowedGroups(configured string) []string {
	var allowedGroups []string

	for group := range strings.SplitSeq(configured, ",") {
		if normalized := NormalizeGroup(group); normalized != "" {
			allowedGroups = append(allowedGroups, normalized)
		}
	}

	return allowedGroups
}

func NormalizeGroup(group string) string {
	return strings.TrimPrefix(strings.TrimSpace(group), "/")
}

func (c ProviderConfig) IsConfigured() bool {
	return c.IssuerURL != "" && c.ClientID != "" && c.clientSecret != ""
}

func (c ProviderConfig) GetAllowedGroups() []string {
	return c.allowedGroups
}

func (c ProviderConfig) HasAllowedGroups() bool {
	return len(c.allowedGroups) > 0
}

func (c ProviderConfig) GetDisplayName() string {
	if c.displayName == "" {
		return defaultDisplayName
	}

	return c.displayName
}

// An instance that names no group admits everyone the provider authenticates;
// one that names any admits only a member of at least one of them.
func (c ProviderConfig) isGroupAllowed(claimedGroups []string) bool {
	if !c.HasAllowedGroups() {
		return true
	}

	for _, claimedGroup := range claimedGroups {
		if slices.Contains(c.allowedGroups, NormalizeGroup(claimedGroup)) {
			return true
		}
	}

	return false
}

// The groups claim is only asked for when it decides something, so an instance
// without an allowed group keeps working against a provider that has no such
// scope and would reject the request for it.
func (c ProviderConfig) getScopes() []string {
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}

	if c.HasAllowedGroups() {
		scopes = append(scopes, "groups")
	}

	return scopes
}

// The secret stays unexported so it cannot reach a response or a log line by
// being carried along with the rest of the configuration.
func (c ProviderConfig) getClientSecret() string {
	return c.clientSecret
}
