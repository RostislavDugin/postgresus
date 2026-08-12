package sshtunnel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The real encryptor resolves its key through the secret-key service and a database; these tests
// only cover which fields get walked and which are skipped. It prefixes unconditionally, including
// the empty string, so that dropping the skip-empty guard in EncryptSensitiveFields is visible.
type prefixingEncryptor struct{}

func (prefixingEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (prefixingEncryptor) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

func enabledConfig() Config {
	return Config{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunneluser",
		Password:  "tunnelpassword",
	}
}

func Test_Validate_WhenTunnelIsDisabled_IgnoresEmptyFields(t *testing.T) {
	config := Config{}

	assert.NoError(t, config.Validate())
}

func Test_Validate_WhenTunnelIsEnabledAndComplete_ReturnsNoError(t *testing.T) {
	config := enabledConfig()

	assert.NoError(t, config.Validate())
}

func Test_Validate_WhenHostIsMissing_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.Host = ""

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenUsernameIsMissing_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.Username = ""

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenPortIsOutOfRange_ReturnsError(t *testing.T) {
	for _, port := range []int{0, -1, 65536} {
		config := enabledConfig()
		config.Port = port

		assert.Error(t, config.Validate(), "port %d must be rejected", port)
	}
}

func Test_Validate_WhenNeitherPasswordNorPrivateKeyIsSet_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.Password = ""
	config.PrivateKey = ""

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenOnlyPrivateKeyIsSet_ReturnsNoError(t *testing.T) {
	config := enabledConfig()
	config.Password = ""
	config.PrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----"

	assert.NoError(t, config.Validate())
}

func Test_HideSensitiveData_WhenCalled_ClearsSecretsAndKeepsTheRest(t *testing.T) {
	config := enabledConfig()
	config.PrivateKey = "private-key"
	config.PrivateKeyPassphrase = "passphrase"

	config.HideSensitiveData()

	assert.Empty(t, config.Password)
	assert.Empty(t, config.PrivateKey)
	assert.Empty(t, config.PrivateKeyPassphrase)
	assert.Equal(t, "bastion.example.com", config.Host)
	assert.Equal(t, "tunneluser", config.Username)
	assert.True(t, config.IsEnabled)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var config *Config

	config.HideSensitiveData()
}

// The edit form never receives the stored secrets back, so it submits them blank. Overwriting on
// blank would wipe the bastion credentials on every unrelated edit.
func Test_Update_WithBlankSecrets_KeepsTheStoredOnes(t *testing.T) {
	storedConfig := enabledConfig()
	storedConfig.PrivateKey = "stored-private-key"
	storedConfig.PrivateKeyPassphrase = "stored-passphrase"

	storedConfig.Update(&Config{
		IsEnabled: true,
		Host:      "new-bastion.example.com",
		Port:      2222,
		Username:  "newuser",
	})

	assert.Equal(t, "tunnelpassword", storedConfig.Password)
	assert.Equal(t, "stored-private-key", storedConfig.PrivateKey)
	assert.Equal(t, "stored-passphrase", storedConfig.PrivateKeyPassphrase)
	assert.Equal(t, "new-bastion.example.com", storedConfig.Host)
	assert.Equal(t, 2222, storedConfig.Port)
	assert.Equal(t, "newuser", storedConfig.Username)
}

func Test_Update_WithNewSecrets_ReplacesTheStoredOnes(t *testing.T) {
	storedConfig := enabledConfig()
	storedConfig.PrivateKey = "stored-private-key"

	incomingConfig := enabledConfig()
	incomingConfig.Password = "new-password"
	incomingConfig.PrivateKey = "new-private-key"
	incomingConfig.PrivateKeyPassphrase = "new-passphrase"

	storedConfig.Update(&incomingConfig)

	assert.Equal(t, "new-password", storedConfig.Password)
	assert.Equal(t, "new-private-key", storedConfig.PrivateKey)
	assert.Equal(t, "new-passphrase", storedConfig.PrivateKeyPassphrase)
}

func Test_Update_WhenTunnelIsDisabledInIncoming_TurnsItOff(t *testing.T) {
	storedConfig := enabledConfig()

	storedConfig.Update(
		&Config{IsEnabled: false, Host: storedConfig.Host, Port: storedConfig.Port, Username: storedConfig.Username},
	)

	assert.False(t, storedConfig.IsEnabled)
	assert.Equal(t, "tunnelpassword", storedConfig.Password)
}

func Test_EncryptSensitiveFields_WhenCalled_EncryptsSecretsAndSkipsEmptyOnes(t *testing.T) {
	encryptor := prefixingEncryptor{}

	config := enabledConfig()
	config.PrivateKey = "private-key"

	require.NoError(t, config.EncryptSensitiveFields(encryptor))

	assert.NotEqual(t, "tunnelpassword", config.Password)
	assert.NotEqual(t, "private-key", config.PrivateKey)
	assert.Empty(t, config.PrivateKeyPassphrase)

	decryptedPassword, err := encryptor.Decrypt(config.Password)
	require.NoError(t, err)
	assert.Equal(t, "tunnelpassword", decryptedPassword)

	decryptedPrivateKey, err := encryptor.Decrypt(config.PrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "private-key", decryptedPrivateKey)
}
