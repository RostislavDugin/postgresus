package sshtunnel

import (
	"fmt"

	"golang.org/x/crypto/ssh"

	"databasus-backend/internal/util/encryption"
)

func buildAuthMethods(config Config, encryptor encryption.FieldEncryptor) ([]ssh.AuthMethod, error) {
	switch config.AuthType {
	case AuthTypePassword:
		password, err := decryptIfNeeded(config.Password, encryptor)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt the SSH tunnel password: %w", err)
		}

		return []ssh.AuthMethod{ssh.Password(password)}, nil
	case AuthTypePrivateKey:
		signer, err := buildPrivateKeySigner(config, encryptor)
		if err != nil {
			return nil, err
		}

		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}

	return nil, fmt.Errorf("invalid SSH tunnel auth type: %s", config.AuthType)
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
