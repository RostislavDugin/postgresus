package users_oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"databasus-backend/internal/util/cache"
)

const attemptNamespace = "oidc_sign_in_attempt:"

// An attempt covers one round trip through the provider's sign-in pages, which
// may include entering a second factor, so the window is wider than a single
// redirect but still far shorter than a session.
const attemptLifetime = 10 * time.Minute

// Attempt holds the per-sign-in secrets the instance keeps to itself. The
// browser receives only the state that keys it; the verifier and the nonce
// never leave the backend, which is what binds a returning code to the attempt
// this instance started.
type Attempt struct {
	CodeVerifier string `json:"codeVerifier"`
	Nonce        string `json:"nonce"`
	RedirectURI  string `json:"redirectUri"`
}

type AttemptStore struct {
	attempts *cache.JSONStore[Attempt]
}

func NewAttemptStore(cacheStore cache.Store) *AttemptStore {
	return &AttemptStore{
		cache.NewJSONStoreWithLifetime[Attempt](cacheStore, attemptNamespace, attemptLifetime),
	}
}

func (s *AttemptStore) Issue(ctx context.Context, redirectURI string) (string, Attempt, error) {
	attempt := Attempt{
		CodeVerifier: generateSecureValue(),
		Nonce:        generateSecureValue(),
		RedirectURI:  redirectURI,
	}

	state := generateSecureValue()

	if err := s.attempts.SetWithLifetime(ctx, cache.ExpiringValue[Attempt]{
		Key:      state,
		Value:    attempt,
		Lifetime: attemptLifetime,
	}); err != nil {
		return "", Attempt{}, err
	}

	return state, attempt, nil
}

// Spend reads the attempt and removes it in one step, so a replayed state finds
// nothing. It returns nil when the state is unknown, expired or already spent.
func (s *AttemptStore) Spend(ctx context.Context, state string) (*Attempt, error) {
	return s.attempts.ReadAndDelete(ctx, state)
}

func GetCodeChallenge(codeVerifier string) string {
	digest := sha256.Sum256([]byte(codeVerifier))

	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func generateSecureValue() string {
	value := make([]byte, 32)

	if _, err := rand.Read(value); err != nil {
		panic("failed to generate secure random value: " + err.Error())
	}

	return base64.RawURLEncoding.EncodeToString(value)
}
