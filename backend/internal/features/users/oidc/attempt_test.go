package users_oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/cache"
)

func Test_AttemptStore_IssuedAttemptIsSpentOnce(t *testing.T) {
	store := NewAttemptStore(cache.GetStore())

	state, issuedAttempt, err := store.Issue(t.Context(), testRedirectURI)
	require.NoError(t, err)
	assert.NotEmpty(t, state)
	assert.NotEmpty(t, issuedAttempt.CodeVerifier)
	assert.NotEmpty(t, issuedAttempt.Nonce)

	spentAttempt, err := store.Spend(t.Context(), state)
	require.NoError(t, err)
	require.NotNil(t, spentAttempt)
	assert.Equal(t, issuedAttempt.CodeVerifier, spentAttempt.CodeVerifier)
	assert.Equal(t, issuedAttempt.Nonce, spentAttempt.Nonce)
	assert.Equal(t, testRedirectURI, spentAttempt.RedirectURI)
}

func Test_AttemptStore_ReplayedStateFindsNothing(t *testing.T) {
	store := NewAttemptStore(cache.GetStore())

	state, _, err := store.Issue(t.Context(), testRedirectURI)
	require.NoError(t, err)

	_, err = store.Spend(t.Context(), state)
	require.NoError(t, err)

	replayedAttempt, err := store.Spend(t.Context(), state)

	require.NoError(t, err)
	assert.Nil(t, replayedAttempt)
}

func Test_AttemptStore_UnknownStateFindsNothing(t *testing.T) {
	store := NewAttemptStore(cache.GetStore())

	unknownAttempt, err := store.Spend(t.Context(), "a-state-this-instance-never-issued")

	require.NoError(t, err)
	assert.Nil(t, unknownAttempt)
}

func Test_AttemptStore_EachAttemptGetsItsOwnSecrets(t *testing.T) {
	store := NewAttemptStore(cache.GetStore())

	firstState, firstAttempt, err := store.Issue(t.Context(), testRedirectURI)
	require.NoError(t, err)

	secondState, secondAttempt, err := store.Issue(t.Context(), testRedirectURI)
	require.NoError(t, err)

	assert.NotEqual(t, firstState, secondState)
	assert.NotEqual(t, firstAttempt.CodeVerifier, secondAttempt.CodeVerifier)
	assert.NotEqual(t, firstAttempt.Nonce, secondAttempt.Nonce)
}

func Test_GetCodeChallenge_IsStableAndHidesTheVerifier(t *testing.T) {
	challenge := GetCodeChallenge("a-verifier")

	assert.Equal(t, challenge, GetCodeChallenge("a-verifier"))
	assert.NotEqual(t, challenge, GetCodeChallenge("another-verifier"))
	assert.NotContains(t, challenge, "a-verifier")
}
