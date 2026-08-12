package sshtunnel

import (
	"errors"

	"databasus-backend/internal/util/encryption"
)

// Consumers must embed this with gorm:"embedded;embeddedPrefix:ssh_" so each engine table owns its
// own bastion columns; the column names below assume that prefix.
type Config struct {
	IsEnabled            bool   `json:"isEnabled"            gorm:"column:is_enabled;type:boolean;not null;default:false"`
	Host                 string `json:"host"                 gorm:"column:host;type:text;not null;default:''"`
	Port                 int    `json:"port"                 gorm:"column:port;type:integer;not null;default:22"`
	Username             string `json:"username"             gorm:"column:username;type:text;not null;default:''"`
	Password             string `json:"password"             gorm:"column:password;type:text;not null;default:''"`
	PrivateKey           string `json:"privateKey"           gorm:"column:private_key;type:text;not null;default:''"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase" gorm:"column:private_key_passphrase;type:text;not null;default:''"`
}

func (c *Config) Validate() error {
	if c == nil || !c.IsEnabled {
		return nil
	}

	if c.Host == "" {
		return errors.New("SSH tunnel host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("SSH tunnel port must be between 1 and 65535")
	}

	if c.Username == "" {
		return errors.New("SSH tunnel username is required")
	}

	if c.Password == "" && c.PrivateKey == "" {
		return errors.New("SSH tunnel requires either a password or a private key")
	}

	return nil
}

func (c *Config) HideSensitiveData() {
	if c == nil {
		return
	}

	c.Password = ""
	c.PrivateKey = ""
	c.PrivateKeyPassphrase = ""
}

func (c *Config) Update(incomingConfig *Config) {
	if c == nil || incomingConfig == nil {
		return
	}

	c.IsEnabled = incomingConfig.IsEnabled
	c.Host = incomingConfig.Host
	c.Port = incomingConfig.Port
	c.Username = incomingConfig.Username

	// A blank secret means "keep the stored one" - the edit form never receives
	// them back from the API, so overwriting on blank would wipe them.
	if incomingConfig.Password != "" {
		c.Password = incomingConfig.Password
	}

	if incomingConfig.PrivateKey != "" {
		c.PrivateKey = incomingConfig.PrivateKey
	}

	if incomingConfig.PrivateKeyPassphrase != "" {
		c.PrivateKeyPassphrase = incomingConfig.PrivateKeyPassphrase
	}
}

func (c *Config) EncryptSensitiveFields(encryptor encryption.FieldEncryptor) error {
	if c == nil {
		return nil
	}

	for _, field := range []*string{
		&c.Password,
		&c.PrivateKey,
		&c.PrivateKeyPassphrase,
	} {
		if *field == "" {
			continue
		}

		encrypted, err := encryptor.Encrypt(*field)
		if err != nil {
			return err
		}

		*field = encrypted
	}

	return nil
}
