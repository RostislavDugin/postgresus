package databases

import (
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	"databasus-backend/internal/features/databases/databases/postgresql"
	"databasus-backend/internal/storage"
	"errors"

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
			database.Postgresql.DatabaseID = &database.ID
		case DatabaseTypeMysql:
			if database.Mysql == nil {
				return errors.New("mysql configuration is required for MySQL database")
			}
			database.Mysql.DatabaseID = &database.ID
		case DatabaseTypeMariadb:
			if database.Mariadb == nil {
				return errors.New("mariadb configuration is required for MariaDB database")
			}
			database.Mariadb.DatabaseID = &database.ID
		case DatabaseTypeMongodb:
			if database.Mongodb == nil {
				return errors.New("mongodb configuration is required for MongoDB database")
			}
			database.Mongodb.DatabaseID = &database.ID
		}

		if isNew {
			if err := tx.Create(database).
				Omit("Postgresql", "Mysql", "Mariadb", "Mongodb", "Notifiers").
				Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(database).
				Omit("Postgresql", "Mysql", "Mariadb", "Mongodb", "Notifiers").
				Error; err != nil {
				return err
			}
		}

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
		case DatabaseTypeMysql:
			database.Mysql.DatabaseID = &database.ID
			if database.Mysql.ID == uuid.Nil {
				database.Mysql.ID = uuid.New()
				if err := tx.Create(database.Mysql).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Mysql).Error; err != nil {
					return err
				}
			}
		case DatabaseTypeMariadb:
			database.Mariadb.DatabaseID = &database.ID
			if database.Mariadb.ID == uuid.Nil {
				database.Mariadb.ID = uuid.New()
				if err := tx.Create(database.Mariadb).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Mariadb).Error; err != nil {
					return err
				}
			}
		case DatabaseTypeMongodb:
			database.Mongodb.DatabaseID = &database.ID
			if database.Mongodb.ID == uuid.Nil {
				database.Mongodb.ID = uuid.New()
				if err := tx.Create(database.Mongodb).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(database.Mongodb).Error; err != nil {
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
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
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
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
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
		case DatabaseTypeMysql:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&mysql.MysqlDatabase{}).Error; err != nil {
				return err
			}
		case DatabaseTypeMariadb:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&mariadb.MariadbDatabase{}).Error; err != nil {
				return err
			}
		case DatabaseTypeMongodb:
			if err := tx.
				Where("database_id = ?", id).
				Delete(&mongodb.MongodbDatabase{}).Error; err != nil {
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
		Preload("Mysql").
		Preload("Mariadb").
		Preload("Mongodb").
		Preload("Notifiers").
		Find(&databases).Error; err != nil {
		return nil, err
	}

	return databases, nil
}

func (r *DatabaseRepository) GetDatabasesIDsByNotifierID(
	notifierID uuid.UUID,
) ([]uuid.UUID, error) {
	var databasesIDs []uuid.UUID

	if err := storage.
		GetDb().
		Table("database_notifiers").
		Where("notifier_id = ?", notifierID).
		Pluck("database_id", &databasesIDs).Error; err != nil {
		return nil, err
	}

	return databasesIDs, nil
}
