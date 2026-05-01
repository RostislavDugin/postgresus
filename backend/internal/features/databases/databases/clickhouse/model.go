package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

var ErrNotImplemented = errors.New("clickhouse support is not yet implemented")

type ClickhouseDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version tools.ClickhouseVersion `json:"version" gorm:"type:text;not null"`

	Host     string `json:"host"     gorm:"type:text;not null"`
	Port     int    `json:"port"     gorm:"type:int;not null"`
	Username string `json:"username" gorm:"type:text;not null"`
	Password string `json:"password" gorm:"type:text"`
	Database string `json:"database" gorm:"column:database;type:text;not null"`
	IsHttps     bool `json:"isHttps"     gorm:"type:boolean;not null;default:false"`
	IsStrictTls bool `json:"isStrictTls" gorm:"type:boolean;not null;default:false"`

	IsDropExisting      bool `json:"isDropExisting"      gorm:"-"`
	IsKeepReplicatedDDL bool `json:"isKeepReplicatedDDL" gorm:"-"`
}

func (c *ClickhouseDatabase) TableName() string {
	return "clickhouse_databases"
}

func (c *ClickhouseDatabase) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}

	if c.Port == 0 {
		return errors.New("port is required")
	}

	if c.Username == "" {
		return errors.New("username is required")
	}

	if c.Password == "" {
		return errors.New("password is required")
	}

	if c.Database == "" {
		return errors.New("database is required")
	}

	return nil
}

func (c *ClickhouseDatabase) ValidateUpdate(_ *ClickhouseDatabase) error {
	return nil
}

func (c *ClickhouseDatabase) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(c.Password, encryptor, databaseID)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	conn, err := OpenConn(ctx, c, password)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close clickhouse connection", "error", closeErr)
		}
	}()

	detectedVersion, err := DetectClickhouseVersion(ctx, conn)
	if err != nil {
		return err
	}
	c.Version = detectedVersion

	var exists uint64
	if err := conn.QueryRow(ctx,
		"SELECT count() FROM system.databases WHERE name = ?",
		c.Database,
	).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("database %q does not exist on server", c.Database)
	}

	return nil
}

func (c *ClickhouseDatabase) HideSensitiveData() {
	if c == nil {
		return
	}

	c.Password = ""
}

func (c *ClickhouseDatabase) Update(incoming *ClickhouseDatabase) {
	c.Version = incoming.Version
	c.Host = incoming.Host
	c.Port = incoming.Port
	c.Username = incoming.Username
	c.Database = incoming.Database
	c.IsHttps = incoming.IsHttps
	c.IsStrictTls = incoming.IsStrictTls
	c.IsDropExisting = incoming.IsDropExisting
	c.IsKeepReplicatedDDL = incoming.IsKeepReplicatedDDL

	if incoming.Password != "" {
		c.Password = incoming.Password
	}
}

func (c *ClickhouseDatabase) EncryptSensitiveFields(
	databaseID uuid.UUID,
	encryptor encryption.FieldEncryptor,
) error {
	if c.Password != "" {
		encrypted, err := encryptor.Encrypt(databaseID, c.Password)
		if err != nil {
			return err
		}

		c.Password = encrypted
	}

	return nil
}

func (c *ClickhouseDatabase) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	return c.PopulateVersion(logger, encryptor, databaseID)
}

func (c *ClickhouseDatabase) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
	databaseID uuid.UUID,
) error {
	if c.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	password, err := decryptPasswordIfNeeded(c.Password, encryptor, databaseID)
	if err != nil {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	conn, err := OpenConn(ctx, c, password)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			logger.Error("failed to close clickhouse connection", "error", closeErr)
		}
	}()

	detected, err := DetectClickhouseVersion(ctx, conn)
	if err != nil {
		return err
	}

	c.Version = detected

	return nil
}

