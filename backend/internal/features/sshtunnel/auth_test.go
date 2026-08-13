package sshtunnel

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func generatePrivateKey(t *testing.T) string {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	require.NoError(t, err)

	return string(pem.EncodeToMemory(pemBlock))
}

// The unusable private key is the point: a builder that still branched on field presence would try
// to parse it and fail, so this pins the auth type as the only thing that decides.
func Test_BuildAuthMethods_WhenAuthTypeIsPassword_ReturnsOnlyThePasswordMethod(t *testing.T) {
	config := enabledConfig()
	config.PrivateKey = "not a key"

	authMethods, err := buildAuthMethods(config, nil)

	require.NoError(t, err)
	require.Len(t, authMethods, 1)
	assert.IsType(t, ssh.Password(""), authMethods[0])
}

func Test_BuildAuthMethods_WhenAuthTypeIsPrivateKey_ReturnsOnlyThePublicKeyMethod(t *testing.T) {
	config := enabledPrivateKeyConfig()
	config.PrivateKey = generatePrivateKey(t)
	config.PrivateKeyPassphrase = ""
	config.Password = "tunnelpassword"

	authMethods, err := buildAuthMethods(config, nil)

	require.NoError(t, err)
	require.Len(t, authMethods, 1)
	assert.IsType(t, ssh.PublicKeys(), authMethods[0])
}

func Test_BuildAuthMethods_WhenAuthTypeIsUnknown_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.AuthType = "CERTIFICATE"

	_, err := buildAuthMethods(config, nil)

	assert.Error(t, err)
}
