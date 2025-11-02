package databases

import (
	"errors"
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DatabaseRepository struct{}

func (r *DatabaseRepository) Save(database *Database) (*Database, error) {
	db := storage.GetDb()

	isNew := database.ID == uuid.Nil
	if isNew {
		database.ID = uuid.New()
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		switch database.Type {
		case DatabaseTypePostgres:
			if database.Postgresql == nil {
				return errors.New("postgresql configuration is required for PostgreSQL database")
			}

			// Ensure DatabaseID is always set and never nil
			database.Postgresql.DatabaseID = &database.ID
		}

		if isNew {
			if err := tx.Create(database).
				Omit("Postgresql", "Notifiers").
				Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(database).
				Omit("Postgresql", "Notifiers").
				Error; err != nil {
				return err
			}
		}

		// Save the specific database type
		switch database.Type {
		case DatabaseTypePostgres:
			database.Postgresql.DatabaseID = &database.ID
			if database.Postgresql.ID == uuid.Nil {
				database.Postgresql.ID = uuid.New()
				if err := tx.Create(database.Postgresql).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Postgresql).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.
			Model(database).
			Association("Notifiers").
			Replace(database.Notifiers); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return database, nil
}

func (r *DatabaseRepository) FindByID(id uuid.UUID) (*Database, error) {
	var database Database

	if err := storage.
		GetDb().
		Preload("Postgresql").
		Preload("Notifiers").
		Where("id = ?", id).
		First(&database).Error; err != nil {
		return nil, err
	}

	return &database, nil
}

func (r *DatabaseRepository) FindByWorkspaceID(workspaceID uuid.UUID) ([]*Database, error) {
	var databases []*Database

	if err := storage.
		GetDb().
		Preload("Postgresql").
		Preload("Notifiers").
		Where("workspace_id = ?", workspaceID).
		Order("CASE WHEN health_status = 'UNAVAILABLE' THEN 1 WHEN health_status = 'AVAILABLE' THEN 2 WHEN health_status IS NULL THEN 3 ELSE 4 END, name ASC").
		Find(&databases).Error; err != nil {
		return nil, err
	}

	return databases, nil
}

func (r *DatabaseRepository) Delete(id uuid.UUID) error {
	db := storage.GetDb()

	return db.Transaction(func(tx *gorm.DB) error {
		var database Database
		if err := tx.Where("id = ?", id).First(&database).Error; err != nil {
			return err
		}

		if err := tx.Model(&database).Association("Notifiers").Clear(); err != nil {
			return err
		}

		switch database.Type {
		case DatabaseTypePostgres:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&postgresql.PostgresqlDatabase{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Delete(&Database{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *DatabaseRepository) IsNotifierUsing(notifierID uuid.UUID) (bool, error) {
	var count int64

	if err := storage.
		GetDb().
		Table("database_notifiers").
		Where("notifier_id = ?", notifierID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *DatabaseRepository) GetAllDatabases() ([]*Database, error) {
	var databases []*Database

	if err := storage.
		GetDb().
		Preload("Postgresql").
		Preload("Notifiers").
		Find(&databases).Error; err != nil {
		return nil, err
	}

	return databases, nil
}
