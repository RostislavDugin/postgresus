package databases

import (
	"postgresus-backend/internal/features/databases/databases/postgresql"
	"postgresus-backend/internal/features/notifiers"
	"postgresus-backend/internal/features/storages"
	"postgresus-backend/internal/util/tools"

	"github.com/google/uuid"
)

func CreateTestDatabase(
	workspaceID uuid.UUID,
	storage *storages.Storage,
	notifier *notifiers.Notifier,
) *Database {
	database := &Database{
		WorkspaceID: &workspaceID,
		Name:        "test " + uuid.New().String(),
		Type:        DatabaseTypePostgres,

		Postgresql: &postgresql.PostgresqlDatabase{
			Version:  tools.PostgresqlVersion16,
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
		},

		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

func RemoveTestDatabase(database *Database) {
	err := databaseRepository.Delete(database.ID)
	if err != nil {
		panic(err)
	}
}
