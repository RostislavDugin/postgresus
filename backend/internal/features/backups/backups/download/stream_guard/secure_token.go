package stream_guard

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateSecureToken returns a URL-safe 256-bit random token used as the secret
// for download and restore streams. A read failure from the OS CSPRNG is
// unrecoverable, so it panics rather than hand back a weak token.
func GenerateSecureToken() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic("failed to generate secure random token: " + err.Error())
	}

	return base64.URLEncoding.EncodeToString(b)
}
