package sshtunnel

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"

	"databasus-backend/internal/util/encryption"
)

func buildAuthMethods(config Config, encryptor encryption.FieldEncryptor) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	if config.Password != "" {
		password, err := decryptIfNeeded(config.Password, encryptor)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt the SSH tunnel password: %w", err)
		}

		authMethods = append(authMethods, ssh.Password(password))
	}

	if config.PrivateKey != "" {
		signer, err := buildPrivateKeySigner(config, encryptor)
		if err != nil {
			return nil, err
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return nil, errors.New("SSH tunnel requires either a password or a private key")
	}

	return authMethods, nil
}

func buildPrivateKeySigner(config Config, encryptor encryption.FieldEncryptor) (ssh.Signer, error) {
	privateKey, err := decryptIfNeeded(config.PrivateKey, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt the SSH tunnel private key: %w", err)
	}

	if config.PrivateKeyPassphrase == "" {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse the SSH tunnel private key: %w", err)
		}

		return signer, nil
	}

	passphrase, err := decryptIfNeeded(config.PrivateKeyPassphrase, encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt the SSH tunnel private key passphrase: %w", err)
	}

	signer, err := ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the SSH tunnel private key: %w", err)
	}

	return signer, nil
}

// A restore target config is never persisted, so it arrives in plaintext with a nil encryptor.
func decryptIfNeeded(value string, encryptor encryption.FieldEncryptor) (string, error) {
	if encryptor == nil {
		return value, nil
	}

	return encryptor.Decrypt(value)
}
