package api_keys

const (
	// TokenPrefix prefixes every raw API key token so it is recognizable in logs/UI and lookups.
	TokenPrefix = "dbs_"

	// PrincipalContextKey holds the authenticated *Principal in the gin context for public routes.
	PrincipalContextKey = "api_key_principal"

	// tokenPrefixLength is how many leading characters of the raw token are stored
	// (the "dbs_" prefix plus 8 base64url chars) for display/disambiguation in the UI.
	tokenPrefixLength = 12
)
